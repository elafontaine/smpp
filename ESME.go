package smpp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"
)

var (
	DebugSmppLogger   *log.Logger
	WarningSmppLogger *log.Logger
	InfoSmppLogger    *log.Logger
	ErrorSmppLogger   *log.Logger
)

type ESME struct {
	clientSocket     net.Conn
	state            *State
	sequenceNumber   int32
	closeChan        chan bool
	commandFunctions map[string]func(*ESME, PDU) error
}

const (
	BOUND_TX  = "BOUND_TX"
	BOUND_RX  = "BOUND_RX"
	BOUND_TRX = "BOUND_TRX"
	OPEN      = "OPEN"
	CLOSED    = "CLOSED"
)

func InstantiateEsme(serverAddress net.Addr, connType string) (esme *ESME, err error) {
	clientSocket, err := net.Dial(connType, serverAddress.String())
	if err != nil {
		return nil, err
	}
	return NewEsme(&clientSocket), nil
}

func NewEsme(clientSocket *net.Conn) (e *ESME) {
	e = &ESME{
		*clientSocket,
		NewESMEState(OPEN),
		0,
		make(chan bool),
		map[string]func(*ESME, PDU) error{},
	}
	registerStandardBehaviours(e)
	return e
}

func registerStandardBehaviours(e *ESME) {
	e.commandFunctions["enquire_link"] = handleEnquiryLinkPduReceived
	e.commandFunctions["submit_sm"] = handleSubmitSmPduReceived
	e.commandFunctions["deliver_sm"] = handleDeliverSmPduReceived
}

func (e *ESME) Close() {
	e.clientSocket.Close()
	e.state.Close()
}

func (e *ESME) getEsmeState() string {
	return e.state.GetState()
}

func (e *ESME) bindTransmitter(systemID, password string) error {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	_, err := e.Send(&pdu)
	return err
}

func (e *ESME) BindTransmitter2(systemID, password string) (resp *PDU, err error) {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	_, err = e.Send(&pdu)
	if err != nil {
		return nil, err
	}
	resp, err = waitForBindResponse(e)
	return resp, err
}

func (e *ESME) bindTransceiver(systemID, password string) error {
	pdu := NewBindTransceiver().WithSystemId(systemID).WithPassword(password)
	_, err := e.Send(&pdu)
	return err
}

func (e *ESME) bindReceiver(systemID, password string) error {
	pdu := NewBindReceiver().WithSystemId(systemID).WithPassword(password)
	_, err := e.Send(&pdu)
	return err
}

func (e *ESME) Send(pdu *PDU) (seq_num int, err error) {
	seq_num = pdu.header.sequenceNumber
	if pdu.header.sequenceNumber == 0 {
		seq_num = int(atomic.AddInt32(&(e.sequenceNumber),1))
	}
	send_pdu := pdu.WithSequenceNumber(seq_num)
	expectedBytes, err := EncodePdu(send_pdu)
	if err != nil {
		return seq_num, err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return seq_num, err
}

func waitForBindResponse(e *ESME) (pdu *PDU, err error) {
	receivedPdu, err := e.receivePdu()
	if err != nil {
		return nil, err
	}
	pdu = &receivedPdu
	err = setESMEStateFromSMSCResponse(pdu, e)
	return pdu, err
}

func setESMEStateFromSMSCResponse(pdu *PDU, Esme *ESME) (err error) {
	if pdu.header.commandStatus == ESME_ROK {
		switch pdu.header.commandId {
		case "bind_receiver_resp":
			Esme.state.setState <- BOUND_RX

		case "bind_transmitter_resp":
			Esme.state.setState <- BOUND_TX

		case "bind_transceiver_resp":
			Esme.state.setState <- BOUND_TRX
		}
	} else {
		err = fmt.Errorf("The answer received wasn't OK or not the type we expected!")
	}
	return err
}

func (e *ESME) receivePdu() (PDU, error) {
	readBuf, LastError := readPduBytesFromConnection(e.clientSocket, time.Now().Add(1*time.Second))
	if LastError != nil {
		if errors.Is(LastError, io.EOF) || errors.Is(LastError, net.ErrClosed) {
			e.Close()
		}
		return PDU{}, fmt.Errorf("Couldn't read on a Connection: \n err =%v", LastError)
	}
	return ParsePdu(readBuf)
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

func (e *ESME) isTransmitterState() bool {
	currentState := e.getEsmeState()
	return (currentState == BOUND_TX || currentState == BOUND_TRX)
}

func (e *ESME) isReceiverState() bool {
	currentState := e.getEsmeState()
	return (currentState == BOUND_RX || currentState == BOUND_TRX)
}
