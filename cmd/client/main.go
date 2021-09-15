package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"strconv"
	"time"
)

const MSG_SIZE = 256 // should be synced with the server

func panicOnErr(ctx string, err error) {
	if err != nil {
		panic(fmt.Sprintf("%s: %s", ctx, err))
	}
}

type Client interface {
	Run()
}

type TCPClient struct {
	addr string
	conn net.Conn
}

func (c TCPClient) Run() {
	var err error
	conn := c.conn
	addr := c.addr

	for i := 0; i < 30; i++ {
		conn, err = net.Dial("tcp", c.addr)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", addr, err)
		time.Sleep(1 * time.Second)
	}
	panicOnErr("Failed to connect", err)
	fmt.Printf("Connected to %s\n", conn.RemoteAddr())
	defer conn.Close()

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

type UDPClient struct {
	addr string
	conn *net.UDPConn
}

func getRequest() []byte {
	rand.Seed(time.Now().UnixNano())
	return []byte("ping" + strconv.Itoa(rand.Intn(0xffff)))
}

func (c UDPClient) Run() {
	conn := c.conn
	addr := c.addr

	udpAddr, err := net.ResolveUDPAddr("udp", c.addr)
	panicOnErr("ResolveUDPAddr", err)

	for i := 0; i < 30; i++ {
		conn, err = net.DialUDP("udp", nil, udpAddr)
		if err == nil {
			break
		}
		fmt.Printf("Failed to connect to %s due to %s. Retrying...\n", addr, err)
		time.Sleep(1 * time.Second)
	}
	panicOnErr("Failed to connect", err)
	fmt.Printf("Connected to %s\n", conn.RemoteAddr())

	// Closes the underlying file descriptor associated with the,
	// socket so that it no longer refers to any file.
	defer conn.Close()

	reply := make([]byte, MSG_SIZE)

	for {
		writeDone := make(chan struct{})
		request := getRequest()
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
		n, _, err := conn.ReadFromUDP(reply)
		panicOnErr("ReadFromUDP", err)
		close(readDone)
		select {
		case <-readDone:
		case <-time.After(1 * time.Second):
			panic("conn.Read timed out")
		}
		if bytes.Compare(request, reply[:n]) != 0 {
			panic(fmt.Sprintf("Invalid reply(%v) for request(%v)", reply, request))
		}
		fmt.Println("client received: ", string(reply[:n]))

		time.Sleep(500 * time.Millisecond)
	}
}

func main() {
	addr := os.Args[1]
	proto := os.Args[2]

	var client Client

	switch proto {

	case "tcp":
		client = TCPClient{
			addr: addr,
		}

	case "udp":
		client = UDPClient{
			addr: addr,
		}
	default:
		panic(fmt.Sprintf("unknown protocol %s", proto))
	}

	for {
		client.Run()
	}

}
