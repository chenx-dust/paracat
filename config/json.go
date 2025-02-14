package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// JSONConfig represents the JSON structure that matches Config
type JSONConfig struct {
	Mode           string            `json:"mode"`
	ListenAddr     string            `json:"listen_addr"`
	RemoteAddr     string            `json:"remote_addr,omitempty"`
	RelayServers   []JSONRelayServer `json:"relay_servers,omitempty"`
	RelayType      *JSONRelayType    `json:"relay_type,omitempty"`
	ChannelSize    *int              `json:"channel_size,omitempty"`
	ReportInterval *string           `json:"report_interval,omitempty"`
	ReconnectDelay *string           `json:"reconnect_delay,omitempty"`
	UDPTimeout     *string           `json:"udp_timeout,omitempty"`
	ScatterType    *string           `json:"scatter_type,omitempty"`
}

type JSONRelayServer struct {
	Addr     string  `json:"addr"`
	ConnType string  `json:"conn_type"`
	Weight   *int    `json:"weight,omitempty"`
	Traffic  *string `json:"traffic,omitempty"`
}

type JSONRelayType struct {
	ListenType  string `json:"listen_type"`
	ForwardType string `json:"forward_type"`
}

const defaultWeight = 1
const defaultChannelSize = 64
const defaultReportInterval = 0 * time.Second
const defaultReconnectDelay = 5 * time.Second
const defaultUDPTimeout = 10 * time.Minute

// LoadFromFile reads and parses a JSON configuration file
func LoadFromFile(filepath string) (*Config, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var jsonConfig JSONConfig
	if err := json.Unmarshal(data, &jsonConfig); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return convertJSONConfig(jsonConfig)
}

// convertJSONConfig converts JSONConfig to Config
func convertJSONConfig(jc JSONConfig) (*Config, error) {
	// Convert mode string to AppMode
	var mode AppMode
	switch jc.Mode {
	case "client":
		mode = ClientMode
	case "relay":
		mode = RelayMode
	case "server":
		mode = ServerMode
	default:
		return nil, fmt.Errorf("invalid mode: %s", jc.Mode)
	}

	channelSize := defaultChannelSize
	if jc.ChannelSize != nil {
		channelSize = *jc.ChannelSize
	}

	reportInterval := defaultReportInterval
	if jc.ReportInterval != nil {
		d, err := time.ParseDuration(*jc.ReportInterval)
		if err != nil {
			return nil, fmt.Errorf("invalid report interval: %w", err)
		}
		reportInterval = d
	}

	reconnectDelay := defaultReconnectDelay
	if jc.ReconnectDelay != nil {
		d, err := time.ParseDuration(*jc.ReconnectDelay)
		if err != nil {
			return nil, fmt.Errorf("invalid reconnect delay: %w", err)
		}
		reconnectDelay = d
	}

	udpTimeout := defaultUDPTimeout
	if jc.UDPTimeout != nil {
		d, err := time.ParseDuration(*jc.UDPTimeout)
		if err != nil {
			return nil, fmt.Errorf("invalid udp timeout: %w", err)
		}
		udpTimeout = d
	}

	config := &Config{
		Mode:           mode,
		ListenAddr:     jc.ListenAddr,
		RemoteAddr:     jc.RemoteAddr,
		RelayServers:   convertJSONRelayServers(jc.RelayServers),
		ChannelSize:    channelSize,
		ReportInterval: reportInterval,
		ReconnectDelay: reconnectDelay,
		ScatterType:    convertJSONScatterType(jc.ScatterType),
		UDPTimeout:     udpTimeout,
	}

	if jc.RelayType != nil {
		config.RelayType = convertJSONRelayType(*jc.RelayType)
	}

	return config, nil
}

func convertJSONRelayServers(jsrs []JSONRelayServer) []RelayServer {
	rs := make([]RelayServer, len(jsrs))
	for i, jsr := range jsrs {
		weight := defaultWeight
		if jsr.Weight != nil {
			weight = *jsr.Weight
		}
		rs[i] = RelayServer{
			Address:  jsr.Addr,
			ConnType: convertJSONConnectionType(jsr.ConnType),
			Weight:   weight,
			Traffic:  convertJSONTrafficType(jsr.Traffic),
		}
	}
	return rs
}

func convertJSONRelayType(jrt JSONRelayType) RelayType {
	return RelayType{
		ListenType:  convertJSONConnectionType(jrt.ListenType),
		ForwardType: convertJSONConnectionType(jrt.ForwardType),
	}
}

func convertJSONConnectionType(connType string) ConnectionType {
	switch connType {
	case "tcp":
		return TCPConnectionType
	case "udp":
		return UDPConnectionType
	case "both":
		return BothConnectionType
	default:
		return NotDefinedConnectionType
	}
}

func convertJSONScatterType(scatterType *string) ScatterType {
	if scatterType == nil {
		return NotDefinedScatterType
	}
	switch *scatterType {
	case "round-robin":
		return RoundRobinScatterType
	case "concurrent":
		return ConcurrentScatterType
	default:
		return NotDefinedScatterType
	}
}

func convertJSONTrafficType(trafficType *string) TrafficType {
	if trafficType == nil {
		return BothTrafficType
	}
	switch *trafficType {
	case "up":
		return UpTrafficType
	case "down":
		return DownTrafficType
	default:
		return BothTrafficType
	}
}
