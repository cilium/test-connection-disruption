package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"sync/atomic"
	"time"

	flag "github.com/spf13/pflag"
)

const MSG_SIZE = 256 // should be synced with the server

var DispatchInterval time.Duration

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func main() {
	flag.DurationVar(&DispatchInterval, "dispatch-interval", 500*time.Millisecond, "TCP packet dispatch interval")
	flag.Parse()

	var (
		conn net.Conn
		err  error
	)
	addr := flag.Args()[0]

	for i := 0; i < 30; i++ {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", addr, err)
		time.Sleep(1 * time.Second)
	}
	panicOnErr("Failed to connect", err)
	fmt.Printf("Connected to %s from %s\n", conn.RemoteAddr(), conn.LocalAddr())
	file, err := os.Create("/tmp/client-ready")
	panicOnErr("Failed to create file", err)
	file.Close()

	request := make([]byte, MSG_SIZE)
	reply := make([]byte, MSG_SIZE)

	var count atomic.Int64
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			fmt.Printf("Operations per second: %d\n", count.Swap(0))
		}
	}()

	for {
		_, err := rand.Read(request)
		panicOnErr("rand.Read", err)

		panicOnErr("conn.SetWriteDeadline", conn.SetWriteDeadline(time.Now().Add(1*time.Second)))
		_, err = conn.Write(request)
		panicOnErr("conn.Write", err)

		panicOnErr("conn.SetReadDeadline", conn.SetReadDeadline(time.Now().Add(1*time.Second)))
		_, err = io.ReadFull(conn, reply)
		panicOnErr("io.ReadFull", err)

		if !bytes.Equal(request, reply) {
			panic(fmt.Sprintf("Invalid reply(%v) for request(%v)", reply, request))
		}

		count.Add(1)
		time.Sleep(DispatchInterval)
	}
}
