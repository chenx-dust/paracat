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

func TCPFowrardLoop[T cancelableContext](ctx T, conn *net.TCPConn, filterChan *channel.FilterChannel) {
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
