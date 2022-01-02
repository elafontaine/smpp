package smpp

import (
	"fmt"
	"net"
	"sync/atomic"
)

type SMSC struct {
	listeningSocket net.Listener
	connections atomic.Value
	State string
}

func NewSMSC(listeningSocket *net.Listener) (s *SMSC) {

	s = &SMSC{listeningSocket: *listeningSocket, connections: atomic.Value{}}
	s.connections.Store([]net.Conn{})
	s.State = "LISTENING"
	return s
}

func (smsc *SMSC) AcceptNewConnectionFromSMSC() (err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		err = fmt.Errorf("couldn't establish connection on the server side successfully: %v", err)
		return err
	}
	old_connections := smsc.connections.Load().([]net.Conn)
	new_connections := append(old_connections, serverConnectionSocket)
	smsc.connections.Store(new_connections)
	return err
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.connections.Load().([]net.Conn))
}

func (s *SMSC) Close() {
	for _, conn := range s.connections.Load().([]net.Conn) {
		conn.Close()
	}
	s.listeningSocket.Close()
}
