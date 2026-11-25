package smpp

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"testing"
	"time"
)

func TestSendReturnsTheMatchingResponse(t *testing.T) {
	smsc, esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)

	resp, err := esme.Send(contextWithin(t, time.Second), NewEnquireLink())
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if resp.Header.CommandId != "enquire_link_resp" || resp.Header.SequenceNumber != 1 {
		t.Errorf("Send() = %v, want enquire_link_resp with sequence_number 1", resp.Header)
	}
}

// The default submit_sm handler must answer with the request's sequence_number, otherwise the
// response never reaches the request waiting for it.
func TestSubmitSmRespUsesTheRequestSequenceNumber(t *testing.T) {
	smsc, esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)
	if _, err := esme.BindTransmitter(contextWithin(t, time.Second), validSystemID, validPassword); err != nil {
		t.Fatalf("BindTransmitter() error = %v", err)
	}

	resp, err := esme.Send(contextWithin(t, time.Second), NewSubmitSM().WithMessage("Hello"))
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if resp.Header.CommandId != "submit_sm_resp" || resp.Header.SequenceNumber != 2 {
		t.Errorf("Send() = %v, want submit_sm_resp with sequence_number 2", resp.Header)
	}
}

// A PDU that can't be encoded is reported to the caller right away and never reaches the wire.
func TestSendAsyncReturnsEncodingErrorsRightAway(t *testing.T) {
	smsc, esme, smscConnection := connectWithSilentSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)

	if _, err := esme.SendAsync(NewSubmitSM().WithDataCoding(256)); err == nil {
		t.Fatal("SendAsync() error = nil, want the encoding error")
	}
	var timeout net.Error
	if _, err := readPduBytesFromConnection(smscConnection, time.Now().Add(100*time.Millisecond)); !errors.As(err, &timeout) || !timeout.Timeout() {
		t.Errorf("the SMSC received something (read error = %v), want nothing", err)
	}
}

func TestResponsesAreMatchedBySequenceNumber(t *testing.T) {
	smsc, esme, smscConnection := connectWithSilentSmsc(t) // the SMSC side doesn't answer: the test does
	defer CloseAndAssertClean(smsc, esme, t)

	first, err1 := esme.SendAsync(NewEnquireLink())
	second, err2 := esme.SendAsync(NewEnquireLink())
	if err1 != nil || err2 != nil {
		t.Fatalf("SendAsync() errors = %v, %v", err1, err2)
	}
	for _, seq := range []int{2, 1} { // answered in the opposite order
		encoded, _ := EncodePdu(NewEnquireLinkResp().WithSequenceNumber(seq))
		if _, err := smscConnection.Write(encoded); err != nil {
			t.Fatalf("writing the response: %v", err)
		}
	}

	if r := receiveWithin(t, second, time.Second); r.Err != nil || r.PDU.Header.SequenceNumber != 2 {
		t.Errorf("second request got %v (err %v), want sequence_number 2", r.PDU.Header, r.Err)
	}
	if r := receiveWithin(t, first, time.Second); r.Err != nil || r.PDU.Header.SequenceNumber != 1 {
		t.Errorf("first request got %v (err %v), want sequence_number 1", r.PDU.Header, r.Err)
	}
}

func TestSendGivesUpWhenTheContextEnds(t *testing.T) {
	smsc, esme, _ := connectWithSilentSmsc(t) // nobody answers
	defer CloseAndAssertClean(smsc, esme, t)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := esme.Send(ctx, NewEnquireLink()); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Send() error = %v, want context.DeadlineExceeded", err)
	}
	// Given up means no longer waiting: the same sequence_number can be used again.
	if _, err := esme.SendAsync(NewEnquireLink().WithSequenceNumber(1)); err != nil {
		t.Errorf("sequence_number 1 still waiting after Send gave up: %v", err)
	}
}

func TestClosingEsmeFailsPendingRequests(t *testing.T) {
	smsc, esme, _ := connectWithSilentSmsc(t) // nobody answers
	defer CloseAndAssertClean(smsc, esme, t)

	responses, err := esme.SendAsync(NewEnquireLink())
	if err != nil {
		t.Fatalf("SendAsync() error = %v", err)
	}
	esme.Close()
	if r := receiveWithin(t, responses, time.Second); !errors.Is(r.Err, net.ErrClosed) {
		t.Errorf("pending request got err = %v, want net.ErrClosed", r.Err)
	}
}

// TCP keeps no message boundaries: one PDU can arrive in several reads.
func TestReadPduWaitsForAPduSplitAcrossWrites(t *testing.T) {
	t.Parallel()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	go func() {
		_, _ = server.Write(enquiryLinkFixture[:6])
		time.Sleep(50 * time.Millisecond)
		_, _ = server.Write(enquiryLinkFixture[6:])
	}()

	got, err := readPduBytesFromConnection(client, time.Now().Add(time.Second))
	if err != nil || !bytes.Equal(got, enquiryLinkFixture) {
		t.Errorf("readPduBytesFromConnection() = %x, %v, want %x", got, err, enquiryLinkFixture)
	}
}

// A command_length outside what a PDU can be means the boundaries are lost; a huge one must
// not make us allocate it.
func TestReadPduRejectsImpossibleLengths(t *testing.T) {
	t.Parallel()
	for _, lengthHex := range []string{"00000008", "ffffffff"} {
		t.Run(lengthHex, func(t *testing.T) {
			client, server := net.Pipe()
			defer client.Close()
			defer server.Close()
			length, _ := hex.DecodeString(lengthHex)
			go func() { _, _ = server.Write(length) }()

			_, err := readPduBytesFromConnection(client, time.Now().Add(time.Second))
			if err == nil || !strings.Contains(err.Error(), "the PDU boundaries are lost") {
				t.Errorf("readPduBytesFromConnection() error = %v, want the boundaries-lost error", err)
			}
		})
	}
}

func contextWithin(t *testing.T, d time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return ctx
}

func receiveWithin(t *testing.T, responses <-chan Response, d time.Duration) Response {
	t.Helper()
	select {
	case r := <-responses:
		return r
	case <-time.After(d):
		t.Fatalf("no response within %v", d)
		return Response{}
	}
}

// Close can run while another goroutine is starting the read loop (the SMSC closing a
// connection that's just being accepted). Run with -race: a WaitGroup Add racing Wait fails.
func TestCloseWhileTheReadLoopStarts(t *testing.T) {
	client, server := net.Pipe()
	defer server.Close()
	esme := NewEsme(client)
	started := make(chan struct{})
	go func() {
		esme.StartControlLoop()
		close(started)
	}()
	esme.Close()
	<-started
}

