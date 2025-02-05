package transport

import (
	"errors"
	"log"
	"net"
	"unsafe"

	"github.com/chenx-dust/paracat/packet"
	"golang.org/x/sys/unix"
)

const (
	UDP_BUFFER_SIZE      = 65535
	UDP_OOB_SIZE         = 64
	UDP_MIN_PADDING_SIZE = 1200
	UDP_MAX_PADDING_DIFF = 300
)

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

func ReceiveUDPRawPackets(conn *net.UDPConn) (packets [][]byte, udpAddr *net.UDPAddr, err error) {
	buf := make([]byte, UDP_BUFFER_SIZE)
	oob := make([]byte, UDP_OOB_SIZE)
	n, oobn, flags, udpAddr, err := conn.ReadMsgUDP(buf, oob)
	if err != nil {
		log.Println("error reading packet:", err)
		return
	}

	packetSize := uint16(n)

	if flags&unix.MSG_TRUNC != 0 {
		log.Println("packet truncated, need increase buffer size")
		err = errors.New("packet truncated")
		return
	}

	var cmsgs []unix.SocketControlMessage
	if flags&unix.MSG_OOB != 0 {
		cmsgs, err = unix.ParseSocketControlMessage(oob[:oobn])
		if err != nil {
			log.Println("error parsing socket control message:", err)
			return
		}

		for _, cmsg := range cmsgs {
			if cmsg.Header.Level == unix.IPPROTO_UDP && cmsg.Header.Type == unix.UDP_SEGMENT {
				packetSize = *(*uint16)(unsafe.Pointer(&cmsg.Data[0]))
				break
			}
		}
	}

	nowPtr := 0
	if packetSize == 0 {
		packets = [][]byte{{}}
	} else {
		packets = make([][]byte, 0, n/int(packetSize))
		for nowPtr+int(packetSize) <= n {
			packets = append(packets, buf[nowPtr:nowPtr+int(packetSize)])
			nowPtr += int(packetSize)
		}
	}
	return
}

func ReceiveUDPPackets(conn *net.UDPConn) (packets []*packet.Packet, udpAddr *net.UDPAddr, err error) {
	rawPackets, udpAddr, err := ReceiveUDPRawPackets(conn)
	if err != nil {
		return
	}

	packets = make([]*packet.Packet, 0, len(rawPackets))
	for _, rawPacket := range rawPackets {
		newPacket, err := packet.Unpack(rawPacket)
		if err != nil {
			log.Println("error unpacking packet:", err)
			continue
		}
		packets = append(packets, newPacket)
	}
	return
}

func SendUDPPackets(conn *net.UDPConn, dstAddr *net.UDPAddr, packets [][]byte) error {
	useGSO := false
	gsoSize := 0
	minSize := len(packets[0])
	maxSize := len(packets[0])
	for _, packet := range packets {
		if len(packet) < minSize {
			minSize = len(packet)
		}
		if len(packet) > maxSize {
			maxSize = len(packet)
		}
	}
	useGSO = maxSize-minSize <= UDP_MAX_PADDING_DIFF && minSize >= UDP_MIN_PADDING_SIZE
	gsoSize = maxSize
	oob := make([]byte, unix.CmsgSpace(2))
	cmsg_hdr := (*unix.Cmsghdr)(unsafe.Pointer(&oob[0]))
	cmsg_hdr.Level = unix.IPPROTO_UDP
	cmsg_hdr.Type = unix.UDP_SEGMENT
	cmsg_hdr.SetLen(unix.CmsgLen(2))
	if useGSO {
		buffer := make([]byte, gsoSize*len(packets))
		for i, packet := range packets {
			copy(buffer[i*gsoSize:(i+1)*gsoSize], packet)
		}
		*(*uint16)(unsafe.Pointer(&oob[unix.CmsgSpace(0)])) = uint16(gsoSize)
		n, oobn, err := conn.WriteMsgUDP(buffer, oob, dstAddr)
		if err != nil {
			log.Println("error sending packet with GSO:", err)
			return err
		}
		if n != len(buffer) {
			log.Println("error writing to udp: wrote", n, "bytes instead of", len(buffer))
			return errors.New("error writing to udp")
		}
		if oobn != len(oob) {
			log.Println("error writing oob to udp: wrote", oobn, "bytes instead of", len(oob))
		}
	} else {
		for _, packet := range packets {
			*(*uint16)(unsafe.Pointer(&oob[unix.CmsgSpace(0)])) = uint16(len(packet))
			n, _, err := conn.WriteMsgUDP(packet, oob, dstAddr)
			if err != nil {
				log.Println("error sending packet:", err)
				return err
			}
			if n != len(packet) {
				log.Println("error writing to udp: wrote", n, "bytes instead of", len(packet))
			}
		}
	}
	return nil
}

func SendUDPLoop[T cancelableContext](ctx T, conn *net.UDPConn, dstAddr *net.UDPAddr, inChan <-chan [][]byte) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case packets := <-inChan:
			err := SendUDPPackets(conn, dstAddr, packets)
			if err != nil {
				log.Println("error sending packet:", err)
				ctx.Cancel()
				return
			}
		}
	}
}
