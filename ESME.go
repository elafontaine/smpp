package smpp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// The library's loggers. By default Debug and Info are discarded and Warning and Error go to
// stderr; replace any of them to change that.
var (
	DebugSmppLogger   = log.New(io.Discard, "DEBUG: ", log.LstdFlags)
	WarningSmppLogger = log.New(os.Stderr, "WARNING: ", log.LstdFlags)
	InfoSmppLogger    = log.New(io.Discard, "INFO: ", log.LstdFlags)
	ErrorSmppLogger   = log.New(os.Stderr, "ERROR: ", log.LstdFlags)
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
	writes           chan writeRequest                        // to runWriter, the only writer of the connection
	pendingOps       chan func(pending map[int]chan Response) // to runPending, the owner of the waiting requests
}

// writeRequest asks runWriter to write one encoded PDU, and gets the Write error back.
type writeRequest struct {
	encoded []byte
	written chan error
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
		writes:           make(chan writeRequest),
		pendingOps:       make(chan func(map[int]chan Response)),
	}
	registerStandardBehaviours(e)
	e.wg.Add(2) // both stop once the ESME is CLOSED
	go e.runWriter()
	go e.runPending()
	return e
}

// Close closes the connection and waits for the ESME's goroutines to stop; every request still
// waiting for its response then gets net.ErrClosed. Calling it again does nothing more.
func (e *ESME) Close() {
	// Claim the loop's Once: a loop that's starting finishes its wg.Add before Wait below (Add
	// racing Wait is a WaitGroup misuse), and a loop not started yet never starts.
	e.startOnce.Do(func() {})
	e.state.SetState(CLOSED)
	e.clientSocket.Close()
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
	duplicate := false
	if !e.withPending(func(pending map[int]chan Response) {
		if duplicate = pending[seq] != nil; !duplicate {
			pending[seq] = responses // before writing: the response can arrive before Write returns
		}
	}) {
		return seq, nil, net.ErrClosed
	}
	if duplicate {
		return seq, nil, fmt.Errorf("sequence_number %d is already waiting for a response", seq)
	}

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

// write hands an encoded PDU to runWriter and returns the Write error.
func (e *ESME) write(encoded []byte) error {
	w := writeRequest{encoded: encoded, written: make(chan error, 1)}
	select {
	case e.writes <- w:
		return <-w.written
	case <-e.state.Done():
		return net.ErrClosed
	}
}

// runWriter is the only writer of the connection: PDUs go out whole, one at a time.
func (e *ESME) runWriter() {
	defer e.wg.Done()
	for {
		select {
		case w := <-e.writes:
			_, err := e.clientSocket.Write(w.encoded)
			w.written <- err
		case <-e.state.Done():
			return
		}
	}
}

// withPending runs f on the requests waiting for their response, inside runPending. It returns
// false, without running f, once the ESME is closed.
func (e *ESME) withPending(f func(pending map[int]chan Response)) bool {
	ran := make(chan struct{})
	select {
	case e.pendingOps <- func(pending map[int]chan Response) { f(pending); close(ran) }:
		<-ran
		return true
	case <-e.state.Done():
		return false
	}
}

// runPending owns the requests waiting for their response, by sequence_number: every access is
// a function it runs, one at a time. Once the ESME is closed it fails those left with
// net.ErrClosed and stops.
func (e *ESME) runPending() {
	defer e.wg.Done()
	pending := map[int]chan Response{}
	for {
		select {
		case op := <-e.pendingOps:
			op(pending)
		case <-e.state.Done():
			for _, responses := range pending {
				responses <- Response{Err: net.ErrClosed}
			}
			return
		}
	}
}

func (e *ESME) forget(seq int) {
	e.withPending(func(pending map[int]chan Response) { delete(pending, seq) })
}

// deliverResponse hands a response (*_resp or generic_nack) to the request waiting for it, and
// tells whether there was one.
func (e *ESME) deliverResponse(pdu PDU) bool {
	if !isResponse(pdu.Header.CommandId) {
		return false
	}
	var responses chan Response
	e.withPending(func(pending map[int]chan Response) {
		responses = pending[pdu.Header.SequenceNumber]
		delete(pending, pdu.Header.SequenceNumber)
	})
	if responses == nil {
		return false
	}
	responses <- Response{PDU: pdu}
	return true
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
			e.state.SetState(CLOSED) // runPending then fails the requests still waiting
			e.clientSocket.Close()
			return
		}
		pdu, err := ParsePdu(raw)
		if errors.Is(err, errUnknownCommandID) {
			e.nack(ESME_RINVCMDID, int(binary.BigEndian.Uint32(raw[12:16]))) // raw is a whole PDU: the header is there
			continue
		}
		if err != nil {
			if isRequest(pdu.Header.CommandId) { // known command, malformed body
				e.nack(ESME_RINVCMDLEN, pdu.Header.SequenceNumber)
			}
			continue
		}
		if e.deliverResponse(pdu) {
			continue
		}
		handler, ok := e.CommandFunctions[pdu.Header.CommandId]
		if !ok {
			if isRequest(pdu.Header.CommandId) { // a request nothing here handles
				e.nack(ESME_RINVCMDID, pdu.Header.SequenceNumber)
			}
			continue
		}
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

func isResponse(commandId string) bool {
	return strings.HasSuffix(commandId, "_resp") || commandId == "generic_nack"
}

// isRequest is false for responses, and for a header that couldn't be read (empty command_id).
func isRequest(commandId string) bool {
	return commandId != "" && !isResponse(commandId)
}

// nack answers a request this ESME can't process with a generic_nack.
func (e *ESME) nack(status string, sequenceNumber int) {
	if err := e.Reply(NewGenericNack().WithSMPPError(status).WithSequenceNumber(sequenceNumber)); err != nil {
		ErrorSmppLogger.Printf("Couldn't send generic_nack (%s, sequence_number %d): %v", status, sequenceNumber, err)
	}
}
