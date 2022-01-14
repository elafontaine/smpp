package smpp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type ESME struct {
	clientSocket net.Conn
	state        string
}

const BOUND_TX = "BOUND_TX"
const BOUND_RX = "BOUND_RX"
const BOUND_TRX = "BOUND_TRX"

func (e ESME) Close() {
	e.clientSocket.Close()
}

func (e ESME) bindTransmiter(systemID, password string) error { //Should we expect the bind_transmitter to return only when the bind is done and valid?
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	err := e.bind(&pdu)
	return err
}

func (e ESME) bindReceiver(systemID, password string) error { //Should we expect the bind_reveicer to return only when the bind is done and valid?
	pdu := NewBindReceiver().WithSystemId(systemID).WithPassword(password)
	err := e.bind(&pdu)
	return err
}

func (e ESME) bind( bindPdu *PDU) error { //Should we expect the bind_reveicer to return only when the bind is done and valid?
	expectedBytes, err := EncodePdu(*bindPdu)
	if err != nil {
		return err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return err
}

func (e ESME) getConnectionState() (state string) {
	return e.state
}


func readPduBytesFromConnection(ConnectionSocket net.Conn, timeout time.Time) ([]byte, error) {
	buffer := bytes.Buffer{}
	err := ConnectionSocket.SetDeadline(timeout)
	if err != nil {
		return nil, err
	}
	readLengthBuffer := make([]byte, 4)
	_, err = ConnectionSocket.Read(readLengthBuffer)
	if err == nil {
		length := int(binary.BigEndian.Uint32(readLengthBuffer))
		if length <= 4 {
			return nil, fmt.Errorf("Received malformed packet : %v", readLengthBuffer)
		}
		readBuf := make([]byte, length-4)
		_, err = ConnectionSocket.Read(readBuf)
		buffer.Write(readLengthBuffer)
		buffer.Write(readBuf)
	}

	return buffer.Bytes(), err
}

func handleBindResponse(Esme *ESME) error {
	esmeReceivedBuf, err := readPduBytesFromConnection(Esme.clientSocket, time.Now().Add(1*time.Second))
	if err != nil {
		return err
	}
	resp, err := ParsePdu(esmeReceivedBuf)
	if err != nil {
		return err
	}
	if resp.header.commandStatus == ESME_ROK {
		switch resp.header.commandId {
		case "bind_receiver_resp":
			Esme.state = BOUND_RX

		case "bind_transmitter_resp":
			Esme.state = BOUND_TX

		case "bind_transceiver_resp":
			Esme.state = BOUND_TRX
		}

	} else {
		err = fmt.Errorf("The answer received wasn't OK or not the type we expected!")
	}
	return err
}