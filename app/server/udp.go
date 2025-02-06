package server

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/transport"
)

type udpConnContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.UDPAddr
	timer  *time.Timer
	conn   *net.UDPConn
	ch     chan buffer.ArgPtr[*buffer.PackedBuffer]
}

func (ctx *udpConnContext) Done() <-chan struct{} {
	return ctx.ctx.Done()
}

func (ctx *udpConnContext) Cancel() {
	ctx.cancel()
}

func (server *Server) newUDPConnContext(addr *net.UDPAddr) *udpConnContext {
	ctx, cancel := context.WithCancel(context.Background())
	newCtx := &udpConnContext{
		ctx:    ctx,
		cancel: cancel,
		addr:   addr,
		timer:  time.NewTimer(server.cfg.UDPTimeout),
		conn:   server.udpListener,
		ch:     make(chan buffer.ArgPtr[*buffer.PackedBuffer], server.cfg.ChannelSize),
	}
	server.scatterer.NewOutput(newCtx.ch)
	newCtx.conn = server.udpListener
	go server.handleUDPConnContextCancel(newCtx)
	go server.handleUDPConnTimeout(newCtx)
	go transport.SendUDPLoop(newCtx, newCtx.conn, newCtx.addr, newCtx.ch)
	server.sourceMutex.Lock()
	server.sourceUDPAddrs[addr.String()] = newCtx
	server.sourceMutex.Unlock()
	return newCtx
}

func (server *Server) handleUDPConnContextCancel(ctx *udpConnContext) {
	<-ctx.ctx.Done()
	log.Println("closing udp connection:", ctx.addr.String())
	ctx.timer.Stop()
	server.scatterer.RemoveOutput(ctx.ch)
	server.sourceMutex.Lock()
	delete(server.sourceUDPAddrs, ctx.addr.String())
	server.sourceMutex.Unlock()
}

func (server *Server) handleUDPListener() {
	for {
		packets, udpAddr, err := transport.ReceiveUDPPackets(server.udpListener)
		if err != nil {
			packets.Release()
			continue
		}

		server.handleUDPAddr(udpAddr)
		server.gatherer.Forward(packets.MoveArg())
	}
}

func (server *Server) handleUDPAddr(addr *net.UDPAddr) {
	server.sourceMutex.RLock()
	ctx, ok := server.sourceUDPAddrs[addr.String()]
	server.sourceMutex.RUnlock()
	if ok {
		ctx.timer.Reset(server.cfg.UDPTimeout)
	} else {
		log.Println("new udp connection from", addr.String())
		server.newUDPConnContext(addr)
	}
}

func (server *Server) handleUDPConnTimeout(ctx *udpConnContext) {
	select {
	case <-ctx.timer.C:
		ctx.cancel()
	case <-ctx.ctx.Done():
		return
	}
}
