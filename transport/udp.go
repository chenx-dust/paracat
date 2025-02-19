package transport

import (
	"errors"
	"io"
	"log"
	"net"
	"unsafe"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/packet"
	"golang.org/x/sys/unix"
)

const MAX_GSO_NUM = 64

func EnableGRO(conn *net.UDPConn) (err error) {
	sysconn, err := conn.SyscallConn()
	if err != nil {
		log.Println("error getting syscall conn:", err)
		return
	}

	err = sysconn.Control(func(fd uintptr) {
		unix.SetsockoptInt(int(fd), unix.IPPROTO_UDP, unix.UDP_GRO, 1)
	})
	if err != nil {
		log.Println("error enabling GRO:", err)
		return
	}
	return
}

func EnableGSO(conn *net.UDPConn) (err error) {
	sysconn, err := conn.SyscallConn()
	if err != nil {
		log.Println("error getting syscall conn:", err)
		return
	}

	err = sysconn.Control(func(fd uintptr) {
		unix.SetsockoptInt(int(fd), unix.IPPROTO_UDP, unix.UDP_SEGMENT, 1)
	})
	if err != nil {
		log.Println("error enabling GSO:", err)
		return
	}
	return
}

func ReceiveUDPRawPackets(conn *net.UDPConn) (buffer.OwnedPtr[*buffer.PackedBuffer], *net.UDPAddr, error) {
	packedBuffer := buffer.NewPackedBuffer()
	oob := make([]byte, buffer.OOB_SIZE)
	n, oobn, flags, udpAddr, err := conn.ReadMsgUDP(packedBuffer.Ptr.Buffer[:], oob)
	if err != nil {
		log.Println("error reading packet:", err)
		return packedBuffer.Move(), nil, err
	}

	var packetSize uint16

	if flags&unix.MSG_TRUNC != 0 {
		log.Println("packet truncated, need increase buffer size")
		err = errors.New("packet truncated")
		return packedBuffer.Move(), nil, err
	}

	var cmsgs []unix.SocketControlMessage
	if flags&unix.MSG_OOB != 0 {
		cmsgs, err = unix.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			log.Println("error parsing socket control message:", err)
			return packedBuffer.Move(), nil, err
		}

		for _, cmsg := range cmsgs {
			if cmsg.Header.Level == unix.IPPROTO_UDP && cmsg.Header.Type == unix.UDP_SEGMENT {
				packetSize = *(*uint16)(unsafe.Pointer(&cmsg.Data[0]))
				break
			}
		}
	} else {
		packetSize = uint16(n)
	}

	nowPtr := 0
	if n > 0 {
		for nowPtr+int(packetSize) <= n {
			packedBuffer.Ptr.SubPackets = append(packedBuffer.Ptr.SubPackets, int(packetSize))
			nowPtr += int(packetSize)
		}
	} else {
		packedBuffer.Ptr.SubPackets = append(packedBuffer.Ptr.SubPackets, 0)
	}
	packedBuffer.Ptr.TotalSize = nowPtr
	return packedBuffer.Move(), udpAddr, nil
}

func ReceiveUDPPackets(conn *net.UDPConn) (buffer.WithBuffer[[]*packet.Packet], *net.UDPAddr, error) {
	rawPackets, udpAddr, err := ReceiveUDPRawPackets(conn)
	if err != nil {
		return buffer.WithBuffer[[]*packet.Packet]{Buffer: rawPackets.Move()}, nil, err
	}

	packets := make([]*packet.Packet, 0, len(rawPackets.Ptr.SubPackets))
	nowPtr := 0
	for _, slice := range rawPackets.Ptr.SubPackets {
		newPacket, parsed, err := packet.Unpack(rawPackets.Ptr.Buffer[nowPtr : nowPtr+slice])
		if err != nil {
			log.Println("error unpacking packet:", err)
			continue
		}
		if parsed != slice {
			log.Println("warning: unpacking packet parsed", parsed, "expected", slice)
		}
		packets = append(packets, newPacket)
		nowPtr += slice
	}
	return buffer.WithBuffer[[]*packet.Packet]{
		Thing:  packets,
		Buffer: rawPackets.Move(),
	}, udpAddr, nil
}

func SendUDPPackets(conn *net.UDPConn, dstAddr *net.UDPAddr, pBuffer_ buffer.BorrowedArgPtr[*buffer.PackedBuffer], enableGSO bool) error {
	pBuffer := pBuffer_.ToBorrowed()
	gsoSize := 0
	minSize := 0
	maxSize := 0
	var oob []byte
	if enableGSO {
		if len(pBuffer.Ptr.SubPackets) > 0 {
			minSize = pBuffer.Ptr.SubPackets[0]
			maxSize = pBuffer.Ptr.SubPackets[0]
			for _, slice := range pBuffer.Ptr.SubPackets {
				if slice < minSize {
					minSize = slice
				}
				if slice > maxSize {
					maxSize = slice
				}
			}
			gsoSize = maxSize
		}
		oob = make([]byte, unix.CmsgSpace(2))
		cmsgHdr := (*unix.Cmsghdr)(unsafe.Pointer(&oob[0]))
		cmsgHdr.Level = unix.IPPROTO_UDP
		cmsgHdr.Type = unix.UDP_SEGMENT
		cmsgHdr.SetLen(unix.CmsgLen(2))
	} else {
		oob = make([]byte, 0)
	}
	if enableGSO && maxSize == minSize && gsoSize > 0 && len(pBuffer.Ptr.SubPackets) <= 64 {
		*(*uint16)(unsafe.Pointer(&oob[unix.CmsgSpace(0)])) = uint16(gsoSize)
		n, oobn, err := conn.WriteMsgUDP(pBuffer.Ptr.Buffer[:pBuffer.Ptr.TotalSize], oob, dstAddr)
		if err != nil {
			log.Println("error sending packet with GSO:", err)
			return err
		}
		if n != pBuffer.Ptr.TotalSize {
			log.Println("error writing to udp: wrote", n, "bytes instead of", pBuffer.Ptr.TotalSize)
			return errors.New("error writing to udp")
		}
		if oobn != len(oob) {
			log.Println("error writing oob to udp: wrote", oobn, "bytes instead of", len(oob))
		}
	} else {
		nowPtr := 0
		for _, slice := range pBuffer.Ptr.SubPackets {
			if enableGSO {
				*(*uint16)(unsafe.Pointer(&oob[unix.CmsgSpace(0)])) = uint16(slice)
			}
			n, _, err := conn.WriteMsgUDP(pBuffer.Ptr.Buffer[nowPtr:nowPtr+slice], oob, dstAddr)
			if err != nil {
				log.Println("error sending packet:", err)
				return err
			}
			if n != slice {
				log.Println("error writing to udp: wrote", n, "bytes instead of", slice)
			}
			nowPtr += slice
		}
	}
	return nil
}

func SendUDPLoop[T cancelableContext](ctx T, conn *net.UDPConn, dstAddr *net.UDPAddr, inChan <-chan buffer.ArgPtr[*buffer.PackedBuffer], enableGSO bool) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case pBuffer_ := <-inChan:
			pBuffer := pBuffer_.ToOwned()
			err := SendUDPPackets(conn, dstAddr, pBuffer.BorrowArg(), enableGSO)
			pBuffer.Release()
			if err != nil {
				log.Println("error sending packet:", err)
				if err == io.EOF {
					ctx.Cancel()
					return
				}
			}
		}
	}
}
