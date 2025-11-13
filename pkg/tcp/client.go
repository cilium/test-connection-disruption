package tcp

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
)

const maxAttempts = 30

type clientStats struct {
	rx, tx atomic.Uint64
	bytes  atomic.Uint64
}

type TcpClient struct {
	config common.ClientConfig
	stats  clientStats
}

func NewTcpClient(config common.ClientConfig) *TcpClient {
	return &TcpClient{
		config: config,
		stats:  clientStats{rx: atomic.Uint64{}, tx: atomic.Uint64{}, bytes: atomic.Uint64{}},
	}
}

func (c *TcpClient) Run(ctx context.Context, cancel context.CancelFunc) {
	fmt.Printf("Starting TCP Client with config: %#v\n", c.config)

	conn, err := c.dial()
	internal.ErrExit("dial remote", err)
	defer conn.Close()

	// Set up request payload.
	request := make([]byte, internal.MsgSize)
	_, err = rand.Read(request)
	internal.ErrExit("generate random payload", err)

	var eg errgroup.Group
	eg.Go(c.writer(ctx, cancel, conn, request))
	eg.Go(c.reader(ctx, cancel, conn, request))

	c.startLogger(ctx)

	common.MarkClientReady()
	internal.ErrExit("Error in writer or reader", eg.Wait())
}

func (c *TcpClient) startLogger(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			select {
			case <-ctx.Done():
				fmt.Println("Stopping Logger")
				return
			default:
				fmt.Printf("Operations per second: tx %d, rx %d, %s/s\n", c.stats.tx.Swap(0), c.stats.rx.Swap(0), internal.ByteString(c.stats.bytes.Swap(0)))
			}
		}
	}()
}

func (c *TcpClient) dial() (net.Conn, error) {
	var conn net.Conn
	var err error
	for range maxAttempts {
		conn, err = net.Dial("tcp", c.config.Address)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", c.config.Address, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	fmt.Printf("Connected to %s from %s\n", conn.RemoteAddr(), conn.LocalAddr())

	return conn, nil
}

func (c *TcpClient) writer(ctx context.Context, cancel context.CancelFunc, conn net.Conn, request []byte) func() error {
	return func() error {
		// Stop the reader when the writer is done, or the ErrGroup will wait forever.
		defer cancel()

		// Start off with the configured packet interval. This will be adjusted
		// based on the time it took to write to the socket.
		pause := c.config.Interval

		// Lock the goroutine to the current OS thread to prevent the runtime from
		// migrating and interrupting it as often. We're manually calling nanosleep,
		// bypassing the runtime's scheduler, to get somewhat accurate sleep
		// behaviour.
		runtime.LockOSThread()

		fmt.Println("Sending requests at a target interval of", c.config.Interval, "with timeout of", c.config.Timeout)

		for {
			// Immediately stop producing packets when the client is shutting down.
			select {
			case <-ctx.Done():
				fmt.Println("Writer shutting down")
				return nil
			default:
			}

			start := time.Now()

			if err := conn.SetWriteDeadline(start.Add(time.Second)); err != nil {
				return fmt.Errorf("set write deadline: %w", err)
			}

			n, err := conn.Write(request)
			if err != nil {
				return fmt.Errorf("conn write: %w", err)
			}
			if n != len(request) {
				return fmt.Errorf("short write: %d", n)
			}

			c.stats.tx.Add(1)
			pause = internal.Pause(c.config.Interval, pause, start)
		}
	}
}

func (c *TcpClient) reader(ctx context.Context, cancel context.CancelFunc, conn net.Conn, request []byte) func() error {
	return func() error {
		// Stop the reader when the writer is done, or the ErrGroup will wait forever.
		defer cancel()

		last := time.Now()
		reply := make([]byte, internal.MsgSize)
		for {
			if err := conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond)); err != nil {
				return fmt.Errorf("set read deadline: %w", err)
			}

			_, err := io.ReadFull(conn, reply)
			// Allow the reader to drain replies before shutting down instead of
			// closing the connection immediately. This reduces the chance of the
			// server seeing a connection reset, which causes red herrings in the
			// server logs when finding potential conn disruptions.
			if errors.Is(err, os.ErrDeadlineExceeded) {
				// Check if the deadline was exceeded as a consequence of shutting down.
				// Require the reader to be fully caught up at this point.
				select {
				case <-ctx.Done():
					if c.stats.tx.Load() == c.stats.rx.Load() {
						fmt.Println("Reader shutting down")
						return nil
					}
				default:
				}

				// Retry while the last reply was received within the timeout.
				if time.Since(last) <= c.config.Timeout {
					continue
				}

				return fmt.Errorf("no reply received within %v timeout: %w", c.config.Timeout, err)
			}
			if errors.Is(err, io.EOF) {
				fmt.Println("Server closed the connection")
				return nil
			}
			if err != nil {
				return fmt.Errorf("read reply: %w", err)
			}

			if !bytes.Equal(request, reply) {
				return fmt.Errorf("invalid reply(%v) to request(%v)", reply, request)
			}

			last = time.Now()
			c.stats.rx.Add(1)
			c.stats.bytes.Add(internal.MsgSize)

			// Check if we're shutting down and reader fully caught up to the writer,
			// for a fast exit.
			select {
			case <-ctx.Done():
				if c.stats.tx.Load() == c.stats.rx.Load() {
					fmt.Println("Reader shutting down")
					return nil
				}
			default:
			}
		}
	}
}
