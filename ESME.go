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

// ESME is one end of an SMPP session (the client side, or the SMSC's side of a connection).
//
// One goroutine, run, owns the session: its state, the requests waiting for a response, the
// handlers running, closing. Everything else reaches it through events. Two pumps do the blocking
// I/O (one reads the connection, one writes it) and own nothing; each received request runs its
// handler in its own goroutine.
type ESME struct {
	CommandFunctions map[string]func(*ESME, PDU) error
	WindowSize       int // most requests handled at once; 0: no limit. Set it before reading starts.

	clientSocket   net.Conn
	defaults       map[string]interface{}
	sequenceNumber atomic.Int32
	onClosed       func(*ESME) // called by run as its last act (the SMSC learns its connection ended)

	// Events for run.
	sends         chan pendingRequest
	forgets       chan int
	stateQueries  chan chan string
	stateChanges  chan stateChange
	startReading  chan struct{}
	closeRequests chan struct{}
	received      chan []byte // from pumpReads
	readFailed    chan struct{}
	handlerDone   chan struct{}
	done          chan struct{} // closed when run returns

	writes chan writeRequest // to pumpWrites, the only writer of the connection

	// Owned by run.
	state   string
	pending map[int]chan Response // requests waiting for their response, by sequence_number
	running int                   // handlers running
	parked  *receivedRequest      // waiting for a free window slot; reading pauses meanwhile
	reading bool
	closing bool
}

// pendingRequest asks run to wait for the response to sequence_number seq.
type pendingRequest struct {
	seq        int
	responses  chan Response
	registered chan error
}

type stateChange struct {
	state   string
	applied chan struct{}
}

type receivedRequest struct {
	handler func(*ESME, PDU) error
	pdu     PDU
}

// writeRequest asks pumpWrites to write one encoded PDU, and gets the Write error back.
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

// InstantiateEsme connects to an SMSC with net.Dial (e.g. "tcp", "unix"). For TLS or any other
// connection, make it yourself and use NewEsme.
func InstantiateEsme(network, address string) (*ESME, error) {
	clientSocket, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewEsme(clientSocket), nil
}

// NewEsme runs an SMPP session on conn. Register CommandFunctions before reading starts
// (StartControlLoop, Send or SendAsync).
func NewEsme(conn net.Conn) *ESME {
	return newEsme(conn, nil)
}

func newEsme(conn net.Conn, onClosed func(*ESME)) *ESME {
	e := &ESME{
		CommandFunctions: map[string]func(*ESME, PDU) error{"enquire_link": handleEnquiryLinkPduReceived},
		clientSocket:     conn,
		defaults:         map[string]interface{}{},
		onClosed:         onClosed,
		sends:            make(chan pendingRequest),
		forgets:          make(chan int),
		stateQueries:     make(chan chan string),
		stateChanges:     make(chan stateChange),
		startReading:     make(chan struct{}),
		closeRequests:    make(chan struct{}),
		received:         make(chan []byte),
		readFailed:       make(chan struct{}),
		handlerDone:      make(chan struct{}),
		done:             make(chan struct{}),
		writes:           make(chan writeRequest),
		state:            OPEN,
		pending:          map[int]chan Response{},
	}
	go e.run()
	go e.pumpWrites()
	return e
}

// Close closes the connection and waits until the session is over: every request still waiting
// for its response gets net.ErrClosed, and running handlers have returned. Calling it again
// does nothing more.
func (e *ESME) Close() {
	e.requestClose()
	<-e.done
}

// requestClose asks the session to close, without waiting for it.
func (e *ESME) requestClose() {
	select {
	case e.closeRequests <- struct{}{}:
	case <-e.done:
	}
}

func (e *ESME) SetDefaults(defaults map[string]interface{}) {
	if defaults == nil {
		panic("Setting the ESME defaults with a nil value!")
	}
	e.defaults = defaults
}

func (e *ESME) GetEsmeState() string {
	reply := make(chan string, 1)
	select {
	case e.stateQueries <- reply:
		return <-reply
	case <-e.done:
		return CLOSED
	}
}

// setState changes the session's state; CLOSED stays CLOSED.
func (e *ESME) setState(state string) {
	change := stateChange{state: state, applied: make(chan struct{})}
	select {
	case e.stateChanges <- change:
		<-change.applied
	case <-e.done:
	}
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
	return resp, e.setStateFromBindResponse(resp)
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
		seq = int(e.sequenceNumber.Add(1))
	}
	encoded, err := EncodePdu(pdu.WithSequenceNumber(seq))
	if err != nil {
		return seq, nil, err
	}
	request := pendingRequest{seq: seq, responses: make(chan Response, 1), registered: make(chan error, 1)}
	select {
	case e.sends <- request: // registered before writing: the response can arrive before Write returns
	case <-e.done:
		return seq, nil, net.ErrClosed
	}
	if err := <-request.registered; err != nil {
		return seq, nil, err
	}
	if err := e.write(encoded); err != nil {
		e.forget(seq)
		return seq, nil, err
	}
	return seq, request.responses, nil
}

// Reply sends a response PDU as is, keeping its sequence_number, without waiting for anything.
func (e *ESME) Reply(pdu PDU) error {
	encoded, err := EncodePdu(pdu)
	if err != nil {
		return err
	}
	return e.write(encoded)
}

// write hands an encoded PDU to pumpWrites and returns the Write error.
func (e *ESME) write(encoded []byte) error {
	w := writeRequest{encoded: encoded, written: make(chan error, 1)}
	select {
	case e.writes <- w:
		return <-w.written
	case <-e.done:
		return net.ErrClosed
	}
}

func (e *ESME) forget(seq int) {
	select {
	case e.forgets <- seq:
	case <-e.done:
	}
}

