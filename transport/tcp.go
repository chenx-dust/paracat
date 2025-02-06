package transport

import (
	"io"
	"log"
	"net"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
)

type cancelableContext interface {
	Done() <-chan struct{}
	Cancel()
}

func ReceiveTCPLoop[T cancelableContext](ctx T, conn *net.TCPConn, gatherer *channel.Gatherer) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		// TODO: use buffer pool
		newPacket, err := packet.ReadPacket(conn)
		if err != nil {
			log.Println("error reading packet:", err)
			if err == io.EOF {
				log.Println("stop handling connection from:", conn.RemoteAddr().String())
				return
			}
			continue
		}
		data := buffer.WithBuffer[[]*packet.Packet]{
			Thing:  []*packet.Packet{newPacket},
			Buffer: buffer.NewPackedBuffer(),
		}
		gatherer.Forward(data.MoveArg())
	}
}

// func processPacketLoop[T cancelableContext](ctx T, channel <-chan buffer.ArgPtr[*buffer.PackedBuffer], gatherer *channel.Gatherer) {
// 	defer ctx.Cancel()
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		case data_ := <-channel:
// 			data := data_.ToOwned()
// 			packets, _, err := packet.ParsePacket(data.Ptr.Buffer[:data.Ptr.TotalSize])
// 			if err != nil {
// 				log.Println("error parsing packet:", err)
// 				continue
// 			}
// 			withBuffer := buffer.WithBuffer[[]*packet.Packet]{
// 				Thing:  packets,
// 				Buffer: data.Move(),
// 			}
// 			gatherer.Forward(withBuffer.MoveArg())
// 		}
// 	}
// }

// func ReceiveTCPLoop[T cancelableContext](ctx T, conn *net.TCPConn, gatherer *channel.Gatherer) {
// 	defer ctx.Cancel()
// 	// reader := bufio.NewReader(conn)
// 	channel := make(chan buffer.ArgPtr[*buffer.PackedBuffer], 16)
// 	go processPacketLoop(ctx, channel, gatherer)
// 	for {
// 		select {
// 		case <-ctx.Done():
// 			return
// 		default:
// 		}
// 		pBuffer := buffer.NewPackedBuffer()
// 		n, err := conn.Read(pBuffer.Ptr.Buffer[:])
// 		if err != nil {
// 			log.Println("error reading packet:", err)
// 			if err == io.EOF {
// 				log.Println("stop handling connection from:", conn.RemoteAddr().String())
// 				return
// 			}
// 			continue
// 		}
// 		pBuffer.Ptr.TotalSize = n
// 		channel <- pBuffer.MoveArg()
// 	}
// }

func SendTCPLoop[T cancelableContext](ctx T, conn *net.TCPConn, inChan <-chan buffer.ArgPtr[*buffer.PackedBuffer]) {
	defer ctx.Cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case data_ := <-inChan:
			data := data_.ToOwned()
			n, err := conn.Write(data.Ptr.Buffer[:data.Ptr.TotalSize])
			if err != nil {
				log.Println("error sending packet:", err)
				data.Release()
				ctx.Cancel()
				return
			}
			if n != data.Ptr.TotalSize {
				log.Println("error writing to tcp: wrote", n, "bytes instead of", data.Ptr.TotalSize)
			}
			data.Release()
		}
	}
}
