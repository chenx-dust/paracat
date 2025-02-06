package relay

import (
	"errors"
	"log"
	"net"
	"sync"

	"github.com/chenx-dust/paracat/config"
	"github.com/chenx-dust/paracat/transport"
)

type Relay struct {
	cfg *config.Config

	listenTCP bool
	listenUDP bool

	tcpListener *net.TCPListener
	udpListener *net.UDPConn

	forwardTCP bool
	forwardUDP bool

	tcpDialer *net.TCPConn
	udpDialer *net.UDPConn
}

func NewRelay(cfg *config.Config) *Relay {
	return &Relay{cfg: cfg}
}

func (relay *Relay) Run() error {
	log.Println("running relay")

	if relay.cfg.RelayType.ListenType == config.NotDefinedConnectionType {
		return errors.New("listen type not defined")
	}
	relay.listenTCP = relay.cfg.RelayType.ListenType&config.TCPConnectionType != 0
	relay.listenUDP = relay.cfg.RelayType.ListenType&config.UDPConnectionType != 0

	if relay.listenTCP {
		tcpAddr, err := net.ResolveTCPAddr("tcp", relay.cfg.ListenAddr)
		if err != nil {
			return err
		}
		relay.tcpListener, err = net.ListenTCP("tcp", tcpAddr)
		if err != nil {
			return err
		}
		log.Println("listening on", relay.cfg.ListenAddr)
	}
	if relay.listenUDP {
		udpAddr, err := net.ResolveUDPAddr("udp", relay.cfg.ListenAddr)
		if err != nil {
			return err
		}
		relay.udpListener, err = net.ListenUDP("udp", udpAddr)
		if err != nil {
			return err
		}
		transport.EnableGRO(relay.udpListener)
		transport.EnableGSO(relay.udpListener)
		log.Println("listening on", relay.cfg.ListenAddr)
	}

	if relay.cfg.RelayType.ForwardType == config.NotDefinedConnectionType {
		return errors.New("forward type not defined")
	}
	relay.forwardTCP = relay.cfg.RelayType.ForwardType&config.TCPConnectionType != 0
	relay.forwardUDP = relay.cfg.RelayType.ForwardType&config.UDPConnectionType != 0

	if relay.forwardTCP {
		tcpAddr, err := net.ResolveTCPAddr("tcp", relay.cfg.RemoteAddr)
		if err != nil {
			return err
		}
		relay.tcpDialer, err = net.DialTCP("tcp", nil, tcpAddr)
		if err != nil {
			return err
		}
		log.Println("forwarding tcp to", relay.cfg.RemoteAddr)
	}
	if relay.forwardUDP {
		udpAddr, err := net.ResolveUDPAddr("udp", relay.cfg.RemoteAddr)
		if err != nil {
			return err
		}
		relay.udpDialer, err = net.DialUDP("udp", nil, udpAddr)
		if err != nil {
			return err
		}
		transport.EnableGRO(relay.udpDialer)
		transport.EnableGSO(relay.udpDialer)
		log.Println("forwarding udp to", relay.cfg.RemoteAddr)
	}

	wg := sync.WaitGroup{}
	if relay.listenTCP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			relay.handleTCPForward()
		}()
	}
	if relay.listenUDP {
		wg.Add(1)
		go func() {
			defer wg.Done()
			relay.handleUDPForward()
		}()
	}
	wg.Wait()

	return nil
}
