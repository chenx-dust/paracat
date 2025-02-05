package server

import (
	"log"
	"net"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/packet"
	"github.com/chenx-dust/paracat/transport"
)

func (server *Server) forwardLoop(ch <-chan []*packet.Packet) {
	for packets := range ch {
		for _, newPacket := range packets {
			server.handleForward(newPacket)
		}
	}
}

func (server *Server) handleForward(newPacket *packet.Packet) (int, error) {
	conn, ok := server.forwardConns[newPacket.ConnID]
	var err error
	if !ok {
		conn, err = net.ListenUDP("udp", nil)
		if err != nil {
			log.Println("error dialing relay:", err)
			return 0, err
		}
		transport.EnableGRO(conn)
		server.forwardConns[newPacket.ConnID] = conn
		go server.handleReverse(conn, newPacket.ConnID)
	}

	remoteAddr, err := net.ResolveUDPAddr("udp", server.cfg.RemoteAddr)
	if err != nil {
		log.Fatalln("error resolving remote addr:", err)
	}

	n, err := conn.WriteToUDP(newPacket.Buffer, remoteAddr)
	if err != nil {
		log.Println("error writing to udp:", err)
	} else if n != len(newPacket.Buffer) {
		log.Println("error writing to udp: wrote", n, "bytes instead of", len(newPacket.Buffer))
	}
	return n, nil
}

func (server *Server) handleReverse(conn *net.UDPConn, connID uint16) {
	remoteAddr, err := net.ResolveUDPAddr("udp", server.cfg.RemoteAddr)
	if err != nil {
		log.Fatalln("error resolving remote addr:", err)
	}
	for {
		packets, addr, err := transport.ReceiveUDPRawPackets(conn)
		if err != nil {
			log.Println("error receiving udp packets:", err)
			continue
		}
		if addr.String() != remoteAddr.String() {
			log.Println("error receiving udp packets: addr mismatch", addr, remoteAddr)
			continue
		}

		for _, rawPacket := range packets {
			packetID := channel.NewPacketID(&server.idIncrement)
			newPacket := &packet.Packet{
				Buffer:   rawPacket,
				ConnID:   connID,
				PacketID: packetID,
			}
			packed := newPacket.Pack()

			server.dispatcher.Dispatch(packed)
		}
	}
}
