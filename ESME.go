package smpp

import (
	"net"
)

type ESME struct {
	clientSocket net.Conn
	state        string
}

func (e ESME) Close() {
	e.clientSocket.Close()
}

func (e ESME) bindTransmiter(systemID, password string) error { //Should we expect the bind_transmitter to return only when the bind is done and valid?
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	expectedBytes, err := EncodePdu(pdu)
	if err != nil {
		return err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return err
}

func (e ESME) bindReceiver(systemID, password string) error { //Should we expect the bind_reveicer to return only when the bind is done and valid?
	pdu := NewBindReceiver().WithSystemId(systemID).WithPassword(password)
	expectedBytes, err := EncodePdu(pdu)
	if err != nil {
		return err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return err
}

func (e ESME) getConnectionState() (state string) {
	return e.state
}
