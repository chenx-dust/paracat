package server

import (
	"context"
	"log"
	"net"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/transport"
)

type tcpConnContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	conn   *net.TCPConn
	ch     chan<- buffer.ArgPtr[*buffer.PackedBuffer]
}

func (ctx *tcpConnContext) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *tcpConnContext) Cancel() {
	ctx.cancel()
}

func (server *Server) newTCPConnContext(conn *net.TCPConn) *tcpConnContext {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan buffer.ArgPtr[*buffer.PackedBuffer], server.cfg.ChannelSize)
	server.scatterer.NewOutput(ch)
	newCtx := &tcpConnContext{ctx, cancel, conn, ch}
	go server.handleTCPConnContextCancel(newCtx)
	go transport.ReceiveTCPLoop(newCtx, conn, server.gatherer)
	go transport.SendTCPLoop(newCtx, conn, ch)
	return newCtx
}

func (server *Server) handleTCPConnContextCancel(ctx *tcpConnContext) {
	<-ctx.ctx.Done()
	log.Println("closing tcp connection:", ctx.conn.RemoteAddr().String())
	ctx.conn.Close()
	server.scatterer.RemoveOutput(ctx.ch)
}

func (server *Server) handleTCP() {
	for {
		conn, err := server.tcpListener.AcceptTCP()
		if err != nil {
			log.Fatalln("error accepting tcp connection:", err)
		}
		log.Println("new tcp connection from", conn.RemoteAddr().String())
		server.newTCPConnContext(conn)
	}
}
