package http

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"io"
	nethttp "net/http"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
)

type HttpClient struct {
	config common.ClientConfig

	requestsCount atomic.Uint64
}

func NewHttpClient(config common.ClientConfig) *HttpClient {
	return &HttpClient{config: config, requestsCount: atomic.Uint64{}}
}

func (c *HttpClient) Run(ctx context.Context, _ context.CancelFunc) {
	fmt.Printf("Starting HTTP Client with config: %v\n", c.config)

	request := make([]byte, internal.MsgSize)
	_, err := rand.Read(request)
	internal.ErrExit("generate random payload", err)

	client := nethttp.Client{
		Transport: &nethttp.Transport{
			DisableKeepAlives:   false,
			MaxIdleConnsPerHost: 1,
		},
		Timeout: c.config.Timeout,
	}

	c.startLogger(ctx)
	internal.ErrExit("client", c.start(ctx, client, request))
}

func (c *HttpClient) start(ctx context.Context, client nethttp.Client, request []byte) error {
	pause := c.config.Interval
	runtime.LockOSThread()

	fmt.Println("Sending requests at a target interval of", c.config.Interval, "with timeout of", c.config.Timeout)
	common.MarkClientReady()
	for {
		// Immediately stop requests when the client is shutting down.
		select {
		case <-ctx.Done():
			fmt.Println("Client shutting down")
			return nil
		default:
		}

		start := time.Now()
		req, err := nethttp.NewRequest("GET", c.config.Address, bytes.NewReader(request))
		if err != nil {
			return fmt.Errorf("error creating request: %w", err)
		}

		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("http request failed: %w", err)
		}

		// Read the body entirely and close it to allow connection reuse (Keep-Alive).
		_, discardErr := io.Copy(io.Discard, resp.Body)
		internal.ErrExit("response body draining", discardErr)
		internal.ErrExit("response body close", resp.Body.Close())

		if resp.StatusCode != nethttp.StatusOK {
			return fmt.Errorf("http request non success status code %d", resp.StatusCode)
		}

		c.requestsCount.Add(1)
		internal.Pause(c.config.Interval, pause, start)
	}
}

func (c *HttpClient) startLogger(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			select {
			case <-ctx.Done():
				fmt.Println("Stopping Logger")
				return
			default:
				fmt.Printf("Requests per second: %d/s\n", c.requestsCount.Swap(0))
			}
		}
	}()
}
