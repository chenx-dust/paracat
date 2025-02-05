package config

import "time"

type AppMode int
type ConnectionType int
type DispatchType int

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
	NotDefinedDispatchType DispatchType = iota
	RoundRobinDispatchType
	ConcurrentDispatchType
)

type Config struct {
	Mode           AppMode
	ListenAddr     string
	RemoteAddr     string        // not necessary in ClientMode
	RelayServers   []RelayServer // only used in ClientMode
	RelayType      RelayType     // only used in RelayMode
	ChannelSize    int
	ReportInterval time.Duration
	ReconnectTimes int           // only used in ClientMode
	ReconnectDelay time.Duration // only used in ClientMode
	UDPTimeout     time.Duration // only used in ServerMode
	DispatchType   DispatchType
}

type RelayServer struct {
	Address  string
	ConnType ConnectionType
	Weight   int
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

func DispatchTypeToString(dispatchType DispatchType) string {
	switch dispatchType {
	case RoundRobinDispatchType:
		return "round-robin"
	case ConcurrentDispatchType:
		return "concurrent"
	default:
		return "unknown"
	}
}
