package smpp

import "net"

type ESME struct {
	clientSocket net.Conn
}

func (e ESME) Close() {
	e.clientSocket.Close()
}
