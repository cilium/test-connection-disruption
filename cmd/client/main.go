package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"time"

	flag "github.com/spf13/pflag"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
	"github.com/cilium/test-connection-disruption/pkg/tcp"
)

func init() {
	internal.ErrExit("being nice", internal.BeNice())
}

func main() {
	cc := common.ClientConfig{}
	cc.RegisterFlags()
	flag.Parse()

	cc.ProtocolConfig.Address = flag.Arg(0)
	if cc.ProtocolConfig.Address == "" {
		fmt.Println("Usage: client --protocol <tcp> <port/address>")
		flag.Usage()
		os.Exit(1)
	}

	// For backwards compatibility, clamp the interval to a minimum of 10ms to
	// avoid overloading resource-constrained CI machines where Cilium runs with
	// monitor aggregation disabled.
	if cc.Interval == 0 {
		cc.Interval = 10 * time.Millisecond
		fmt.Println("Zero interval changed to", cc.Interval, "for backwards compatibility.")
	}

	var handler common.ProtocolHandler

	switch common.Protocol(cc.Protocol) {
	case common.ProtocolTCP:
		handler = tcp.NewTcpClient(cc)
	default:
		fmt.Printf("Invalid Protocol: %s\n", cc.Protocol)
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	handler.Run(ctx, cancel)
}
