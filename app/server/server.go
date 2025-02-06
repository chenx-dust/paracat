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

	gatherer    *channel.Gatherer
	scatterer   *channel.Scatterer
	idIncrement atomic.Uint32

	sourceMutex    sync.RWMutex
	sourceUDPAddrs map[string]*udpConnContext

	forwardConns map[uint16]*net.UDPConn
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg:            cfg,
		gatherer:       channel.NewGatherer(cfg.ChannelSize),
		scatterer:      channel.NewScatterer(cfg.ScatterType),
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

	go server.handleForward(server.gatherer.GetOutChan())

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
				pkg, band := server.scatterer.StatisticIn.GetAndReset()
				log.Printf("scatter in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.scatterer.StatisticOut.GetAndReset()
				log.Printf("scatter out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.gatherer.StatisticIn.GetAndReset()
				log.Printf("gather in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = server.gatherer.StatisticOut.GetAndReset()
				log.Printf("gather out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, server.cfg.ReportInterval, float64(band)/server.cfg.ReportInterval.Seconds()/1024/1024)

				// buffer.BufferTraceBack.Lock()
				// for k, v := range buffer.BufferTraceBack.TraceBack {
				// 	log.Printf("buffer trace back: %s, %d", k, v)
				// }
				// buffer.BufferTraceBack.Unlock()
			}
		}()
	}
	wg.Wait()

	return nil
}
