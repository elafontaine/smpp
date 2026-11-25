package smpp

import (
	"context"
	"errors"
	"net"
)

// SMSC is the server side of SMPP: it accepts connections and runs an ESME for each.
//
// One goroutine, run, owns the list of connections; everything else reaches it through events.
// An accept pump does the blocking Accept and owns nothing.
type SMSC struct {
	SystemId string
	Password string

	listener net.Listener

	// Events for run.
	accepted      chan acceptedConn
	closedEsmes   chan *ESME
	closeRequests chan struct{}
	listQueries   chan chan []*ESME
	done          chan struct{} // closed when run returns: the SMSC and all its connections are gone

	// Owned by run.
	esmes   []*ESME
	closing bool
}

// acceptedConn hands a new connection to run, which answers with its ESME (nil once closing).
type acceptedConn struct {
	conn         net.Conn
	startReading bool
	esme         chan *ESME
}

// NewSMSC serves SMPP on listener: any net.Listener, e.g. from net.Listen ("tcp", "unix") or
// tls.Listen.
func NewSMSC(listener net.Listener, systemId string, password string) *SMSC {
	s := &SMSC{
		SystemId:      systemId,
		Password:      password,
		listener:      listener,
		accepted:      make(chan acceptedConn),
		closedEsmes:   make(chan *ESME),
		closeRequests: make(chan struct{}),
		listQueries:   make(chan chan []*ESME),
		done:          make(chan struct{}),
	}
	go s.run()
	return s
}

// Start accepts connections until Close.
func (s *SMSC) Start() {
	go s.pumpAccepts()
}

// Close stops the SMSC: it accepts no more connections and every connection is told to close.
// It returns right away, without waiting for them; Shutdown waits.
func (s *SMSC) Close() {
	select {
	case s.closeRequests <- struct{}{}:
	case <-s.done:
	}
}

// Shutdown closes the SMSC (see Close) and waits until it and every connection are gone, or
// until ctx ends.
func (s *SMSC) Shutdown(ctx context.Context) error {
	s.Close()
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Connections returns the SMSC's current connections, as a copy.
func (s *SMSC) Connections() []*ESME {
	reply := make(chan []*ESME, 1)
	select {
	case s.listQueries <- reply:
		return <-reply
	case <-s.done:
		return nil
	}
}

func (s *SMSC) GetNumberOfConnection() int {
	return len(s.Connections())
}

// run owns the list of connections and handles its events one at a time, never blocking. Once
// closing, it keeps going until every connection has reported its end.
func (s *SMSC) run() {
	defer close(s.done)
	for !s.closing || len(s.esmes) > 0 {
		select {
		case a := <-s.accepted:
			s.onAccept(a)
		case e := <-s.closedEsmes:
			s.onEsmeClosed(e)
		case <-s.closeRequests:
			s.onClose()
		case reply := <-s.listQueries:
			reply <- append([]*ESME(nil), s.esmes...)
		}
	}
}

func (s *SMSC) onAccept(a acceptedConn) {
	if s.closing {
		a.conn.Close()
		a.esme <- nil
		return
	}
	e := newEsme(a.conn, s.reportClosed)
	e.CommandFunctions["bind_receiver"] = s.handleBindOperation
	e.CommandFunctions["bind_transceiver"] = s.handleBindOperation
	e.CommandFunctions["bind_transmitter"] = s.handleBindOperation
	e.CommandFunctions["submit_sm"] = handleSubmitSmPduReceived   // simulator answer
	e.CommandFunctions["deliver_sm"] = handleDeliverSmPduReceived // refused: an SMSC doesn't take deliver_sm
	s.esmes = append(s.esmes, e)
	if a.startReading {
		e.StartControlLoop()
	}
	a.esme <- e
}

func (s *SMSC) onEsmeClosed(e *ESME) {
	for i, x := range s.esmes {
		if x == e {
			s.esmes = append(s.esmes[:i], s.esmes[i+1:]...)
			return
		}
	}
}

// onClose stops accepting and asks every connection to close; each reports back when it's done.
func (s *SMSC) onClose() {
	if s.closing {
		return
	}
	s.closing = true
	s.listener.Close()
	for _, e := range s.esmes {
		e.requestClose()
	}
}

// reportClosed is each connection's last act: it tells run the connection is gone.
func (s *SMSC) reportClosed(e *ESME) {
	select {
	case s.closedEsmes <- e:
	case <-s.done:
	}
}

// handOver gives an accepted connection to run. Once the SMSC is closing or gone, the connection
// is closed instead.
func (s *SMSC) handOver(conn net.Conn, startReading bool) (*ESME, error) {
	a := acceptedConn{conn: conn, startReading: startReading, esme: make(chan *ESME, 1)}
	select {
	case s.accepted <- a:
		if e := <-a.esme; e != nil {
			return e, nil
		}
	case <-s.done:
		conn.Close()
	}
	return nil, net.ErrClosed
}

// acceptNewConnectionFromSMSC accepts one connection without reading it (tests drive it).
func (s *SMSC) acceptNewConnectionFromSMSC() (*ESME, error) {
	conn, err := s.listener.Accept()
	if err != nil {
		return nil, err
	}
	return s.handOver(conn, false)
}

// pumpAccepts accepts connections for run until the listener is closed.
func (s *SMSC) pumpAccepts() {
	for {
		conn, err := s.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			InfoSmppLogger.Printf("SMSC wasn't able to accept a new connection: %v", err)
			continue
		}
		if _, err := s.handOver(conn, true); err != nil {
			return
		}
	}
}
