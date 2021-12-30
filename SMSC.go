package smpp

import (
	"fmt"
	"net"
)

type SMSC struct {
	listeningSocket net.Listener
	connections []net.Conn
}

func NewSMSC(listeningSocket *net.Listener) (s SMSC) {
	s = SMSC{listeningSocket: *listeningSocket, connections: []net.Conn{}}
	return s
}

func (smsc *SMSC) AcceptNewConnectionFromSMSC() (err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		err = fmt.Errorf("couldn't establish connection on the server side successfully: %v", err)
		return err
	}
	smsc.connections = append(smsc.connections, serverConnectionSocket)
	return err
}

func (s SMSC) GetNumberOfConnection() int {
	return len(s.connections)
}

func (s SMSC) Close() {
	for _, conn := range s.connections {
		conn.Close()
	}
	s.listeningSocket.Close()
}
