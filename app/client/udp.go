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
}

func (client *Client) newUDPRelay(addr *net.UDPAddr) (relay *udpRelay, err error) {
	relay = &udpRelay{addr: addr}
	err = client.connectUDPRelay(relay)
	return
}

func (client *Client) connectUDPRelay(relay *udpRelay) error {
	var err error
	for retry := 0; retry < client.cfg.ReconnectTimes; retry++ {
		relay.conn, err = net.DialUDP("udp", nil, relay.addr)
		if err != nil {
			log.Println("error dialing udp:", err, "retry:", retry)
			time.Sleep(client.cfg.ReconnectDelay)
			continue
		}
		err = transport.EnableGRO(relay.conn)
		// relay.gro = err == nil
		// err = client.enableGSO(relay.conn)
		// relay.gso = err == nil

		relay.ctx, relay.cancel = context.WithCancel(context.Background())
		client.dispatcher.NewOutput(relay)
		go client.handleUDPRelayCancel(relay)
		go client.handleUDPRelayRecv(relay)
		return nil
	}
	return err
}

func (client *Client) handleUDPRelayCancel(relay *udpRelay) {
	<-relay.ctx.Done()
	relay.conn.Close()
	client.dispatcher.RemoveOutput(relay)
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
		packets, _, err := transport.ReceiveUDPPackets(relay.conn)
		if err != nil {
			continue
		}

		client.filterChan.Forward(packets)
	}
}

func (relay *udpRelay) Write(packet []byte) (n int, err error) {
	n, err = relay.conn.Write(packet)
	if err != nil {
		log.Println("error writing packet:", err)
		relay.cancel()
	} else if n != len(packet) {
		log.Println("error writing packet: wrote", n, "bytes instead of", len(packet))
	}
	return
}
