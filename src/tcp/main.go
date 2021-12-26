package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"

	"github.com/babolivier/go-doh-client"
)

var resolver doh.Resolver

func SecondChunk(b *[]byte) string {
	start := 0
	for i := 0; i < len(*b); i++ {
		if (*b)[i] == ' ' {
			start = i + 1
			break
		}
	}

	end := 0
	for i := start; i < len(*b); i++ {
		if (*b)[i] == ':' {
			end = i
			break
		}
	}

	return string(*b)[start+1 : end]
}

func handleRequest(conn net.Conn) {
	buf := make([]byte, 0, 4096) // big buffer
	tmp := make([]byte, 256)     // using small tmo buffer for demonstrating

	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		buf = append(buf, tmp[:n]...)

	}
	fmt.Println(string(tmp))

	a, _, err := resolver.LookupA(SecondChunk(&buf))
	if err != nil {
		panic(err)
	}
	ip := a[0].IP4

	server_conn, err := net.Dial("tcp", ip+":443")
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error Dialing to server")
	}
	defer server_conn.Close()

	_, write_err := server_conn.Write(buf)
	if write_err != nil {
		log.Fatal(write_err)
	}
	log.Println("sent to server")
}

func handleHttpsFromServer(conn net.Conn) {
	buf := make([]byte, 0, 4096) // big buffer
	tmp := make([]byte, 256)     // using small tmo buffer for demonstrating

	for {
		n, err := conn.Read(tmp)
		if err != nil {
			if err != io.EOF {
				fmt.Println("read error:", err)
			}
			break
		}
		buf = append(buf, tmp[:n]...)

	}
	fmt.Println(string(tmp))
	server_conn, err := net.Dial("tcp", "127.0.0.1:443")
	if err != nil {
		fmt.Println(err)
		log.Fatal("Error Dialing to server")
	}
	defer server_conn.Close()

	_, write_err := server_conn.Write(buf)
	if write_err != nil {
		log.Fatal(write_err)
	}
}

func main() {
	resolver = doh.Resolver{
		Host:  "8.8.8.8",
		Class: doh.IN,
	}

	clientListener, err := net.Listen("tcp", ":8080")
	clientListener.(*net.TCPListener).SetDeadline(time.Now().Add(time.Second * 2))
	if err != nil {
		log.Println(err)
	}
	defer clientListener.Close()

	serverHttpListener, err := net.Listen("tcp", ":80")
	if err != nil {
		log.Println(err)
	}
	defer serverHttpListener.Close()

	serverHttpsListener, err := net.Listen("tcp", ":443")
	if err != nil {
		log.Println(err)
	}
	defer serverHttpsListener.Close()

	fmt.Println("Litener created")

	for {
		conn, err := clientListener.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		defer conn.Close()

		go handleRequest(conn)

		server_https_conn, err := serverHttpsListener.Accept()
		if err != nil {
			log.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		defer server_https_conn.Close()

		go handleHttpsFromServer(server_https_conn)
	}
}
