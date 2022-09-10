package main

import (
	"fmt"
	"net"
	"os"

	. "github.com/elafontaine/smpp"
)

func main() {
	// Server side preparations
	connType := "tcp"
	serversocket, _ := net.Listen(connType, "0.0.0.0:0")
	smsc := NewSMSC(&serversocket, "MySystemId", "Password")

	// Client side example (using the prepared server side)
	esme, err := InstantiateEsme(serversocket.Addr(), connType)
	if err != nil {
		fmt.Print(fmt.Errorf("Issue starting ESME connection to server! : %v", err))
		os.Exit(1)
	}
	smsc.Start()  // Only useful as we use a smsc server

	_, err = esme.BindTransmitter("MySystemId", "Password") // Currently skipping the response pdu check
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't bind esme : %v", err))
		os.Exit(1)
	}

	submitSm := NewSubmitSM().
		WithDestinationAddress("5551234567").
		WithDataCoding(3).
		WithMessage("Hello, how are you today ?").
		WithSourceAddress("5557654321")
	_, err = esme.Send(&submitSm) // skipping the response pdu checks
	if err != nil {
		fmt.Print(fmt.Errorf("Couldn't send on esme : %v", err))
		os.Exit(1)
	}
	fmt.Print("Reached the end of the program!")
}
