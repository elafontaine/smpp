package smpp

import "net"

type ESME struct {
	clientSocket net.Conn
}

func (e ESME) Close() {
	e.clientSocket.Close()
}

func (e ESME) bindTransmiter(systemID, password string) error {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	expectedBytes, err := EncodePdu(pdu)
	if err != nil {
		return err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return err
}
