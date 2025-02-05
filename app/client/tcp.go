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
		client.dispatcher.NewOutput(relay)
		go client.handleTCPRelayCancel(relay)
		go transport.TCPFowrardLoop(relay, relay.conn, client.filterChan)
		return nil
	}
	return err
}

func (client *Client) handleTCPRelayCancel(relay *tcpRelay) {
	<-relay.ctx.Done()
	relay.conn.Close()
	client.dispatcher.RemoveOutput(relay)
	err := client.connectTCPRelay(relay)
	if err != nil {
		log.Println("failed to reconnect tcp relay:", err)
	}
}

func (relay *tcpRelay) Write(packet []byte) (n int, err error) {
	n, err = relay.conn.Write(packet)
	if err != nil {
		log.Println("error writing packet:", err)
		log.Println("stop handling connection to:", relay.conn.RemoteAddr().String())
		relay.cancel()
	} else if n != len(packet) {
		log.Println("error writing packet: wrote", n, "bytes instead of", len(packet))
	}
	return
}
