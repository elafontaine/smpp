package smpp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// A sequence_number filled in before sending is sent as is, not replaced by the session's next one.
func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, Esme, smsc_connection := connectWithSilentSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	firstPdu := NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(42)
	secondPdu := NewSubmitSM().WithSequenceNumber(7)
	_, LastError1 := Esme.SendAsync(firstPdu)
	_, LastError2 := Esme.SendAsync(secondPdu)
	if LastError1 != nil || LastError2 != nil {
		t.Errorf("Error writing : %v, %v", LastError1, LastError2)
	}
	assertReceivedPduIsSameAsExpected(smsc_connection, t, firstPdu)
	assertReceivedPduIsSameAsExpected(smsc_connection, t, secondPdu)
}

// Closing an ESME a second time must neither block nor panic. (That Close closes the socket
// is checked by TestSendingOnClosedEsmeReturnError and TestReceivingOnClosingEsmeReturnError.)
func TestClosingEsmeTwiceDoesntBlock(t *testing.T) {
	smsc, Esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme.Close()
	Esme.Close()
}

func TestSendingOnClosedEsmeReturnError(t *testing.T) {
	smsc, Esme, _ := connectWithSilentSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	Esme.Close()

	_, LastError1 := Esme.BindTransmitter(contextWithin(t, time.Second), validSystemID, validPassword)
	if !errors.Is(LastError1, net.ErrClosed) {
		t.Errorf("Error in test : %v", LastError1)
	}
}

func TestAssigningEsmeSomeDefaultsWithBinding(t *testing.T) {
	smsc, Esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	esmeDefaults := map[string]interface{}{
		"system_id": validSystemID,
		"password":  validPassword,
	}
	Esme.SetDefaults(esmeDefaults)
	err := Esme.BindAsTransmitter(contextWithin(t, time.Second))

	if err != nil || Esme.GetEsmeState() != BOUND_TX {
		t.Errorf("Error in test : %v", err)
	}
}

// Requests sent from many goroutines at once each get their own sequence_number, 1 to n.
func TestSendingPduIncreaseSequenceNumberAcrossGoroutines(t *testing.T) {
	smsc, esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)

	const requests = 100
	sent := make(chan int)
	for i := 0; i < requests; i++ {
		go func() {
			seq, _, err := esme.sendAsync(NewEnquireLink())
			if err != nil {
				t.Errorf("Issue sending enquire_link: %v", err)
			}
			sent <- seq
		}()
	}
	expected, actual := []int{}, []int{}
	for i := 0; i < requests; i++ {
		expected = append(expected, i+1)
		actual = append(actual, <-sent)
	}

	assert.ElementsMatch(t, expected, actual)
}

func TestCanRegisterCustomFunctionWithinEsme(t *testing.T) {
	smsc, esme, _ := connectWithSilentSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)

	message_ids := messageIdsGenerator(t)

	esme.CommandFunctions["deliver_sm"] = func(e *ESME, p PDU) error {
		pdu_resp := NewDeliverSMResp().
			WithMessageId(<-message_ids).
			WithSequenceNumber(p.Header.SequenceNumber)
		if !e.isReceiverState() {
			pdu_resp = pdu_resp.WithSMPPError(ESME_RINVBNDSTS)
		}
		return e.Reply(pdu_resp)
	}

	esme.StartControlLoop()

	smsc_esme := smsc.Connections()[0]
	pdu, smsc_err := smsc_esme.Send(contextWithin(t, time.Second), NewDeliverSM().WithMessage("Hello"))
	if smsc_err != nil {
		t.Errorf("Something failed while waiting for client to answer to server: %v", smsc_err)
	}
	if pdu.Header == (Header{}) || pdu.Body.MandatoryParameter["message_id"] != "0" {
		t.Errorf("Didn't receive expected Pdu: %v", pdu)
	}
}

func messageIdsGenerator(t *testing.T) chan string {
	message_ids := make(chan string)
	stop := make(chan struct{})
	t.Cleanup(func() { close(stop) })
	go func() {
		for i := 0; ; i++ {
			select {
			case message_ids <- fmt.Sprint(i):
			case <-stop:
				return
			}
		}
	}()
	return message_ids
}

func assertReceivedPduIsSameAsExpected(smsc_connection net.Conn, t *testing.T, expectedPDU PDU) {
	ActualPdu, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("We didn't read the PDU we sent correctly : %v", LastError)
	}
	expectedBytes, LastError := EncodePdu(expectedPDU)
	if !bytes.Equal(ActualPdu, expectedBytes) || LastError != nil {
		t.Errorf("We didn't receive expected PDU (sequence Number wrong?) : %v", LastError)
	}
}
