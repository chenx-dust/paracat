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
	UDP_BUFFER_SIZE = 65535
	UDP_OOB_SIZE    = 64
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
		log.Fatalln("error reading packet:", err)
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
