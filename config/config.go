package config

import "time"

type AppMode int
type ConnectionType int
type ScatterType int
type TrafficType int

const (
	NotDefined AppMode = iota
	ClientMode
	RelayMode // for udp-to-tcp or tcp-to-udp
	ServerMode
)

const (
	NotDefinedConnectionType ConnectionType = iota
	TCPConnectionType
	UDPConnectionType
	BothConnectionType
)

const (
	NotDefinedScatterType ScatterType = iota
	RoundRobinScatterType
	ConcurrentScatterType
)

const (
	BothTrafficType TrafficType = iota
	UpTrafficType
	DownTrafficType
)

type Config struct {
	Mode           AppMode
	ListenAddr     string
	RemoteAddr     string        // not necessary in ClientMode
	RelayServers   []RelayServer // only used in ClientMode
	RelayType      RelayType     // only used in RelayMode
	ChannelSize    int
	ReportInterval time.Duration
	ReconnectDelay time.Duration // only used in ClientMode
	UDPTimeout     time.Duration // only used in ServerMode
	ScatterType    ScatterType
	MaxUDPSize     uint16
	EnableGRO      bool
	EnableGSO      bool
}

type RelayServer struct {
	Address  string
	ConnType ConnectionType
	Weight   int
	Traffic  TrafficType
}

type RelayType struct {
	ListenType  ConnectionType
	ForwardType ConnectionType
}

func ConnTypeToString(connType ConnectionType) string {
	switch connType {
	case TCPConnectionType:
		return "tcp"
	case UDPConnectionType:
		return "udp"
	case BothConnectionType:
		return "both"
	default:
		return "unknown"
	}
}

func ScatterTypeToString(scatterType ScatterType) string {
	switch scatterType {
	case RoundRobinScatterType:
		return "round-robin"
	case ConcurrentScatterType:
		return "concurrent"
	default:
		return "unknown"
	}
}

func TrafficTypeToString(trafficType TrafficType) string {
	switch trafficType {
	case BothTrafficType:
		return "both"
	case UpTrafficType:
		return "up"
	case DownTrafficType:
		return "down"
	default:
		return "unknown"
	}
}
