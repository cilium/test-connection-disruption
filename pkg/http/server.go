package http

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	nethttp "net/http"
	"os"

	"github.com/cilium/test-connection-disruption/internal"
	"github.com/cilium/test-connection-disruption/pkg/common"
)

type HttpServer struct {
	config common.ServerConfig
}

func NewHttpServer(config common.ServerConfig) *HttpServer {
	return &HttpServer{config: config}
}

func (s *HttpServer) Run(ctx context.Context, _ context.CancelFunc) {
	fmt.Printf("Starting HTTP Server with config: %#v\n", s.config)

	listener, err := net.Listen("tcp", ":"+s.config.Address)
	internal.ErrExit("TcpServer listen", err)
	defer listener.Close()

	nethttp.HandleFunc("/", s.httpEcho)
	server := &nethttp.Server{
		Handler: http.DefaultServeMux,
	}
	waitCtx, waitCancel := context.WithCancel(context.Background())
	go func() {
		defer waitCancel()

		<-ctx.Done()
		fmt.Println("Closing HTTP Server")
		internal.ErrExit("http server shutdown", server.Shutdown(waitCtx))
	}()

	common.MarkServerReady()
	err = server.Serve(listener)
	internal.ErrExit("HttpServer listener", err)

	<-waitCtx.Done()
}

func (s *HttpServer) httpEcho(w nethttp.ResponseWriter, r *nethttp.Request) {
	fmt.Printf("New HTTP request: %s (%s)\n", r.URL.String(), r.Method)

	w.Header().Set("X-Echo-Server", "Test-Connection-Disruption")
	w.WriteHeader(nethttp.StatusOK)

	_, err := io.Copy(w, r.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying request body: %s\n", r.URL)
		return
	}

	r.Body.Close()
}
