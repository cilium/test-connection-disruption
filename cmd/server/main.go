package main

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
)

const MSG_SIZE = 256 // should be synced with the client

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func read(conn net.Conn) {
	fmt.Println("New connection from", conn.RemoteAddr())
	buf := make([]byte, MSG_SIZE)
	for {
		_, err := io.ReadFull(conn, buf)
		if errors.Is(err, io.EOF) {
			fmt.Println("Closed connection from", conn.RemoteAddr())
			return
		}
		panicOnErr("io.ReadFull", err)
		_, err = conn.Write(buf)
		if errors.Is(err, io.EOF) {
			fmt.Println("Closed connection to", conn.RemoteAddr())
			return
		}
		panicOnErr("io.Write", err)
	}
}

func main() {
	port := os.Args[1]
	listen, err := net.Listen("tcp", ":"+port)
	panicOnErr("net.Listen", err)

	file, err := os.Create("/tmp/server-ready")
	panicOnErr("Failed to create file", err)
	file.Close()

	fmt.Printf("Listening on %s...\n", port)

	for {
		conn, err := listen.Accept()
		panicOnErr("Accept", err)
		go read(conn)
	}
}
