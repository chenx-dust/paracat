package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/buffer"
	"github.com/chenx-dust/paracat/transport"
)

type udpRelay struct {
	ctx    context.Context
	cancel context.CancelFunc
	addr   *net.UDPAddr
	conn   *net.UDPConn
	ch     chan buffer.ArgPtr[*buffer.PackedBuffer]
}

func (relay *udpRelay) Done() <-chan struct{} {
	return relay.ctx.Done()
}

func (relay *udpRelay) Cancel() {
	relay.cancel()
}

func (client *Client) newUDPRelay(addr *net.UDPAddr) (relay *udpRelay) {
	relay = &udpRelay{addr: addr}
	client.connectUDPRelay(relay)
	return
}

func (client *Client) connectUDPRelay(relay *udpRelay) {
	var err error
	retry := 0
	for {
		relay.conn, err = net.ListenUDP("udp", nil)
		if err == nil {
			break
		}
		log.Println("error dialing udp:", err, "retry:", retry)
		time.Sleep(client.cfg.ReconnectDelay)
		retry++
	}
	if client.cfg.EnableGRO {
		transport.EnableGRO(relay.conn)
	}
	if client.cfg.EnableGSO {
		transport.EnableGSO(relay.conn)
	}

	relay.ctx, relay.cancel = context.WithCancel(context.Background())
	relay.ch = make(chan buffer.ArgPtr[*buffer.PackedBuffer], client.cfg.ChannelSize)
	client.scatterer.NewOutput(relay.ch)
	go client.handleUDPRelayCancel(relay)
	go client.handleUDPRelayRecv(relay)
	go transport.SendUDPLoop(relay, relay.conn, relay.addr, relay.ch, client.cfg.EnableGSO)
}

func (client *Client) handleUDPRelayCancel(relay *udpRelay) {
	<-relay.ctx.Done()
	log.Println("closing udp relay:", relay.addr)
	relay.conn.Close()
	client.scatterer.RemoveOutput(relay.ch)
	client.connectUDPRelay(relay)
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
			packets.Release()
			continue
		}
		if addr.String() != relay.addr.String() {
			packets.Release()
			log.Println("error receiving udp packets: addr mismatch", addr, relay.addr)
			continue
		}
		client.gatherer.Forward(packets.MoveArg())
	}
}
