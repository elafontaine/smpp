package smpp

import "net"

type SMSC struct {
	listeningSocket net.Listener
	connections []net.Conn
}

func NewSMSC(listeningSocket net.Listener) (s *SMSC) {
	s = &SMSC{listeningSocket: listeningSocket, connections: []net.Conn{}}
	return s
}

func (s SMSC) Close() {
	s.listeningSocket.Close()
}
