package smpp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"math/big"
	"net"
	"path/filepath"
	"testing"
	"time"
)

// The SMSC takes any net.Listener and the ESME any net.Conn: TLS and Unix sockets work the same
// way as plain TCP.
func TestSmscAndEsmeOverOtherTransports(t *testing.T) {
	serverTLS, clientTLS := selfSignedTLS(t)
	tests := []struct {
		name    string
		listen  func(t *testing.T) (net.Listener, error)
		connect func(t *testing.T, address string) (*ESME, error)
	}{
		{
			"unix socket",
			func(t *testing.T) (net.Listener, error) {
				return net.Listen("unix", filepath.Join(t.TempDir(), "smsc.sock"))
			},
			func(t *testing.T, address string) (*ESME, error) { return InstantiateEsme("unix", address) },
		},
		{
			"TLS",
			func(t *testing.T) (net.Listener, error) { return tls.Listen("tcp", "127.0.0.1:0", serverTLS) },
			func(t *testing.T, address string) (*ESME, error) {
				conn, err := tls.Dial("tcp", address, clientTLS)
				if err != nil {
					return nil, err
				}
				return NewEsme(conn), nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			listener, err := tt.listen(t)
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			smsc := NewSMSC(listener, validSystemID, validPassword)
			smsc.Start()
			esme, err := tt.connect(t, listener.Addr().String())
			if err != nil {
				t.Fatalf("connect: %v", err)
			}
			defer CloseAndAssertClean(smsc, esme, t)

			if _, err := esme.BindTransmitter(contextWithin(t, 2*time.Second), validSystemID, validPassword); err != nil {
				t.Fatalf("BindTransmitter() error = %v", err)
			}
			resp, err := esme.Send(contextWithin(t, 2*time.Second), NewEnquireLink())
			if err != nil || resp.Header.CommandId != "enquire_link_resp" {
				t.Errorf("Send(enquire_link) = %v, %v; want an enquire_link_resp", resp.Header, err)
			}
		})
	}
}

// selfSignedTLS returns a server config with a fresh self-signed certificate for 127.0.0.1, and
// a client config that trusts it.
func selfSignedTLS(t *testing.T) (server, client *tls.Config) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating a key: %v", err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		IPAddresses:  []net.IP{net.IPv4(127, 0, 0, 1)},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating the certificate: %v", err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parsing the certificate: %v", err)
	}
	trusted := x509.NewCertPool()
	trusted.AddCert(certificate)
	server = &tls.Config{Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}}}
	client = &tls.Config{RootCAs: trusted}
	return server, client
}
