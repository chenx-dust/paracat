package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/transport"
)

type udpRelay struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.UDPAddr
	conn   *net.UDPConn
	ch     chan [][]byte
}

func (relay *udpRelay) Done() <-chan struct{} {
	return relay.ctx.Done()
}

func (relay *udpRelay) Cancel() {
	relay.cancel()
}

func (client *Client) newUDPRelay(addr *net.UDPAddr) (relay *udpRelay, err error) {
	relay = &udpRelay{addr: addr}
	err = client.connectUDPRelay(relay)
	return
}

func (client *Client) connectUDPRelay(relay *udpRelay) error {
	var err error
	for retry := 0; retry < client.cfg.ReconnectTimes; retry++ {
		relay.conn, err = net.ListenUDP("udp", nil)
		if err != nil {
			log.Println("error dialing udp:", err, "retry:", retry)
			time.Sleep(client.cfg.ReconnectDelay)
			continue
		}
		transport.EnableGRO(relay.conn)
		transport.EnableGSO(relay.conn)

		relay.ctx, relay.cancel = context.WithCancel(context.Background())
		relay.ch = make(chan [][]byte, client.cfg.ChannelSize)
		client.dispatcher.NewOutput(relay.ch)
		go client.handleUDPRelayCancel(relay)
		go client.handleUDPRelayRecv(relay)
		go transport.SendUDPLoop(relay, relay.conn, relay.addr, relay.ch)
		return nil
	}
	return err
}

func (client *Client) handleUDPRelayCancel(relay *udpRelay) {
	<-relay.ctx.Done()
	log.Println("closing udp relay:", relay.addr)
	relay.conn.Close()
	client.dispatcher.RemoveOutput(relay.ch)
	err := client.connectUDPRelay(relay)
	if err != nil {
		log.Println("failed to reconnect udp relay:", err)
	}
}

func (client *Client) handleUDPRelayRecv(relay *udpRelay) {
	defer relay.cancel()
	for {
		select {
		case <-relay.ctx.Done():
			return
		default:
		}
		packets, addr, err := transport.ReceiveUDPPackets(relay.conn)
		if err != nil {
			continue
		}
		if addr.String() != relay.addr.String() {
			log.Println("error receiving udp packets: addr mismatch", addr, relay.addr)
			continue
		}
		client.filterChan.Forward(packets)
	}
}
