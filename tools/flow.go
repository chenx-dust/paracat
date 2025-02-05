// A utility to test the throughput of UDP packets.
package main

import (
	"flag"
	"fmt"
	"net"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

func main() {
	target := flag.String("target", "", "Target address with port")
	packetSize := flag.Int("size", 1472, "UDP packet size (bytes)")
	batchSize := flag.Int("batch", 44, "Batch send size (packets)")
	listen := flag.Int("listen", 0, "Listen port")
	reverse := flag.Bool("reverse", false, "Reverse mode")
	bidirectional := flag.Bool("bidirectional", false, "Bidirectional mode")
	flag.Parse()

	if *target == "" && *listen == 0 {
		flag.Usage()
		return
	}

	var sendConn *net.UDPConn
	if *target != "" {
		var err error
		sendConn, err = net.ListenUDP("udp", nil)
		fmt.Println("Sender local address: ", sendConn.LocalAddr())
		if err != nil {
			panic(err)
		}
		if *reverse || *bidirectional {
			trySend(sendConn, *target)
			go receive(sendConn, *packetSize, *batchSize)
		}
		if !*reverse || *bidirectional {
			go send(sendConn, *target, *packetSize, *batchSize)
		}
	}

	var listenConn *net.UDPConn
	if *listen != 0 {
		var err error
		listenAddr := &net.UDPAddr{Port: *listen}
		listenConn, err = net.ListenUDP("udp", listenAddr)
		fmt.Println("Listening on: ", listenConn.LocalAddr())
		if err != nil {
			panic(err)
		}
		if *reverse || *bidirectional {
			addr := tryReceive(listenConn)
			fmt.Println("Reverse target address: ", addr)
			go send(listenConn, addr.String(), *packetSize, *batchSize)
		}
		if !*reverse || *bidirectional {
			go receive(listenConn, *packetSize, *batchSize)
		}
	}

	select {}
}

func send(conn *net.UDPConn, target string, packetSize int, batchSize int) {
	// 准备目标地址
	dstAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		panic(err)
	}

	sysconn, err := conn.SyscallConn()
	if err != nil {
		panic(err)
	}

	err = sysconn.Control(func(fd uintptr) {
		unix.SetsockoptInt(int(fd), syscall.IPPROTO_UDP, unix.UDP_SEGMENT, 1)
	})
	if err != nil {
		panic(err)
	}

	err = conn.SetWriteBuffer(packetSize * batchSize)
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, packetSize*batchSize)
	oob := make([]byte, unix.CmsgSpace(2))

	cmsg_hdr := (*unix.Cmsghdr)(unsafe.Pointer(&oob[0]))
	cmsg_hdr.Level = syscall.IPPROTO_UDP
	cmsg_hdr.Type = unix.UDP_SEGMENT
	cmsg_hdr.SetLen(unix.CmsgLen(2))

	*(*uint16)(unsafe.Pointer(&oob[unix.CmsgSpace(0)])) = uint16(packetSize)

	for {
		_, _, err := conn.WriteMsgUDP(buffer, oob, dstAddr)
		if err != nil {
			fmt.Println("Send failed: ", err)
			continue
		}
	}
}

func trySend(conn *net.UDPConn, target string) {
	dstAddr, err := net.ResolveUDPAddr("udp", target)
	if err != nil {
		panic(err)
	}

	buffer := make([]byte, 0)
	_, err = conn.WriteToUDP(buffer, dstAddr)
	if err != nil {
		panic(err)
	}
}

func receive(conn *net.UDPConn, assumePacketSize int, assumeBatchSize int) {
	port := conn.LocalAddr().(*net.UDPAddr).Port

	sysconn, err := conn.SyscallConn()
	if err != nil {
		panic(err)
	}

	err = sysconn.Control(func(fd uintptr) {
		unix.SetsockoptInt(int(fd), syscall.IPPROTO_UDP, unix.UDP_GRO, 1)
	})
	if err != nil {
		panic(err)
	}

	conn.SetReadBuffer(assumePacketSize * assumeBatchSize)

	var byteCount uint64
	buffer := make([]byte, 65535) // 64KB receive buffer

	go func() {
		for {
			n, _, err := conn.ReadFromUDP(buffer)
			if err != nil {
				fmt.Printf("Receive error: %v\n", err)
				continue
			}
			atomic.AddUint64(&byteCount, uint64(n))
		}
	}()

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		bytes := atomic.SwapUint64(&byteCount, 0)
		mbps := float64(bytes*8) / 1e6
		fmt.Printf(":%d Receive: %.2f Mbps | %.2f MB/s\n", port, mbps, float64(bytes)/1e6)
	}
}

func tryReceive(conn *net.UDPConn) (addr *net.UDPAddr) {
	buffer := make([]byte, 1500)
	_, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		panic(err)
	}
	return addr
}
