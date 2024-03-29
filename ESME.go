package smpp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	DebugSmppLogger   *log.Logger
	WarningSmppLogger *log.Logger
	InfoSmppLogger    *log.Logger
	ErrorSmppLogger   *log.Logger
)

// ESME is the client side of the SMPP protocol.  Users should be
// managing their ESMEs to connect to an SMPP server (SMSC).
type ESME struct {
	clientSocket     net.Conn
	state            *State
	sequenceNumber   int32
	CommandFunctions map[string]func(*ESME, PDU) error
	defaults         map[string]interface{}
	wg               sync.WaitGroup
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
	return NewEsme(clientSocket), nil
}

func NewEsme(clientSocket net.Conn) (e *ESME) {
	e = &ESME{
		clientSocket,
		NewESMEState(OPEN),
		0,
		map[string]func(*ESME, PDU) error{},
		map[string]interface{}{},
		sync.WaitGroup{},
	}
	registerStandardBehaviours(e)
	return e
}

func (e *ESME) Close() {
	e.clientSocket.Close()
	e.state.Close()
	e.wg.Wait()
}

func (e *ESME) SetDefaults(defaults map[string]interface{}) {
	if defaults == nil {
		panic("Setting the ESME defaults with a nil value!")
	}
	e.defaults = defaults
}

func (e *ESME) GetEsmeState() string {
	return e.state.GetState()
}

func (e *ESME) BindTransmitter(systemID, password string) (resp *PDU, err error) {
	pdu := NewBindTransmitter().WithSystemId(systemID).WithPassword(password)
	return e.bindWithSmsc(pdu)
}

func (e *ESME) BindAsTransmitter() (err error) {
	pdu := NewBindTransmitter().WithDefaults(e.defaults)
	_, err = e.bindWithSmsc(pdu)
	return err
}

func (e *ESME) BindTransceiver(systemID, password string) (resp *PDU, err error) {
	pdu := NewBindTransceiver().WithSystemId(systemID).WithPassword(password)
	return e.bindWithSmsc(pdu)
}

func (e *ESME) BindReceiver(systemID, password string) (resp *PDU, err error) {
	pdu := NewBindReceiver().WithSystemId(systemID).WithPassword(password)
	return e.bindWithSmsc(pdu)
}

func (e *ESME) Send(pdu *PDU) (seq_num int, err error) {
	if pdu.Header.SequenceNumber == 0 {
		seq_num = int(atomic.AddInt32(&(e.sequenceNumber), 1))
	} else {
		seq_num = pdu.Header.SequenceNumber
	}
	send_pdu := pdu.WithSequenceNumber(seq_num)
	expectedBytes, err := EncodePdu(send_pdu)
	if err != nil {
		return seq_num, err
	}
	_, err = e.clientSocket.Write(expectedBytes)
	return seq_num, err
}

func (e *ESME) bindWithSmsc(pdu PDU) (*PDU, error) {
	_, err := e.Send(&pdu)
	if err != nil {
		return nil, err
	}
	resp, err := waitForBindResponse(e)
	return resp, err
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
	if pdu.Header.CommandStatus == ESME_ROK {
		switch pdu.Header.CommandId {
		case "bind_receiver_resp":
			Esme.state.setState <- BOUND_RX

		case "bind_transmitter_resp":
			Esme.state.setState <- BOUND_TX

		case "bind_transceiver_resp":
			Esme.state.setState <- BOUND_TRX
		}
	} else {
		err = fmt.Errorf("The answer received wasn't OK or not the type we expected : %v", pdu)
	}
	return err
}

func (e *ESME) receivePdu() (PDU, error) {
	readBuf, LastError := readPduBytesFromConnection(e.clientSocket, time.Now().Add(1*time.Second))
	if LastError != nil {
		if errors.Is(LastError, io.EOF) || errors.Is(LastError, net.ErrClosed) {
			go e.Close()
		}
		return PDU{}, fmt.Errorf("Couldn't read on a Connection: \n err =%w", LastError)
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
	currentState := e.GetEsmeState()
	return (currentState == BOUND_TX || currentState == BOUND_TRX)
}

func (e *ESME) isReceiverState() bool {
	currentState := e.GetEsmeState()
	return (currentState == BOUND_RX || currentState == BOUND_TRX)
}

func registerStandardBehaviours(e *ESME) {
	e.CommandFunctions["enquire_link"] = handleEnquiryLinkPduReceived
	e.CommandFunctions["submit_sm"] = handleSubmitSmPduReceived
	e.CommandFunctions["deliver_sm"] = handleDeliverSmPduReceived
}

func (e *ESME) StartControlLoop() {
	e.wg.Add(1)
	go e.pduDispatcher()
}

func (e *ESME) pduDispatcher() {
	for e.GetEsmeState() != CLOSED {
		pdu, err := e.receivePdu()
		if err != nil || pdu.Header == (Header{}) {
			continue
		}
		e.CommandFunctions[pdu.Header.CommandId](e, pdu)
	}
	e.wg.Done()
}
