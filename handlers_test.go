package smpp

import (
	"context"
	"encoding/hex"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// blockingHandler counts how many calls run at once and keeps each one until release is closed.
type blockingHandler struct {
	running, maxRunning, calls atomic.Int32
	release                    chan struct{}
}

func (h *blockingHandler) handle(e *ESME, p PDU) error {
	n := h.running.Add(1)
	for {
		m := h.maxRunning.Load()
		if n <= m || h.maxRunning.CompareAndSwap(m, n) {
			break
		}
	}
	<-h.release
	h.running.Add(-1)
	h.calls.Add(1)
	return e.Reply(NewEnquireLinkResp().WithSequenceNumber(p.Header.SequenceNumber))
}

// sendEnquireLinksToEsme connects an ESME whose enquire_link handler is h, and has the SMSC
// side send it n enquire_links.
func sendEnquireLinksToEsme(t *testing.T, windowSize, n int, h *blockingHandler) (*SMSC, *ESME, []<-chan Response) {
	smsc, _, esme := connectEsmeAndSmscTogether(t)
	esme.WindowSize = windowSize
	esme.CommandFunctions["enquire_link"] = h.handle
	esme.StartControlLoop()
	smscSide := smsc.ESMEs.Load().([]*ESME)[0]
	var responses []<-chan Response
	for i := 0; i < n; i++ {
		r, err := smscSide.SendAsync(NewEnquireLink())
		if err != nil {
			t.Fatalf("SendAsync() error = %v", err)
		}
		responses = append(responses, r)
	}
	return smsc, esme, responses
}

func eventually(condition func() bool) bool {
	for deadline := time.Now().Add(time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		if condition() {
			return true
		}
	}
	return condition()
}

func TestHandlersRunConcurrentlyWithoutWindow(t *testing.T) {
	h := &blockingHandler{release: make(chan struct{})}
	smsc, esme, responses := sendEnquireLinksToEsme(t, 0, 3, h)
	defer CloseAndAssertClean(smsc, esme, t)
	release := sync.OnceFunc(func() { close(h.release) })
	defer release() // runs before the cleanup, which waits for running handlers

	if !eventually(func() bool { return h.running.Load() == 3 }) {
		t.Errorf("%d handlers running at once, want 3", h.running.Load())
	}
	release()
	for _, r := range responses {
		if got := receiveWithin(t, r, time.Second); got.Err != nil {
			t.Errorf("response error = %v", got.Err)
		}
	}
}

func TestWindowSizeLimitsConcurrentHandlers(t *testing.T) {
	h := &blockingHandler{release: make(chan struct{})}
	smsc, esme, responses := sendEnquireLinksToEsme(t, 2, 3, h)
	defer CloseAndAssertClean(smsc, esme, t)
	release := sync.OnceFunc(func() { close(h.release) })
	defer release() // runs before the cleanup, which waits for running handlers

	if !eventually(func() bool { return h.running.Load() == 2 }) {
		t.Fatalf("%d handlers running at once, want 2 (the window)", h.running.Load())
	}
	time.Sleep(50 * time.Millisecond) // give a third handler time to start, if the window leaked
	if got := h.running.Load(); got != 2 {
		t.Errorf("%d handlers running at once, want 2 (the window)", got)
	}
	release()
	for _, r := range responses {
		if got := receiveWithin(t, r, time.Second); got.Err != nil {
			t.Errorf("response error = %v", got.Err)
		}
	}
	if got := h.maxRunning.Load(); got != 2 {
		t.Errorf("at most %d handlers ran at once, want 2", got)
	}
}

// A handler runs outside the read loop, so it can itself send a request and wait for the answer.
func TestHandlerCanSendAndWaitForAResponse(t *testing.T) {
	smsc, _, esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, esme, t)
	esme.CommandFunctions["deliver_sm"] = func(e *ESME, p PDU) error {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if _, err := e.Send(ctx, NewEnquireLink()); err != nil {
			return e.Reply(NewDeliverSMResp().WithSequenceNumber(p.Header.SequenceNumber).WithMessageId("").WithSMPPError(ESME_RSYSERR))
		}
		return e.Reply(NewDeliverSMResp().WithSequenceNumber(p.Header.SequenceNumber).WithMessageId("ok"))
	}
	esme.StartControlLoop()

	smscSide := smsc.ESMEs.Load().([]*ESME)[0]
	resp, err := smscSide.Send(contextWithin(t, 3*time.Second), NewDeliverSM().WithMessage("Hello"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if resp.Body.MandatoryParameter["message_id"] != "ok" {
		t.Errorf("deliver_sm_resp = %v, want message_id ok (the handler's own Send must have worked)", resp)
	}
}

// The SMSC's control loop must end with the SMSC, not keep running (or spinning) after Close.
func TestSmscControlLoopStopsOnClose(t *testing.T) {
	smsc, err := GetSmscSimulatorServer()
	if err != nil {
		t.Fatalf("GetSmscSimulatorServer() error = %v", err)
	}
	if !eventually(func() bool { return controlLoopsRunning() > 0 }) {
		t.Fatal("no SMSC control loop running")
	}
	smsc.Close()
	if !eventually(func() bool { return controlLoopsRunning() == 0 }) {
		t.Errorf("%d SMSC control loops still running after Close", controlLoopsRunning())
	}
}

func controlLoopsRunning() int {
	return goroutinesIn("(*SMSC).smscControlLoop(")
}

// goroutinesIn counts the goroutines whose stack contains frame. The buffer grows until the
// whole dump fits: a truncated dump would miss goroutines.
func goroutinesIn(frame string) int {
	buf := make([]byte, 1<<20)
	for {
		n := runtime.Stack(buf, true)
		if n < len(buf) {
			return strings.Count(string(buf[:n]), frame)
		}
		buf = make([]byte, 2*len(buf))
	}
}

// A connection's cleanup can finish after the SMSC has closed: it must not then wait forever
// to hand its ESME to the SMSC's control loop, which is gone.
func TestConnectionCleanupDoesntWaitForAClosedSmsc(t *testing.T) {
	for i := 0; i < 20; i++ { // the leak needs the cleanup to lose a race with the close
		smsc, _, esme := connectEsmeAndSmscTogether(t)
		esme.Close() // the SMSC side notices and starts its cleanup...
		smsc.Close() // ...while the SMSC closes
	}
	if !eventually(func() bool { return goroutinesIn("(*SMSC).closeAndRemoveEsme(") == 0 }) {
		t.Errorf("%d connection cleanups stuck waiting for a closed SMSC", goroutinesIn("(*SMSC).closeAndRemoveEsme("))
	}
}

// A PDU whose command_id the ESME doesn't know gets a generic_nack (ESME_RINVCMDID) carrying
// its sequence_number, and the connection stays up.
func TestUnknownCommandIdIsAnsweredWithGenericNack(t *testing.T) {
	smsc, esme, smscConnection := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, esme, t)
	esme.StartControlLoop()

	unknown, _ := hex.DecodeString("00000010" + "00001115" + "00000000" + "0000002a") // sequence_number 42
	if _, err := smscConnection.Write(unknown); err != nil {
		t.Fatalf("writing the PDU: %v", err)
	}
	raw, err := readPduBytesFromConnection(smscConnection, time.Now().Add(time.Second))
	if err != nil {
		t.Fatalf("no answer: %v", err)
	}
	answer, err := ParsePdu(raw)
	if err != nil {
		t.Fatalf("ParsePdu() error = %v", err)
	}
	want := Header{CommandLength: 16, CommandId: "generic_nack", CommandStatus: ESME_RINVCMDID, SequenceNumber: 42}
	if answer.Header != want {
		t.Errorf("answer header = %+v, want %+v", answer.Header, want)
	}
	if state := esme.GetEsmeState(); state == CLOSED {
		t.Errorf("ESME state = %s, want the connection kept", state)
	}
}
