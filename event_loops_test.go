package smpp

import (
	"errors"
	"net"
	"runtime"
	"strings"
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

// One goroutine owns each object: per connection, the ESME's run plus its two I/O pumps; for the
// SMSC, its run plus its accept pump. Nothing else (no watcher per connection).
func TestOneGoroutinePerConnectionPlusOneForTheSmsc(t *testing.T) {
	smsc, esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)
	esme.StartControlLoop()

	want := map[string]int{
		"(*ESME).run(":         2, // the client, and the SMSC's side of the connection
		"(*ESME).pumpReads(":   2,
		"(*ESME).pumpWrites(":  2,
		"(*SMSC).run(":         1,
		"(*SMSC).pumpAccepts(": 1,
	}
	total := 0
	for frame, count := range want {
		total += count
		if !eventually(func() bool { return goroutinesIn(frame) == count }) {
			t.Errorf("%d goroutines in %s, want %d", goroutinesIn(frame), frame, count)
		}
	}
	if !eventually(func() bool { return goroutinesInTheLibrary() == total }) {
		t.Errorf("%d goroutines run ESME or SMSC code, want %d", goroutinesInTheLibrary(), total)
	}
}

// goroutinesInTheLibrary counts the goroutines running ESME or SMSC code, whatever the function.
func goroutinesInTheLibrary() int {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	for n == len(buf) {
		buf = make([]byte, 2*len(buf))
		n = runtime.Stack(buf, true)
	}
	count := 0
	for _, goroutine := range strings.Split(string(buf[:n]), "\n\n") {
		if strings.Contains(goroutine, "smpp.(*ESME).") || strings.Contains(goroutine, "smpp.(*SMSC).") {
			count++
		}
	}
	return count
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
