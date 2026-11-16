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

func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, Esme, smsc_connection := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	firstPdu := NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1)
	secondPdu := NewSubmitSM().WithSequenceNumber(2)
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
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme.Close()
	Esme.Close()
}

func TestSendingOnClosedEsmeReturnError(t *testing.T) {
	smsc, Esme, _ := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	Esme.Close()

	_, LastError1 := Esme.BindTransmitter(contextWithin(t, time.Second), validSystemID, validPassword)
	if !errors.Is(LastError1, net.ErrClosed) {
		t.Errorf("Error in test : %v", LastError1)
	}
}

func TestAssigningEsmeSomeDefaultsWithBinding(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
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

func TestSendingPduIncreaseSequenceNumberAcrossGoroutines(t *testing.T) {
	smsc, _, esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, esme, t)

	all_seq_numbers := make(chan int)
	seq_numbers_expected := []int{}
	seq_numbers_actual := []int{}
	iterations := 100
	for i := 0; i <= iterations; i++ {
		go func() {
			resp, err := esme.Send(contextWithin(t, 5*time.Second), NewEnquireLink())
			if err != nil {
				t.Errorf("Issue sending enquire_link: %v", err)
			}
			all_seq_numbers <- resp.Header.SequenceNumber
		}()
	}
	for i := 0; i <= iterations; i++ {
		seq_numbers_expected = append(seq_numbers_expected, i+1)
		seq_numbers_actual = append(seq_numbers_actual, <-all_seq_numbers)
	}

	assert.ElementsMatch(t, seq_numbers_actual, seq_numbers_expected)
}

func TestCanRegisterCustomFunctionWithinEsme(t *testing.T) {
	smsc, esme, _ := GetSmscAndConnectEsme(t)
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

	smsc_esme := smsc.ESMEs.Load().([]*ESME)[0]
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

func GetSmscAndConnectEsme(t *testing.T) (*SMSC, *ESME, net.Conn) {
	smsc, err := GetSmscSimulatorServer() // this doesn't start the SMSC, hence why the manual accept
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr(), connType)
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	_, err = smsc.acceptNewConnectionFromSMSC()
	if err != nil {
		t.Errorf("couldn't accept client to server successfully: %v", err)
	}
	smsc_connection := smsc.ESMEs.Load().([]*ESME)[0].clientSocket
	return smsc, Esme, smsc_connection
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
