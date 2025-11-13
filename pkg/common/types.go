package common

import (
	"context"
	"time"

	flag "github.com/spf13/pflag"
)

type Protocol string

const (
	ProtocolTCP  Protocol = "tcp"
	ProtocolHTTP Protocol = "http"
)

type ProtocolConfig struct {
	Protocol string
	Address  string
}

func (pc *ProtocolConfig) RegisterFlags() {
	flag.StringVar(&pc.Protocol, "protocol", string(ProtocolTCP), "Transport protocol to use")
}

type ServerConfig struct {
	ProtocolConfig
}

func (sc *ServerConfig) RegisterFlags() {
	sc.ProtocolConfig.RegisterFlags()
}

type ClientConfig struct {
	ProtocolConfig

	Interval time.Duration
	Timeout  time.Duration
}

func (cc *ClientConfig) RegisterFlags() {
	cc.ProtocolConfig.RegisterFlags()

	flag.DurationVar(&cc.Interval, "dispatch-interval", 50*time.Millisecond, "Client request dispatch interval")
	flag.DurationVar(&cc.Timeout, "timeout", 5*time.Second, "Client exits when no reply is received within this duration")
}

type ProtocolHandler interface {
	Run(ctx context.Context, cancel context.CancelFunc)
}
