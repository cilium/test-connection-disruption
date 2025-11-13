package common

import (
	"os"

	"github.com/cilium/test-connection-disruption/internal"
)

func MarkServerReady() {
	file, err := os.Create("/tmp/server-ready")
	internal.ErrExit("create ready file", err)
	internal.ErrExit("close ready file", file.Close())
}

func MarkClientReady() {
	file, err := os.Create("/tmp/client-ready")
	internal.ErrExit("create ready file", err)
	internal.ErrExit("close ready file", file.Close())
}
