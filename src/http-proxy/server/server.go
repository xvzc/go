package server

import (
	"crypto/rand"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/soheilhy/cmux"
)

func handleTunneling(w http.ResponseWriter, r *http.Request) {
	fmt.Println("HTTPS Requested")
	dest_conn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)

	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	hijacker, ok := w.(http.Hijacker)

	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}

	client_conn, _, err := hijacker.Hijack()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
	}

	go transfer(dest_conn, client_conn)
	go transfer(client_conn, dest_conn)
}

func transfer(destination io.WriteCloser, source io.ReadCloser) {
	defer destination.Close()
	defer source.Close()

	io.Copy(destination, source)
}

func handleHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println("HTTP Requested")
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	copyHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func copyHeader(dst, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}

func serveHTTP(listener net.Listener) {
	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleHTTP(w, r)
		}),
		// Disable HTTP/2.
		// TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if err := server.Serve(listener); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

func serveHTTPS(listener net.Listener) {
	// Load certificates.
	certificate, err := tls.LoadX509KeyPair("cert.pem", "key.pem")
	if err != nil {
		log.Panic(err)
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{certificate},
		Rand:         rand.Reader,
	}

	// Create TLS listener.
	tlsl := tls.NewListener(listener, config)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleTunneling(w, r)
		}),
		// Disable HTTP/2.
		// TLSNextProto: make(map[string]func(*http.Server, *tls.Conn, http.Handler)),
	}

	if err := server.Serve(tlsl); err != cmux.ErrListenerClosed {
		panic(err)
	}
}

// This is an example for serving HTTP and HTTPS on the same port.
func Start(port string) {
	// Create the TCP listener.
	listener, err := net.Listen("tcp", "127.0.0.1:"+port)
	if err != nil {
		log.Panic(err)
	}

	// Create a mux.
	m := cmux.New(listener)

	// We first match on HTTP 1.1 methods.
	httpListener := m.Match(cmux.HTTP1Fast())

	// If not matched, we assume that its TLS.
	//
	// Note that you can take this listener, do TLS handshake and
	// create another mux to multiplex the connections over TLS.
	tlsListener := m.Match(cmux.Any())

	go serveHTTP(httpListener)
	go serveHTTPS(tlsListener)

	if err := m.Serve(); !strings.Contains(err.Error(), "use of closed network connection") {
		panic(err)
	}
}
