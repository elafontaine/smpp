package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"time"

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
	smsc := NewSMSC(serversocket, "MySystemId", "Password")

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

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err = esme.BindTransmitter(ctx, "MySystemId", "Password")
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't bind esme : %v", err))
		os.Exit(1)
	}

	submitSm := NewSubmitSM().
		WithDestinationAddress("5551234567").
		WithDataCoding(3).
		WithMessage("Hello, how are you today ?").
		WithSourceAddress("5557654321")
	resp, err := esme.Send(ctx, submitSm)
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't send on esme : %v", err))
		os.Exit(1)
	}
	fmt.Printf("submit_sm answered with message_id %v\n", resp.Body.MandatoryParameter["message_id"])
	fmt.Println("Reached the end of the program!")
}
