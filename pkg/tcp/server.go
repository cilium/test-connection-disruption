package tcp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
)

type TcpServer struct {
	config common.ServerConfig
}

func NewTcpServer(config common.ServerConfig) *TcpServer {
	return &TcpServer{config: config}
}

func (s *TcpServer) Run(ctx context.Context, _ context.CancelFunc) {
	fmt.Printf("Starting TCP Server with config: %#v\n", s.config)

	listen, err := net.Listen("tcp", ":"+s.config.Address)
	internal.ErrExit("TcpServer listen", err)

	go func() {
		<-ctx.Done()
		fmt.Println("Closing listener")
		listen.Close()
	}()

	common.MarkServerReady()

	wg := &sync.WaitGroup{}
	s.accept(ctx, wg, listen)

	wg.Wait()
}

func (s *TcpServer) accept(ctx context.Context, wg *sync.WaitGroup, listen net.Listener) {
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

			s.read(ctx, wg, conn)
		}
	}()
}

func (s *TcpServer) read(ctx context.Context, wg *sync.WaitGroup, conn net.Conn) {
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
