package server

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/transport"
)

type udpConnContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.UDPAddr
	timer  *time.Timer
	dialer *net.UDPConn
}

func (server *Server) newUDPConnContext(addr *net.UDPAddr) *udpConnContext {
	ctx, cancel := context.WithCancel(context.Background())
	newCtx := &udpConnContext{ctx, cancel, addr, time.NewTimer(server.cfg.UDPTimeout), server.udpListener}
	server.dispatcher.NewOutput(newCtx)
	go server.handleUDPConnContextCancel(newCtx)
	return newCtx
}

func (server *Server) handleUDPConnContextCancel(ctx *udpConnContext) {
	<-ctx.ctx.Done()
	ctx.timer.Stop()
	server.dispatcher.RemoveOutput(ctx)
	server.sourceMutex.Lock()
	delete(server.sourceUDPAddrs, ctx.addr.String())
	server.sourceMutex.Unlock()
}

func (server *Server) handleUDP() {
	for {
		packets, udpAddr, err := transport.ReceiveUDPPackets(server.udpListener)
		if err != nil {
			continue
		}

		server.handleUDPAddr(udpAddr)
		server.filterChan.Forward(packets)
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
		newCtx := server.newUDPConnContext(addr)
		server.sourceMutex.Lock()
		server.sourceUDPAddrs[addr.String()] = newCtx
		server.sourceMutex.Unlock()
		go server.handleUDPConnTimeout(newCtx)
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

func (ctx *udpConnContext) Write(packet []byte) (n int, err error) {
	n, err = ctx.dialer.WriteToUDP(packet, ctx.addr)
	if err != nil {
		log.Println("error writing packet:", err)
	} else if n != len(packet) {
		log.Println("error writing packet: wrote", n, "bytes instead of", len(packet))
	}
	return
}
