package smpp

import (
	"errors"
	"net"
	"testing"
	"time"
)

// The ESME's state lives in its own goroutine: binds change it, CLOSED is for good.
func TestEsmeStateIsClosedForGood(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	esme := NewEsme(client)
	if state := esme.GetEsmeState(); state != OPEN {
		t.Errorf("new ESME state = %s, want OPEN", state)
	}
	esme.setState(BOUND_TX)
	if state := esme.GetEsmeState(); state != BOUND_TX {
		t.Errorf("state after the bind = %s, want BOUND_TX", state)
	}

	esme.Close()
	esme.setState(OPEN) // too late
	if state := esme.GetEsmeState(); state != CLOSED {
		t.Errorf("state after Close = %s, want CLOSED", state)
	}
}

// While a closing ESME waits for a running handler its goroutine is still there; the state must
// stay CLOSED all the same.
func TestEsmeStateStaysClosedWhileClosing(t *testing.T) {
	h := &blockingHandler{release: make(chan struct{})}
	smsc, esme, _ := sendEnquireLinksToEsme(t, 0, 1, h)
	defer CloseAndAssertClean(smsc, esme, t)
	defer close(h.release)
	if !eventually(func() bool { return h.running.Load() == 1 }) {
		t.Fatal("the handler never started")
	}

	esme.requestClose() // the handler keeps the ESME's goroutine alive
	esme.setState(BOUND_TX)
	if state := esme.GetEsmeState(); state != CLOSED {
		t.Errorf("state set while closing = %s, want CLOSED", state)
	}
}

// Reading the state never waits behind a handler: handlers run outside the ESME's goroutine.
func TestEsmeStateAnswersWhileAHandlerRuns(t *testing.T) {
	h := &blockingHandler{release: make(chan struct{})}
	smsc, esme, _ := sendEnquireLinksToEsme(t, 0, 1, h)
	defer CloseAndAssertClean(smsc, esme, t)
	defer close(h.release) // runs before the cleanup, which waits for the handler
	if !eventually(func() bool { return h.running.Load() == 1 }) {
		t.Fatal("the handler never started")
	}

	answered := make(chan string, 1)
	go func() { answered <- esme.GetEsmeState() }()
	select {
	case state := <-answered:
		if state != OPEN {
			t.Errorf("state = %s, want OPEN", state)
		}
	case <-time.After(time.Second):
		t.Fatal("GetEsmeState() waited for the running handler")
	}
}

// A connection accepted just as the SMSC goes away must be closed, not left waiting for an SMSC
// goroutine that's gone.
func TestConnectionAcceptedAfterCloseIsClosed(t *testing.T) {
	smsc := newSmsc(t)
	smsc.Close()
	if !eventually(func() bool { return controlLoopsRunning() == 0 }) {
		t.Fatal("the SMSC never stopped")
	}
	client, smscSide := net.Pipe()
	defer client.Close()

	handedOver := make(chan error, 1)
	go func() {
		_, err := smsc.handOver(smscSide, true) // what the accept pump does with a new connection
		handedOver <- err
	}()
	select {
	case err := <-handedOver:
		if !errors.Is(err, net.ErrClosed) {
			t.Errorf("handOver() error = %v, want net.ErrClosed", err)
		}
	case <-time.After(time.Second):
		t.Fatal("the accepted connection waits forever for a closed SMSC")
	}
	_ = client.SetReadDeadline(time.Now().Add(time.Second))
	var timeout net.Error
	if _, err := client.Read(make([]byte, 1)); err == nil || (errors.As(err, &timeout) && timeout.Timeout()) {
		t.Errorf("the accepted connection is still open (read error = %v)", err)
	}
}
