package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"runtime"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"
	"golang.org/x/sync/errgroup"

	"github.com/cilium/test-connection-disruption/internal"
)

const maxAttempts = 30

func init() {
	internal.ErrExit("being nice", internal.BeNice())
}

var stats struct {
	rx, tx atomic.Uint64
	bytes  atomic.Uint64
}

var args struct {
	addr     string
	interval time.Duration
	latency  time.Duration
}

func main() {
	flag.DurationVar(&args.interval, "dispatch-interval", 50*time.Millisecond, "TCP packet dispatch interval")
	flag.DurationVar(&args.latency, "latency", 250*time.Millisecond, "Maximum expected latency for the connection, used for setting read deadlines")
	flag.Parse()

	args.addr = flag.Arg(0)
	if args.addr == "" {
		flag.Usage()
		os.Exit(1)
	}

	// For backwards compatibility, clamp the interval to a minimum of 10ms to
	// avoid overloading resource-constrained CI machines where Cilium runs with
	// monitor aggregation disabled.
	if args.interval == 0 {
		args.interval = 10 * time.Millisecond
		fmt.Println("Zero interval changed to", args.interval, "for backwards compatibility.")
	}

	conn, err := dial()
	internal.ErrExit("dial remote", err)
	defer conn.Close()

	// Set up request payload.
	request := make([]byte, internal.MsgSize)
	_, err = rand.Read(request)
	internal.ErrExit("generate random payload", err)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	var eg errgroup.Group
	eg.Go(writer(ctx, cancel, conn, request))
	eg.Go(reader(ctx, cancel, conn, request))

	startLogger()

	ready()

	internal.ErrExit("Error in writer or reader", eg.Wait())
}

func dial() (net.Conn, error) {
	var conn net.Conn
	var err error
	for range maxAttempts {
		conn, err = net.Dial("tcp", args.addr)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", args.addr, err)
		time.Sleep(time.Second)
	}
	if err != nil {
		return nil, err
	}

	fmt.Printf("Connected to %s from %s\n", conn.RemoteAddr(), conn.LocalAddr())

	return conn, nil
}

func startLogger() {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			fmt.Printf("Operations per second: tx %d, rx %d, %s/s\n", stats.tx.Swap(0), stats.rx.Swap(0), internal.ByteString(stats.bytes.Swap(0)))
		}
	}()
}

func writer(ctx context.Context, cancel context.CancelFunc, conn net.Conn, request []byte) func() error {
	return func() error {
		// Stop the reader when the writer is done, or the ErrGroup will wait forever.
		defer cancel()

		// Start off with the configured packet interval. This will be adjusted
		// based on the time it took to write to the socket.
		pause := args.interval

		// Lock the goroutine to the current OS thread to prevent the runtime from
		// migrating and interrupting it as often. We're manually calling nanosleep,
		// bypassing the runtime's scheduler, to get somewhat accurate sleep
		// behaviour.
		runtime.LockOSThread()

		fmt.Println("Sending requests at a target interval of", args.interval, "with max expected latency of", args.latency)

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

			stats.tx.Add(1)

			// Sleep for the duration determined during the previous round. Use a
			// direct call to nanosleep(2) since the regular [time.Sleep] is
			// implemented by the Go runtime and gets coalesced to reduce syscall
			// overhead. This leads to wildly unexpected sleep durations.
			internal.Sleep(pause)

			// Adjust the sleep interval for the next cycle based on the time it took
			// to write to the socket and when the OS scheduler woke us up.
			delta := args.interval - time.Since(start)

			// Smoothen the approach to the target interval by adjusting the pause
			// interval by half the delta.
			pause += (delta / 2)

			// Ensure pause stays within bounds. On a permanent deficit, it would
			// run negative and overflow at some point.
			pause = min(max(pause, -args.interval), args.interval)
		}
	}
}

func reader(ctx context.Context, cancel context.CancelFunc, conn net.Conn, request []byte) func() error {
	return func() error {
		// Stop the reader when the writer is done, or the ErrGroup will wait forever.
		defer cancel()

		reply := make([]byte, internal.MsgSize)
		for {
			deadline := args.interval + args.latency
			if err := conn.SetReadDeadline(time.Now().Add(deadline)); err != nil {
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
					if stats.tx.Load() == stats.rx.Load() {
						fmt.Println("Reader shutting down")
						return nil
					}
				default:
				}

				return fmt.Errorf("no reply received within %v deadline: %w (%d tx, %d rx)", deadline, err, stats.tx.Load(), stats.rx.Load())
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

			stats.rx.Add(1)
			stats.bytes.Add(internal.MsgSize)

			// Check if we're shutting down and reader fully caught up to the writer,
			// for a fast exit.
			select {
			case <-ctx.Done():
				if stats.tx.Load() == stats.rx.Load() {
					fmt.Println("Reader shutting down")
					return nil
				}
			default:
			}
		}
	}
}

func ready() {
	file, err := os.Create("/tmp/client-ready")
	internal.ErrExit("create ready file", err)
	internal.ErrExit("close ready file", file.Close())
}
