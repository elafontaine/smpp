package smpp

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"
)

type ESME struct {
	clientSocket   net.Conn
	state          string
	sequenceNumber int
}

const (
	BOUND_TX  = "BOUND_TX"
	BOUND_RX  = "BOUND_RX"
	BOUND_TRX = "BOUND_TRX"
	OPEN      = "OPEN"
	CLOSED    = "CLOSED"
)

func (e *ESME) Close() (err error) {
	err = e.clientSocket.Close()
	return err
}

func (e *ESME) bindTransmitter(systemID, password string) error {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	_, err := e.send(&pdu)
	return err
}

func (e *ESME) bindTransmitter2(systemID, password string) (resp *PDU, err error) {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	_, err = e.send(&pdu)
	if err != nil {
		return nil, err
	}
	resp, err = waitForBindResponse(e)
	return resp, err
}

func (e *ESME) bindTransceiver(systemID, password string) error {
	pdu := NewBindTransceiver().WithSystemId(systemID).WithPassword(password)
	_, err := e.send(&pdu)
	return err
}

func (e *ESME) bindReceiver(systemID, password string) error {
	pdu := NewBindReceiver().WithSystemId(systemID).WithPassword(password)
	_, err := e.send(&pdu)
	return err
}

func (e *ESME) send(pdu *PDU) (seq_num int, err error) { //Should we expect the bind_reveicer to return only when the bind is done and valid?
	seq_num = pdu.header.sequenceNumber
	if pdu.header.sequenceNumber == 0 {
		e.sequenceNumber++
		seq_num = e.sequenceNumber
	}
	send_pdu := pdu.WithSequenceNumber(seq_num)
	expectedBytes, err := EncodePdu(send_pdu)
	if err != nil {
		return seq_num, err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return seq_num, err
}

func (e *ESME) getConnectionState() (state string) {
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

func waitForBindResponse(Esme *ESME) (pdu *PDU, err error) {
	receivedBuf, err := readPduBytesFromConnection(Esme.clientSocket, time.Now().Add(1*time.Second))
	if err != nil {
		return nil, err
	}
	parsedPdu, err := ParsePdu(receivedBuf)
	pdu = &parsedPdu
	if err != nil {
		return nil, err
	}
	err = SetESMEStateFromSMSCResponse(pdu, Esme)
	return pdu, err
}

func SetESMEStateFromSMSCResponse(pdu *PDU, Esme *ESME) (err error) {
	if pdu.header.commandStatus == ESME_ROK {
		switch pdu.header.commandId {
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
