package main

import (
	"fmt"
	"net"
	"os"
	"time"
)

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func main() {
	var (
		conn net.Conn
		err  error
	)
	addr := os.Args[1]

	for i := 0; i < 30; i++ {
		conn, err = net.Dial("tcp", addr)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", addr, err)
		time.Sleep(1 * time.Second)
	}
	panicOnErr("Failed to connect", err)
	fmt.Printf("Connected to %s\n", conn.RemoteAddr())

	msg := []byte("hello world")
	for {
		_, err = conn.Write(msg)
		panicOnErr("conn.Write", err)
		time.Sleep(500 * time.Millisecond)
	}
}
