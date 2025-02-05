package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/transport"
)

type tcpRelay struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.TCPAddr
	conn   *net.TCPConn
	ch     chan [][]byte
}

func (relay *tcpRelay) Done() <-chan struct{} {
	return relay.ctx.Done()
}

func (relay *tcpRelay) Cancel() {
	relay.cancel()
}

func (client *Client) newTCPRelay(addr *net.TCPAddr) (relay *tcpRelay, err error) {
	relay = &tcpRelay{addr: addr}
	err = client.connectTCPRelay(relay)
	return
}

func (client *Client) connectTCPRelay(relay *tcpRelay) error {
	var err error
	for retry := 0; retry < client.cfg.ReconnectTimes; retry++ {
		relay.conn, err = net.DialTCP("tcp", nil, relay.addr)
		if err != nil {
			log.Println("error dialing tcp:", err, "retry:", retry)
			time.Sleep(client.cfg.ReconnectDelay)
			continue
		}
		relay.ctx, relay.cancel = context.WithCancel(context.Background())
		relay.ch = make(chan [][]byte, client.cfg.ChannelSize)
		client.dispatcher.NewOutput(relay.ch)
		go client.handleTCPRelayCancel(relay)
		go transport.SendTCPLoop(relay, relay.conn, relay.ch)
		go transport.ReceiveTCPLoop(relay, relay.conn, client.filterChan)
		return nil
	}
	return err
}

func (client *Client) handleTCPRelayCancel(relay *tcpRelay) {
	<-relay.ctx.Done()
	log.Println("closing tcp relay:", relay.addr)
	relay.conn.Close()
	client.dispatcher.RemoveOutput(relay.ch)
	err := client.connectTCPRelay(relay)
	if err != nil {
		log.Println("failed to reconnect tcp relay:", err)
	}
}
