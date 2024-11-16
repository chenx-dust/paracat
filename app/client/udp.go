package client

import (
	"context"
	"log"
	"net"
	"time"

	"github.com/chenx-dust/paracat/packet"
)

type udpRelay struct {
	ctx      context.Context
	cancel   context.CancelFunc
	addr     *net.UDPAddr
	conn     *net.UDPConn
	sendChan <-chan []byte
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
			time.Sleep(client.cfg.ReconnectDelay * time.Second)
			continue
		}
		relay.ctx, relay.cancel = context.WithCancel(context.Background())
		relay.sendChan = client.dispatcher.NewOutChan()
		go client.handleUDPRelayCancel(relay)
		go client.handleUDPRelayRecv(relay)
		go client.handleUDPRelaySend(relay)
		return nil
	}
	return err
}

func (client *Client) handleUDPRelayCancel(relay *udpRelay) {
	<-relay.ctx.Done()
	relay.conn.Close()
	client.dispatcher.CloseOutChan(relay.sendChan)
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
		buf := make([]byte, client.cfg.BufferSize)
		n, err := relay.conn.Read(buf)
		if err != nil {
			log.Println("error reading from reverse conn:", err)
			log.Println("close reverse conn from:", relay.conn.RemoteAddr())
			return
		}

		newPacket, err := packet.Unpack(buf[:n])
		if err != nil {
			log.Println("error unpacking packet:", err)
			continue
		}

		select {
		case client.filterChan.InChan() <- newPacket:
		default:
			log.Println("filter channel is full, drop packet")
		}
	}
}

func (client *Client) handleUDPRelaySend(relay *udpRelay) {
	defer relay.cancel()
	for {
		select {
		case packet := <-relay.sendChan:
			n, err := relay.conn.Write(packet)
			if err != nil {
				log.Println("error writing packet:", err)
				log.Println("stop handling connection to:", relay.conn.RemoteAddr().String())
				return
			}
			if n != len(packet) {
				log.Println("error writing packet: wrote", n, "bytes instead of", len(packet))
				continue
			}
		case <-relay.ctx.Done():
			return
		}
	}
}
