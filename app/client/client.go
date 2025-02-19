package client

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

type Client struct {
	cfg *config.Config

	gatherer    *channel.Gatherer
	scatterer   *channel.Scatterer
	idIncrement atomic.Uint32

	udpListener *net.UDPConn

	connMutex     sync.RWMutex
	connIncrement atomic.Uint32
	connIDAddrMap map[uint16]*net.UDPAddr
	connAddrIDMap map[string]uint16
}

func NewClient(cfg *config.Config) *Client {
	return &Client{
		cfg:           cfg,
		gatherer:      channel.NewGatherer(cfg.ChannelSize),
		scatterer:     channel.NewScatterer(cfg.ScatterType),
		connIDAddrMap: make(map[uint16]*net.UDPAddr),
		connAddrIDMap: make(map[string]uint16),
	}
}

func (client *Client) Run() error {
	log.Println("running client")

	udpAddr, err := net.ResolveUDPAddr("udp", client.cfg.ListenAddr)
	if err != nil {
		return err
	}
	client.udpListener, err = net.ListenUDP("udp", udpAddr)
	if err != nil {
		return err
	}

	if client.cfg.EnableGRO {
		transport.EnableGRO(client.udpListener)
	}
	if client.cfg.EnableGSO {
		transport.EnableGSO(client.udpListener)
	}

	log.Println("listening on", client.cfg.ListenAddr)

	go client.handleReverse(client.gatherer.GetOutChan())

	client.dialRelays()

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		client.handleForward()
	}()

	if client.cfg.ReportInterval > 0 {
		go func() {
			ticker := time.NewTicker(client.cfg.ReportInterval)
			defer ticker.Stop()
			for range ticker.C {
				pkg, band := client.scatterer.StatisticIn.GetAndReset()
				log.Printf("scatter in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.scatterer.StatisticOut.GetAndReset()
				log.Printf("scatter out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.gatherer.StatisticIn.GetAndReset()
				log.Printf("gather in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.gatherer.StatisticOut.GetAndReset()
				log.Printf("gather out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)

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
