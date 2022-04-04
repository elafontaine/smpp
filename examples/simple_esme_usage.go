package main

import (
	"fmt"
	"net"
	"os"

	. "github.com/elafontaine/smpp"
)

func main() {
	connType := "tcp"
	serversocket, _ := net.Listen(connType, "0.0.0.0:0")
	smsc := NewSMSC(&serversocket, "MySystemId", "Password")

	esme, err := InstantiateEsme(serversocket.Addr(), connType)
	if err != nil {
		fmt.Errorf("Issue starting ESME connection to server! : %s", err)
		os.Exit(1)
	}
	smsc.Start()

	_, err = esme.BindTransmitter("MySystemId", "Password")
	if err != nil {
		fmt.Errorf("Couldn't bind esme : %s", err)
		os.Exit(1)
	}

	submitSm := NewSubmitSM().WithDestinationAddress("5551234567").WithDataCoding(3).WithMessage("Hello, how are you today ?").WithSourceAddress("5557654321")
	_, err = esme.Send(&submitSm)
	if err != nil {
		fmt.Errorf("Couldn't send on esme : %s", err)
		os.Exit(1)
	}
	fmt.Print("Reached the end of the program!")
}
