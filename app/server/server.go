package server

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chenx-dust/paracat/channel"
	"github.com/chenx-dust/paracat/config"
	"github.com/chenx-dust/paracat/transport"
)

type Server struct {
	cfg         *config.Config
	tcpListener *net.TCPListener
	udpListener *net.UDPConn

	filterChan  *channel.FilterChannel
	dispatcher  *channel.Dispatcher
	idIncrement atomic.Uint32

	sourceMutex    sync.RWMutex
	sourceUDPAddrs map[string]*udpConnContext

	forwardConns map[uint16]*net.UDPConn
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg:            cfg,
		filterChan:     channel.NewFilterChannel(cfg.ChannelSize),
		dispatcher:     channel.NewDispatcher(cfg.DispatchType),
		forwardConns:   make(map[uint16]*net.UDPConn),
		sourceUDPAddrs: make(map[string]*udpConnContext),
	}
}

func (server *Server) Run() error {
	log.Println("running server")

	tcpAddr, err := net.ResolveTCPAddr("tcp", server.cfg.ListenAddr)
	if err != nil {
		return err
	}
	server.tcpListener, err = net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return err
	}

	udpAddr, err := net.ResolveUDPAddr("udp", server.cfg.ListenAddr)
	if err != nil {
		return err
	}
	server.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}
	log.Println("listening on", server.cfg.ListenAddr)
	log.Println("dialing to", server.cfg.RemoteAddr)

	transport.EnableGRO(server.udpListener)
	transport.EnableGSO(server.udpListener)

	go server.handleForward(server.filterChan.GetOutChan())

	wg := sync.WaitGroup{}
	wg.Add(3)
	go func() {
		defer wg.Done()
		server.handleTCP()
	}()
	go func() {
		defer wg.Done()
		server.handleUDPListener()
	}()
	if server.cfg.ReportInterval > 0 {
		go func() {
			ticker := time.NewTicker(server.cfg.ReportInterval)
			defer ticker.Stop()
			for range ticker.C {
				pkg, band := server.dispatcher.StatisticIn.GetAndReset()
				log.Printf("dispatcher in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.dispatcher.StatisticOut.GetAndReset()
				log.Printf("dispatcher out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.filterChan.StatisticIn.GetAndReset()
				log.Printf("filter in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.filterChan.StatisticOut.GetAndReset()
				log.Printf("filter out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
			}
		}()
	}
	wg.Wait()

	return nil
}
