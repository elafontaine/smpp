package smpp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
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

// maxPduLength bounds command_length, so a corrupt length can't make us allocate gigabytes.
// 128 KiB leaves room for a full 64 KiB message_payload.
const maxPduLength = 128 * 1024

// Response is what SendAsync delivers: the response PDU, or why none will come.
type Response struct {
	PDU PDU
	Err error
}

// ESME is the client side of the SMPP protocol.  Users should be
// managing their ESMEs to connect to an SMPP server (SMSC).
type ESME struct {
	clientSocket     net.Conn
	state            *State
	sequenceNumber   int32
	CommandFunctions map[string]func(*ESME, PDU) error
	WindowSize       int // most requests handled at once; 0: no limit. Set it before the loop starts.
	defaults         map[string]interface{}
	wg               sync.WaitGroup
	startOnce        sync.Once
	writeMu          sync.Mutex // one PDU on the wire at a time
	pendingMu        sync.Mutex
	pending          map[int]chan Response // requests waiting for their response, by sequence_number; nil once closed
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
		clientSocket:     clientSocket,
		state:            NewESMEState(OPEN),
		CommandFunctions: map[string]func(*ESME, PDU) error{},
		defaults:         map[string]interface{}{},
		pending:          map[int]chan Response{},
	}
	registerStandardBehaviours(e)
	return e
}