// setStateFromBindResponse records a bind the SMSC accepted, and errors on one it refused.
func (e *ESME) setStateFromBindResponse(pdu PDU) error {
	if pdu.Header.CommandStatus != ESME_ROK {
		return fmt.Errorf("The answer received wasn't OK or not the type we expected : %v", &pdu)
	}
	switch pdu.Header.CommandId {
	case "bind_receiver_resp":
		e.setState(BOUND_RX)
	case "bind_transmitter_resp":
		e.setState(BOUND_TX)
	case "bind_transceiver_resp":
		e.setState(BOUND_TRX)
	}
	return nil
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

// StartControlLoop starts reading the connection: responses go to the requests waiting for
// them, requests to CommandFunctions. Send and SendAsync start it when needed; starting it again
// does nothing.
func (e *ESME) StartControlLoop() {
	select {
	case e.startReading <- struct{}{}:
	case <-e.done:
	}
}

// run owns the session and handles its events one at a time, never blocking on I/O. Once
// closing, it keeps going until the running handlers have returned.
func (e *ESME) run() {
	defer e.finish()
	for !e.closing || e.running > 0 {
		received := e.received
		if e.parked != nil || e.closing {
			received = nil // window full or closing: take no PDU, the read pump waits
		}
		select {
		case request := <-e.sends:
			e.onSend(request)
		case seq := <-e.forgets:
			delete(e.pending, seq)
		case reply := <-e.stateQueries:
			reply <- e.state
		case change := <-e.stateChanges:
			e.onStateChange(change)
		case <-e.startReading:
			e.onStartReading()
		case <-e.closeRequests:
			e.onClose()
		case raw := <-received:
			e.onPdu(raw)
		case <-e.readFailed: // connection gone, or PDU boundaries lost
			e.onClose()
		case <-e.handlerDone:
			e.onHandlerDone()
		}
	}
}

func (e *ESME) finish() {
	close(e.done)
	if e.onClosed != nil {
		e.onClosed(e)
	}
}

func (e *ESME) onSend(request pendingRequest) {
	switch {
	case e.closing:
		request.registered <- net.ErrClosed
	case e.pending[request.seq] != nil:
		request.registered <- fmt.Errorf("sequence_number %d is already waiting for a response", request.seq)
	default:
		e.pending[request.seq] = request.responses
		request.registered <- nil
		e.onStartReading()
	}
}

func (e *ESME) onStateChange(change stateChange) {
	if !e.closing {
		e.state = change.state
	}
	close(change.applied)
}

func (e *ESME) onStartReading() {
	if !e.reading && !e.closing {
		e.reading = true
		go e.pumpReads()
	}
}

// onClose closes the connection and fails the requests still waiting. run then waits for the
// running handlers.
func (e *ESME) onClose() {
	if e.closing {
		return
	}
	e.closing = true
	e.state = CLOSED
	e.clientSocket.Close()
	for _, responses := range e.pending {
		responses <- Response{Err: net.ErrClosed}
	}
	e.pending = nil
	e.parked = nil
}

// onPdu hands a response to the request waiting for it, or starts the handler of a request, or
// answers a request nothing here can process with a generic_nack.
func (e *ESME) onPdu(raw []byte) {
	pdu, err := ParsePdu(raw)
	if errors.Is(err, errUnknownCommandID) {
		go e.nack(ESME_RINVCMDID, int(binary.BigEndian.Uint32(raw[12:16]))) // raw is a whole PDU: the header is there
		return
	}
	if err != nil {
		if isRequest(pdu.Header.CommandId) { // known command, malformed body
			go e.nack(ESME_RINVCMDLEN, pdu.Header.SequenceNumber)
		}
		return
	}
	if responses, waiting := e.pending[pdu.Header.SequenceNumber]; waiting && isResponse(pdu.Header.CommandId) {
		delete(e.pending, pdu.Header.SequenceNumber)
		responses <- Response{PDU: pdu}
		return
	}
	handler, ok := e.CommandFunctions[pdu.Header.CommandId]
	if !ok {
		if isRequest(pdu.Header.CommandId) { // a request nothing here handles
			go e.nack(ESME_RINVCMDID, pdu.Header.SequenceNumber)
		}
		return
	}
	request := receivedRequest{handler: handler, pdu: pdu}
	if e.WindowSize > 0 && e.running >= e.WindowSize {
		e.parked = &request
		return
	}
	e.startHandler(request)
}

func (e *ESME) startHandler(request receivedRequest) {
	e.running++
	go func() {
		if err := request.handler(e, request.pdu); err != nil {
			ErrorSmppLogger.Printf("Error handling %s PDU: %v", request.pdu.Header.CommandId, err)
		}
		e.handlerDone <- struct{}{} // run is waiting for it: it doesn't stop while handlers run
	}()
}

func (e *ESME) onHandlerDone() {
	e.running--
	if e.parked != nil {
		request := *e.parked
		e.parked = nil
		e.startHandler(request)
	}
}

// pumpReads reads PDUs for run until the connection fails.
func (e *ESME) pumpReads() {
	for {
		raw, err := readPduBytesFromConnection(e.clientSocket, time.Time{})
		if err != nil {
			select {
			case e.readFailed <- struct{}{}:
			case <-e.done:
			}
			return
		}
		select {
		case e.received <- raw:
		case <-e.done:
			return
		}
	}
}

// pumpWrites is the only writer of the connection: PDUs go out whole, one at a time.
func (e *ESME) pumpWrites() {
	for {
		select {
		case w := <-e.writes:
			_, err := e.clientSocket.Write(w.encoded)
			w.written <- err
		case <-e.done:
			return
		}
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
