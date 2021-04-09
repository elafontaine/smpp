package smpp

import "net"

type SMSC struct {
	listeningSocket net.Listener
}

func (s SMSC) Close() {
	s.listeningSocket.Close()
}