// Close closes the connection, waits for the read loop and running handlers to stop, and fails
// every request still waiting for its response with net.ErrClosed. Calling it again does
// nothing more.
func (e *ESME) Close() {
	// Claim the loop's Once: a loop that's starting finishes its wg.Add before Wait below (Add
	// racing Wait is a WaitGroup misuse), and a loop not started yet never starts.
	e.startOnce.Do(func() {})
	e.state.SetState(CLOSED)
	e.clientSocket.Close()
	e.wg.Wait()
	e.failPending() // requests made while no read loop was running
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

func (e *ESME) BindTransmitter(ctx context.Context, systemID, password string) (PDU, error) {
	return e.bindWithSmsc(ctx, NewBindTransmitter().WithSystemId(systemID).WithPassword(password))
}

func (e *ESME) BindAsTransmitter(ctx context.Context) error {
	_, err := e.bindWithSmsc(ctx, NewBindTransmitter().WithDefaults(e.defaults))
	return err
}

func (e *ESME) BindTransceiver(ctx context.Context, systemID, password string) (PDU, error) {
	return e.bindWithSmsc(ctx, NewBindTransceiver().WithSystemId(systemID).WithPassword(password))
}

func (e *ESME) BindReceiver(ctx context.Context, systemID, password string) (PDU, error) {
	return e.bindWithSmsc(ctx, NewBindReceiver().WithSystemId(systemID).WithPassword(password))
}

func (e *ESME) bindWithSmsc(ctx context.Context, pdu PDU) (PDU, error) {
	resp, err := e.Send(ctx, pdu)
	if err != nil {
		return resp, err
	}
	return resp, setESMEStateFromSMSCResponse(&resp, e)
}

// Send sends a request and waits for its response, until ctx ends. A sequence_number of 0 is
// replaced by the next one of the session.
func (e *ESME) Send(ctx context.Context, pdu PDU) (PDU, error) {
	seq, responses, err := e.sendAsync(pdu)
	if err != nil {
		return PDU{}, err
	}
	select {
	case r := <-responses:
		return r.PDU, r.Err
	case <-ctx.Done():
		e.forget(seq) // a late response is then dropped
		return PDU{}, ctx.Err()
	}
}

// SendAsync sends a request and returns without waiting. Encoding and writing errors come back
// right away; the channel then delivers the response, or net.ErrClosed if the ESME closes first.
func (e *ESME) SendAsync(pdu PDU) (<-chan Response, error) {
	_, responses, err := e.sendAsync(pdu)
	return responses, err
}

func (e *ESME) sendAsync(pdu PDU) (int, <-chan Response, error) {
	seq := pdu.Header.SequenceNumber
	if seq == 0 {
		seq = int(atomic.AddInt32(&e.sequenceNumber, 1))
	}
	encoded, err := EncodePdu(pdu.WithSequenceNumber(seq))
	if err != nil {
		return seq, nil, err
	}

	responses := make(chan Response, 1) // the read loop never waits on a slow caller
	e.pendingMu.Lock()
	switch {
	case e.pending == nil:
		e.pendingMu.Unlock()
		return seq, nil, net.ErrClosed
	case e.pending[seq] != nil:
		e.pendingMu.Unlock()
		return seq, nil, fmt.Errorf("sequence_number %d is already waiting for a response", seq)
	}
	e.pending[seq] = responses // before writing: the response can arrive before Write returns
	e.pendingMu.Unlock()

	e.StartControlLoop()
	if err := e.write(encoded); err != nil {
		e.forget(seq)
		return seq, nil, err
	}
	return seq, responses, nil
}

// Reply sends a response PDU as is, keeping its sequence_number, without waiting for anything.
func (e *ESME) Reply(pdu PDU) error {
	encoded, err := EncodePdu(pdu)
	if err != nil {
		return err
	}
	return e.write(encoded)
}

func (e *ESME) write(encoded []byte) error {
	e.writeMu.Lock()
	defer e.writeMu.Unlock()
	_, err := e.clientSocket.Write(encoded)
	return err
}

func (e *ESME) forget(seq int) {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	delete(e.pending, seq)
}

func (e *ESME) failPending() {
	e.pendingMu.Lock()
	defer e.pendingMu.Unlock()
	for _, responses := range e.pending {
		responses <- Response{Err: net.ErrClosed}
	}
	e.pending = nil
}

// deliverResponse hands a response (*_resp or generic_nack) to the request waiting for it, and
// tells whether there was one.
func (e *ESME) deliverResponse(pdu PDU) bool {
	if !strings.HasSuffix(pdu.Header.CommandId, "_resp") && pdu.Header.CommandId != "generic_nack" {
		return false
	}
	e.pendingMu.Lock()
	responses, ok := e.pending[pdu.Header.SequenceNumber]
	delete(e.pending, pdu.Header.SequenceNumber)
	e.pendingMu.Unlock()
	if ok {
		responses <- Response{PDU: pdu}
	}
	return ok
}

func setESMEStateFromSMSCResponse(pdu *PDU, Esme *ESME) (err error) {
	if pdu.Header.CommandStatus == ESME_ROK {
		switch pdu.Header.CommandId {
		case "bind_receiver_resp":
			Esme.state.SetState(BOUND_RX)

		case "bind_transmitter_resp":
			Esme.state.SetState(BOUND_TX)

		case "bind_transceiver_resp":
			Esme.state.SetState(BOUND_TRX)
		}
	} else {
		err = fmt.Errorf("The answer received wasn't OK or not the type we expected : %v", pdu)
	}
	return err
}

// readPduBytesFromConnection reads one whole PDU. A zero deadline waits as long as needed.
func readPduBytesFromConnection(ConnectionSocket net.Conn, deadline time.Time) ([]byte, error) {
	if err := ConnectionSocket.SetReadDeadline(deadline); err != nil {
		return nil, err
	}
	lengthBytes := make([]byte, 4)
	if _, err := io.ReadFull(ConnectionSocket, lengthBytes); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint32(lengthBytes))
	if length < 16 || length > maxPduLength {
		return nil, fmt.Errorf("command_length %d is outside 16..%d: the PDU boundaries are lost", length, maxPduLength)
	}
	pdu := make([]byte, length)
	copy(pdu, lengthBytes)
	if _, err := io.ReadFull(ConnectionSocket, pdu[4:]); err != nil {
		return nil, err
	}
	return pdu, nil
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

// StartControlLoop starts the loop that reads every PDU from the connection: responses go to
// the requests waiting for them, requests to CommandFunctions. Register CommandFunctions
// before starting it. Send and SendAsync start it when needed; starting it again does nothing.
func (e *ESME) StartControlLoop() {
	e.startOnce.Do(func() {
		e.wg.Add(1)
		go e.pduDispatcher()
	})
}

// pduDispatcher is the only reader of the connection. Each request runs its handler in its own
// goroutine; once WindowSize handlers are running, it stops reading until one of them finishes.
func (e *ESME) pduDispatcher() {
	defer e.wg.Done()
	var window chan struct{}
	if e.WindowSize > 0 {
		window = make(chan struct{}, e.WindowSize)
	}
	for {
		raw, err := readPduBytesFromConnection(e.clientSocket, time.Time{})
		if err != nil { // connection gone, or PDU boundaries lost: nothing more can be read reliably
			e.state.SetState(CLOSED)
			e.clientSocket.Close()
			e.failPending()
			return
		}
		pdu, err := ParsePdu(raw)
		if errors.Is(err, errUnknownCommandID) {
			sequenceNumber := int(binary.BigEndian.Uint32(raw[12:16])) // the header is there: raw is a whole PDU
			if err := e.Reply(NewGenericNack().WithSMPPError(ESME_RINVCMDID).WithSequenceNumber(sequenceNumber)); err != nil {
				ErrorSmppLogger.Printf("Couldn't answer an unknown command_id with generic_nack: %v", err)
			}
			continue
		}
		if err != nil {
			continue // malformed PDU of a known command, boundaries intact: skipped
		}
		if e.deliverResponse(pdu) {
			continue
		}
		if handler, ok := e.CommandFunctions[pdu.Header.CommandId]; ok {
			if window != nil {
				window <- struct{}{} // waits while the window is full
			}
			e.wg.Add(1) // Close waits for running handlers too
			go func() {
				defer e.wg.Done()
				if window != nil {
					defer func() { <-window }()
				}
				if err := handler(e, pdu); err != nil {
					ErrorSmppLogger.Printf("Error handling %s PDU: %v", pdu.Header.CommandId, err)
				}
			}()
		}
	}
}
