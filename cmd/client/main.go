package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

const MSG_SIZE = 256 // should be synced with the server

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
	file, err := os.Create("/tmp/client-ready")
	panicOnErr("Failed to create file", err)
	file.Close()

	request := make([]byte, MSG_SIZE)
	reply := make([]byte, MSG_SIZE)

	for {
		_, err := rand.Read(request)
		panicOnErr("rand.Read", err)
		writeDone := make(chan struct{})
		go func() {
			_, err = conn.Write(request)
			panicOnErr("conn.Write", err)
			close(writeDone)
		}()
		select {
		case <-writeDone:
		case <-time.After(1 * time.Second):
			panic("conn.Write timed out")
		}

		readDone := make(chan struct{})
		go func() {
			_, err = io.ReadFull(conn, reply)
			panicOnErr("io.ReadFull", err)
			close(readDone)
		}()
		select {
		case <-readDone:
		case <-time.After(1 * time.Second):
			panic("conn.Read timed out")
		}
		if bytes.Compare(request, reply) != 0 {
			panic(fmt.Sprintf("Invalid reply(%v) for request(%v)", reply, request))
		}

		time.Sleep(500 * time.Millisecond)
	}
}
