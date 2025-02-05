package transport

import (
	"log"
	"net"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
)

type cancelableContext interface {
	Done() <-chan struct{}
	Cancel()
}

func ReceiveTCPLoop[T cancelableContext](ctx T, conn *net.TCPConn, filterChan *channel.FilterChannel) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		newPacket, err := packet.ReadPacket(conn)
		if err != nil {
			log.Println("error reading packet:", err)
			log.Println("stop handling connection from:", conn.RemoteAddr().String())
			return
		}

		filterChan.Forward([]*packet.Packet{newPacket})
	}
}

func SendTCPLoop[T cancelableContext](ctx T, conn *net.TCPConn, inChan <-chan [][]byte) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case packets := <-inChan:
			buffer := make([]byte, 0)
			for _, packet := range packets {
				buffer = append(buffer, packet...)
			}
			n, err := conn.Write(buffer)
			if err != nil {
				log.Println("error sending packet:", err)
				ctx.Cancel()
				return
			}
			if n != len(buffer) {
				log.Println("error writing to tcp: wrote", n, "bytes instead of", len(buffer))
			}
		}
	}
}
