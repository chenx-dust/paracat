package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/transport"
)

type tcpRelay struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.TCPAddr
	conn   *net.TCPConn
	ch     chan buffer.ArgPtr[*buffer.PackedBuffer]
}

func (relay *tcpRelay) Done() <-chan struct{} {
	return relay.ctx.Done()
}

func (relay *tcpRelay) Cancel() {
	relay.cancel()
}

func (client *Client) newTCPRelay(addr *net.TCPAddr) (relay *tcpRelay) {
	relay = &tcpRelay{addr: addr}
	client.connectTCPRelay(relay)
	return
}

func (client *Client) connectTCPRelay(relay *tcpRelay) {
	var err error
	retry := 0
	for {
		relay.conn, err = net.DialTCP("tcp", nil, relay.addr)
		if err == nil {
			break
		}
		log.Println("error dialing tcp:", err, "retry:", retry)
		time.Sleep(client.cfg.ReconnectDelay)
		retry++
	}
	relay.ctx, relay.cancel = context.WithCancel(context.Background())
	relay.ch = make(chan buffer.ArgPtr[*buffer.PackedBuffer], client.cfg.ChannelSize)
	client.scatterer.NewOutput(relay.ch)
	go client.handleTCPRelayCancel(relay)
	go transport.SendTCPLoop(relay, relay.conn, relay.ch)
	go transport.ReceiveTCPLoop(relay, relay.conn, client.gatherer)
}

func (client *Client) handleTCPRelayCancel(relay *tcpRelay) {
	<-relay.ctx.Done()
	log.Println("closing tcp relay:", relay.addr)
	relay.conn.Close()
	client.scatterer.RemoveOutput(relay.ch)
	client.connectTCPRelay(relay)
}
