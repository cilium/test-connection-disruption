package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	flag "github.com/spf13/pflag"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
	"github.com/cilium/test-connection-disruption/pkg/tcp"
)

func init() {
	internal.ErrExit("being nice", internal.BeNice())
}

func main() {
	sc := common.ServerConfig{}
	sc.RegisterFlags()
	flag.Parse()

	sc.ProtocolConfig.Address = flag.Arg(0)
	if sc.ProtocolConfig.Address == "" {
		fmt.Println("Usage: server --protocol <tcp> <port/address>")
		flag.Usage()
		os.Exit(1)
	}

	var handler common.ProtocolHandler

	switch common.Protocol(sc.Protocol) {
	case common.ProtocolTCP:
		handler = tcp.NewTcpServer(sc)
	default:
		fmt.Printf("Invalid Protocol: %s\n", sc.Protocol)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	handler.Run(ctx, cancel)
}
