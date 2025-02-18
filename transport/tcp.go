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
	pBuffer := buffer.NewPackedBuffer()
	start := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := conn.Read(pBuffer.Ptr.Buffer[start:])
		if err != nil {
			log.Println("error reading packet:", err)
			if err == io.EOF {
				log.Println("stop handling connection from:", conn.RemoteAddr().String())
				return
			}
			continue
		}
		pBuffer.Ptr.TotalSize += n
		packets, remain, err := packet.ParsePacket(pBuffer.Ptr.Buffer[:start+n])
		if err != nil {
			log.Println("error parsing packet:", err)
			continue
		}
		withBuffer := buffer.WithBuffer[[]*packet.Packet]{
			Thing:  packets,
			Buffer: pBuffer.Move(),
		}
		newBuffer := buffer.NewPackedBuffer()
		if remain > 0 && remain < n {
			newBuffer.Ptr.TotalSize = remain
			copy(newBuffer.Ptr.Buffer[:remain], pBuffer.Ptr.Buffer[n-remain:n])
		} else {
			remain = 0
		}
		start = remain
		pBuffer = newBuffer.Move()
		gatherer.Forward(withBuffer.MoveArg())
	}
}

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
				if err == io.EOF {
					ctx.Cancel()
					return
				}
			}
			if n != data.Ptr.TotalSize {
				log.Println("error writing to tcp: wrote", n, "bytes instead of", data.Ptr.TotalSize)
			}
			data.Release()
		}
	}
}
