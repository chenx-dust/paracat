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

	filterChan  *channel.FilterChannel
	dispatcher  *channel.Dispatcher
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
		filterChan:    channel.NewFilterChannel(cfg.ChannelSize),
		dispatcher:    channel.NewDispatcher(cfg.DispatchType),
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

	transport.EnableGRO(client.udpListener)
	transport.EnableGSO(client.udpListener)

	log.Println("listening on", client.cfg.ListenAddr)

	go client.handleReverse(client.filterChan.GetOutChan())

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
				pkg, band := client.dispatcher.StatisticIn.GetAndReset()
				log.Printf("dispatcher in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.dispatcher.StatisticOut.GetAndReset()
				log.Printf("dispatcher out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.filterChan.StatisticIn.GetAndReset()
				log.Printf("filter in: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
				pkg, band = client.filterChan.StatisticOut.GetAndReset()
				log.Printf("filter out: %d packets, %d bytes in %s, %.2f MB/s", pkg, band, client.cfg.ReportInterval, float64(band)/client.cfg.ReportInterval.Seconds()/1024/1024)
			}
		}()
	}

	wg.Wait()

	return nil
}
