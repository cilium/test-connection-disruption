package main

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

const MSG_SIZE = 256 // should be synced with the client
const WRITE_TIME_OUT = 5 * time.Second

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

type Server interface {
	Start()
}

type TCPServer struct {
	addr string
}

func (s TCPServer) Start() {
	listen, err := net.Listen("tcp", s.addr)
	panicOnErr("net.Listen", err)
	defer listen.Close()

	fmt.Printf("Listening on %s...\n", s.addr)

	for {
		conn, err := listen.Accept()
		panicOnErr("Accept", err)
		go read(conn)
	}
}

type UDPServer struct {
	addr string
}

func (s UDPServer) Start() {
	pc, err := net.ListenPacket("udp", s.addr)
	panicOnErr("net.Listen", err)
	fmt.Printf("UDP server listening on %s\n", pc.LocalAddr().String())
	defer pc.Close()

	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		for {
			buf := make([]byte, MSG_SIZE)
			n, addr, err := pc.ReadFrom(buf)
			panicOnErr("ReadFrom", err)
			// fmt.Println("Received ", string(buf[0:n]), " from ", addr)

			deadline := time.Now().Add(WRITE_TIME_OUT)
			err = pc.SetWriteDeadline(deadline)
			panicOnErr("SetWriteDeadline", err)

			n, err = pc.WriteTo(buf[:n], addr)
			panicOnErr("WriteTo", err)
		}
	}()

	wg.Wait()
}

func read(conn net.Conn) {
	fmt.Println("New connection from", conn.RemoteAddr())
	buf := make([]byte, MSG_SIZE)
	for {
		_, err := io.ReadFull(conn, buf)
		panicOnErr("io.ReadFull", err)
		_, err = conn.Write(buf)
		panicOnErr("io.Write", err)
	}
}

func main() {
	port := os.Args[1]
	proto := os.Args[2]
	addr := ":" + port

	var server Server

	switch proto {

	case "tcp":
		server = TCPServer{
			addr: addr,
		}

	case "udp":
		server = UDPServer{
			addr: addr,
		}
	default:
		panic(fmt.Sprintf("unknown protocol %s", proto))
	}

	for {
		server.Start()
	}
}
