package smpp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
)

const (
	connhost = "localhost"
	connport = "0"
	connType = "tcp"
)

func TestServerInstantiationAndConnectClient(t *testing.T) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	smsc, err := StartSmscSimulatorServer(serverSocket)
	defer smsc.Close()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()
	expectedBytes := []byte("hello")
	Esme.clientSocket.Write(expectedBytes)
	readBuf, err := AcceptNewConnectionAndReadFromSMSC(smsc)
	if err != nil {
		t.Errorf("Couldn't read on a newly established Connection: %v", err)
	}
	if bytes.Equal(readBuf, expectedBytes) {
		t.Errorf("We didn't receive what we sent")
	}
}

func AcceptNewConnectionAndReadFromSMSC(smsc SMSC) (readBuf []byte, err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		err = errors.New(fmt.Sprintf("Couldn't establish connection on the server side successfully: %v", err))
		return nil, err
	}
	readBuf = make([]byte, 4096)
	_, err = serverConnectionSocket.Read(readBuf)
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
