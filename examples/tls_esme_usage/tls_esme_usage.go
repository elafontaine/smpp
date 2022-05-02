package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"

	. "github.com/elafontaine/smpp"
)

func main() {
	connType := "tcp"
	cer, err := tls.LoadX509KeyPair("server.crt","server.key")
	if err != nil {
        fmt.Print(fmt.Errorf("%v", err))
        os.Exit(1)
    }
	config := tls.Config{
		Certificates: []tls.Certificate{cer},
		ServerName:                  "0.0.0.0",
		InsecureSkipVerify:          true,
	}
	serversocket, _ := tls.Listen(connType, "0.0.0.0:0", &config)
	smsc := NewSMSC(&serversocket, "MySystemId", "Password")

	esme_socket, err := net.Dial(connType,serversocket.Addr().String())
	esme_tls_socket := tls.Client(esme_socket,&config)
	if err != nil {
		fmt.Print(fmt.Errorf("%v", err))
		os.Exit(1)
	}

	esme := NewEsme(esme_tls_socket)
	if err != nil {
		fmt.Print(fmt.Errorf("Issue starting ESME connection to server! : %v", err))
		os.Exit(1)
	}
	smsc.Start()

	_, err = esme.BindTransmitter("MySystemId", "Password")
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't bind esme : %v", err))
		os.Exit(1)
	}

	submitSm := NewSubmitSM().
		WithDestinationAddress("5551234567").
		WithDataCoding(3).
		WithMessage("Hello, how are you today ?").
		WithSourceAddress("5557654321")
	_, err = esme.Send(&submitSm)
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't send on esme : %v", err))
		os.Exit(1)
	}
	fmt.Print("Reached the end of the program!")
}
