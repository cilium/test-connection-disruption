package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"sync"

	"github.com/cilium/test-connection-disruption/internal"
)

func init() {
	internal.ErrExit("being nice", internal.BeNice())
}

func main() {
	flag.Parse()
	port := flag.Arg(0)
	if port == "" {
		fmt.Println("Usage: server <port>")
		os.Exit(1)
	}

	listen, err := net.Listen("tcp", ":"+port)
	internal.ErrExit("listen", err)

	ctx, _ := signal.NotifyContext(context.Background(), os.Interrupt)
	go func() {
		<-ctx.Done()
		fmt.Println("Closing listener")
		listen.Close()
	}()

	ready()

	fmt.Printf("Listening on port %s...\n", port)

	wg := &sync.WaitGroup{}
	accept(ctx, wg, listen)

	wg.Wait()
}

func accept(ctx context.Context, wg *sync.WaitGroup, listen net.Listener) {
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			conn, err := listen.Accept()
			if errors.Is(err, net.ErrClosed) {
				fmt.Println("Listener closed")
				return
			}
			internal.ErrExit("accept conn", err)

			read(ctx, wg, conn)
		}
	}()
}

func read(ctx context.Context, wg *sync.WaitGroup, conn net.Conn) {
	wg.Add(1)

	ctx, cancel := context.WithCancel(ctx)
	go func() {
		<-ctx.Done()
		fmt.Println("Closing connection to", conn.RemoteAddr())
		conn.Close()
	}()

	go func() {
		defer wg.Done()

		// Make sure context done channel unblocks when the reader exits so we don't
		// leak the goroutine created above.
		defer cancel()

		fmt.Println("New connection from", conn.RemoteAddr())
		defer conn.Close()

		// Read+write one message at a time.
		buf := make([]byte, internal.MsgSize)
		for {
			_, err := io.ReadFull(conn, buf)
			if errors.Is(err, net.ErrClosed) || errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error reading from %s: %s\n", conn.RemoteAddr(), err)
				return
			}

			_, err = conn.Write(buf)
			if errors.Is(err, net.ErrClosed) {
				return
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error writing to %s: %s\n", conn.RemoteAddr(), err)
				return
			}
		}
	}()
}

func ready() {
	file, err := os.Create("/tmp/server-ready")
	internal.ErrExit("create ready file", err)
	internal.ErrExit("close ready file", file.Close())
}
