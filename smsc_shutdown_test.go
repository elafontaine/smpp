package smpp

import (
	"context"
	"sync"
	"testing"
	"time"
)

// smscWithABusyConnection gives an SMSC with n connections, accepted by hand. The first
// connection's SMSC side is in the middle of handling an enquire_link: its handler doesn't
// return until release is called.
func smscWithABusyConnection(t *testing.T, n int) (smsc *SMSC, smscSides []*ESME, release func()) {
	smsc, err := GetSmscSimulatorServer()
	if err != nil {
		t.Fatalf("GetSmscSimulatorServer() error = %v", err)
	}
	handling, released := make(chan struct{}), make(chan struct{})
	release = sync.OnceFunc(func() { close(released) })
	t.Cleanup(release) // a failing test must not leave the handler stuck
	var clients []*ESME
	for i := 0; i < n; i++ {
		client, err := InstantiateEsme(smsc.listeningSocket.Addr(), connType)
		if err != nil {
			t.Fatalf("InstantiateEsme() error = %v", err)
		}
		t.Cleanup(client.Close)
		clients = append(clients, client)
		smscSide, err := smsc.acceptNewConnectionFromSMSC()
		if err != nil {
			t.Fatalf("accepting: %v", err)
		}
		smscSides = append(smscSides, smscSide)
	}
	smscSides[0].CommandFunctions["enquire_link"] = func(e *ESME, p PDU) error {
		close(handling)
		<-released
		return e.Reply(NewEnquireLinkResp().WithSequenceNumber(p.Header.SequenceNumber))
	}
	for _, smscSide := range smscSides {
		smscSide.StartControlLoop()
	}
	if _, err := clients[0].SendAsync(NewEnquireLink()); err != nil {
		t.Fatalf("SendAsync() error = %v", err)
	}
	<-handling
	return smsc, smscSides, release
}

// Nothing the SMSC started may outlive it.
func assertNothingLeftRunning(t *testing.T) {
	t.Helper()
	for _, frame := range []string{"(*SMSC).watchConnection(", "(*SMSC).smscControlLoop(", "(*ESME).pduDispatcher("} {
		if !eventually(func() bool { return goroutinesIn(frame) == 0 }) {
			t.Errorf("%d goroutines still in %s", goroutinesIn(frame), frame)
		}
	}
}

// Shutdown is for when everything must be gone: it returns once every connection is closed,
// including one busy in a handler, and it closes them all at once, not one after the other.
func TestShutdownWaitsForEveryConnection(t *testing.T) {
	smsc, smscSides, release := smscWithABusyConnection(t, 2)

	shutdown := make(chan error, 1)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdown <- smsc.Shutdown(ctx)
	}()
	if !eventually(func() bool { return smscSides[1].GetEsmeState() == CLOSED }) {
		t.Error("the idle connection waited for the busy one to close")
	}
	select {
	case err := <-shutdown:
		t.Fatalf("Shutdown() returned (%v) while a handler was still running", err)
	case <-time.After(100 * time.Millisecond):
	}

	release()
	select {
	case err := <-shutdown:
		if err != nil {
			t.Fatalf("Shutdown() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Shutdown() didn't return once the handler finished")
	}
	for i, smscSide := range smscSides {
		if state := smscSide.GetEsmeState(); state != CLOSED {
			t.Errorf("connection %d state = %s, want CLOSED", i, state)
		}
	}
	AssertSmscIsClosedAndClean(smsc, t)
	assertNothingLeftRunning(t)
}

// Close is for when the SMSC's resources aren't wanted any more: it returns right away, even
// with a handler still running. Shutdown then waits for the rest, which must leave no leak.
func TestCloseReturnsWithoutWaiting(t *testing.T) {
	smsc, smscSides, release := smscWithABusyConnection(t, 1)

	closed := make(chan struct{})
	go func() {
		smsc.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Close() waited for the running handler")
	}

	release()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := smsc.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown() after Close() error = %v", err)
	}
	if state := smscSides[0].GetEsmeState(); state != CLOSED {
		t.Errorf("connection state = %s, want CLOSED", state)
	}
	AssertSmscIsClosedAndClean(smsc, t)
	assertNothingLeftRunning(t)
}
