package main

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"reflect"
	"sync"
	"time"

	"github.com/google/martian/mitm"
	"github.com/google/martian/v3/h2"
)

type Config struct {
	ca                     *x509.Certificate
	capriv                 interface{}
	priv                   *rsa.PrivateKey
	keyID                  []byte
	validity               time.Duration
	org                    string
	h2Config               *h2.Config
	getCertificate         func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	roots                  *x509.CertPool
	skipVerify             bool
	handshakeErrorCallback func(*http.Request, error)

	certmu sync.RWMutex
	certs  map[string]*tls.Certificate
}

func main() {
	ca, priv, err := mitm.NewAuthority("martian.proxy", "Martian Authority", 24*time.Hour)
	if err != nil {
		fmt.Print("error NewAuthority")
	}

	c, err := mitm.NewConfig(ca, priv)
	if err != nil {
		fmt.Print("error NewConfig")
	}

	c.SetValidity(20 * time.Hour)
	c.SetOrganization("Test Organization")

	protos := []string{"http/1.1"}

	conf := c.TLS()
	if got := conf.NextProtos; !reflect.DeepEqual(got, protos) {
		fmt.Print("error")
	}
	if conf.InsecureSkipVerify {
		fmt.Print("error")
	}

	// Simulate a TLS connection without SNI.
	clientHello := &tls.ClientHelloInfo{
		ServerName: "",
	}

	if _, err := conf.GetCertificate(clientHello); err == nil {
		fmt.Print("error")
	}

	clientHello.ServerName = "example.com"

}
