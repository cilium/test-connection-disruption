package main

import (
	"fmt"
	"net"
	"os"
)

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

func read(conn net.Conn) {
	fmt.Println("New connection from", conn.RemoteAddr())
	buf := make([]byte, 128)
	for {
		_, err := conn.Read(buf)
		panicOnErr("conn.Read", err)
	}
}

func main() {
	port := os.Args[1]
	listen, err := net.Listen("tcp", ":"+port)
	panicOnErr("net.Listen", err)

	fmt.Printf("Listening on %s...\n", port)

	for {
		conn, err := listen.Accept()
		panicOnErr("Accept", err)
		go read(conn)
	}
}
