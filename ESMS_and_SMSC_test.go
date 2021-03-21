package smpp

import (
	"bytes"
	"net"
	"testing"
)

const (
	connhost = "localhost"
	connport = "0"
	connType = "tcp"
)

func TestServerInstantiationAndGetRandomPort(t *testing.T) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	if err != nil {
		t.Errorf("couldn't start server successfully %v", err)
	}
	defer serverSocket.Close()
}
func TestServerInstantiationAndConnectClient(t *testing.T) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	defer serverSocket.Close()

	clientSocket, err := net.Dial(connType, serverSocket.Addr().String())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer clientSocket.Close()

	expectedBytes := []byte("hello")
	clientSocket.Write(expectedBytes)
	serverConnectionSocket, err := serverSocket.Accept()
	if err != nil {
		t.Errorf("Couldn't establish connection on the server side successfully: %v", err)
	}
	defer serverConnectionSocket.Close()
	readBuf := make([]byte, 4096)
	_, err = serverConnectionSocket.Read(readBuf)
	if err != nil {
		t.Errorf("Couldn't read from established Connection: %v", err)
	}
	if bytes.Equal(readBuf, expectedBytes) {
		t.Errorf("We didn't receive what we sent")
	}
}
