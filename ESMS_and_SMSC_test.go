package smpp

import (
	"bytes"
	"fmt"
	"net"
	"testing"
	"time"
)

const (
	connhost = "localhost"
	connport = "0"
	connType = "tcp"
)

func TestServerInstantiationAndConnectClient(t *testing.T) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	if err != nil {
		t.Errorf("couldn't start listening socket: %v", err)
	}
	smsc, err := StartSmscSimulatorServer(serverSocket)
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	defer smsc.Close()

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()
	err2 := Esme.bindTransmiter("SystemId", "Password")
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	readBuf, err := AcceptNewConnectionAndReadFromSMSC(smsc)
	if err != nil {
		t.Errorf("Couldn't read on a newly established Connection: %v", err)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId("SystemId").WithPassword("Password"))
	tempReadBuf := readBuf[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
}

func AcceptNewConnectionAndReadFromSMSC(smsc SMSC) (readBuf []byte, err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		err = fmt.Errorf("Couldn't establish connection on the server side successfully: %v", err)
		return nil, err
	}
	readBuf = make([]byte, 4096)
	err = serverConnectionSocket.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = serverConnectionSocket.Read(readBuf)
	if err != nil {
		return nil, err
	}
	return readBuf, err
}

func InstantiateEsme(serverAddress net.Addr) (esme ESME, err error) {
	clientSocket, err := net.Dial(connType, serverAddress.String())
	esme = ESME{clientSocket}
	return esme, err
}

func StartSmscSimulatorServer(serverSocket net.Listener) (smsc SMSC, err error) {
	smsc = SMSC{serverSocket}
	return smsc, err
}
