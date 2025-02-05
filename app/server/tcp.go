package server

import (
	"context"
	"log"
	"net"

	"github.com/chenx-dust/paracat/transport"
)

type tcpConnContext struct {
	ctx     context.Context
	cancel_ context.CancelFunc
	conn    *net.TCPConn
}

func (ctx *tcpConnContext) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *tcpConnContext) Cancel() {
	ctx.cancel_()
}

func (server *Server) newTCPConnContext(conn *net.TCPConn) *tcpConnContext {
	ctx, cancel := context.WithCancel(context.Background())
	server.dispatcher.NewOutput(conn)
	newCtx := &tcpConnContext{ctx, cancel, conn}
	go server.handleTCPConnContextCancel(newCtx)
	return newCtx
}

func (server *Server) handleTCPConnContextCancel(ctx *tcpConnContext) {
	<-ctx.ctx.Done()
	ctx.conn.Close()
	server.dispatcher.RemoveOutput(ctx.conn)
}

func (server *Server) handleTCP() {
	for {
		conn, err := server.tcpListener.AcceptTCP()
		if err != nil {
			log.Fatalln("error accepting tcp connection:", err)
		}
		server.handleTCPConn(conn)
	}
}

func (server *Server) handleTCPConn(conn *net.TCPConn) {
	log.Println("new tcp connection from", conn.RemoteAddr().String())
	ctx := server.newTCPConnContext(conn)
	go transport.TCPFowrardLoop(ctx, conn, server.filterChan)
}
