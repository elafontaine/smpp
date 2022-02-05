package smpp

import (
	"fmt"
	"net"
	"sync/atomic"
)

const (
	LISTENING = "LISTENING"
)

type SMSC struct {
	listeningSocket net.Listener
	ESMEs           atomic.Value
	State           string
}

func NewSMSC(listeningSocket *net.Listener) (s *SMSC) {
	s = &SMSC{listeningSocket: *listeningSocket, ESMEs: atomic.Value{}}
	s.ESMEs.Store([]ESME{})
	s.State = LISTENING
	return s
}

func (smsc *SMSC) AcceptNewConnectionFromSMSC() (conn *net.Conn, err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	conn = &serverConnectionSocket

	e := ESME{
		clientSocket:   serverConnectionSocket,
		state:          OPEN,
		sequenceNumber: 0,
	}

	if err != nil {
		err = fmt.Errorf("couldn't establish connection on the server side successfully: %v", err)
		return nil, err
	}
	old_connections := smsc.ESMEs.Load().([]ESME)
	new_connections := append(old_connections, e)
	smsc.ESMEs.Store(new_connections)
	return conn, err
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.ESMEs.Load().([]ESME))
}

func (s *SMSC) Close() {
	for _, conn := range s.ESMEs.Load().([]ESME) {
		conn.Close()
	}
	s.listeningSocket.Close()
}
