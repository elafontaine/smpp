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
	sequence_number1, LastError1 := Esme.Send(&firstPdu)
	sequence_number2, LastError2 := Esme.Send(&secondPdu)
	if sequence_number1 != 1 || sequence_number2 != 2 {
		t.Errorf("Sending sequence number isn't as expected !")
	}
	if LastError1 != nil || LastError2 != nil {
		t.Errorf("Error writing : %v", LastError2)
	}
	assertReceivedPduIsSameAsExpected(smsc_connection, t, firstPdu)
	assertReceivedPduIsSameAsExpected(smsc_connection, t, secondPdu)
}

func TestClosingEsmeCloseSocketAndDoesntBlock(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme.Close()
	Esme.Close()
}

func TestSendingOnClosedEsmeReturnError(t *testing.T) {
	smsc, Esme, _ := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	Esme.Close()
	
	_, LastError1 := Esme.BindTransmitter(validSystemID,validPassword)
	if !errors.Is(LastError1,net.ErrClosed){
		t.Errorf("Error in test : %v", LastError1)
	}
}

func TestReceivingOnClosingEsmeReturnError(t *testing.T) {
	smsc, Esme, _ := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	
	firstPdu := NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1)
	Esme.Send(&firstPdu)
	Esme.Close()
	_, err := Esme.receivePdu()
	if !errors.Is(err,net.ErrClosed){ 
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
	for i:= 0; i<= iterations; i++ {
		go func(){
			enquireLink := NewEnquireLink()
			actual_seq_num, err := esme.Send(&enquireLink)
			if err != nil {
				t.Error("Issue sending enquire_link")
			}
			all_seq_numbers <- actual_seq_num
		}()
	}
	for i:= 0; i<= iterations; i++ {
		seq_numbers_expected = append(seq_numbers_expected, i+1)
		seq_numbers_actual = append(seq_numbers_actual, <-all_seq_numbers)
	}
		
	assert.ElementsMatch(t,seq_numbers_actual, seq_numbers_expected)
}

func TestCanRegisterCustomFunctionWithinEsme(t *testing.T) {
	smsc, esme, _ := GetSmscAndConnectEsme(t)
	defer CloseAndAssertClean(smsc, esme, t)

	message_ids := messageIdsGenerator()

	esme.CommandFunctions["deliver_sm"] = func(e *ESME, p PDU) error {
		pdu_resp := NewDeliverSMResp().
			WithMessageId(<-message_ids).
			WithSequenceNumber(p.Header.SequenceNumber)
		if ! e.isReceiverState() {
			pdu_resp = pdu_resp.WithSMPPError(ESME_RINVBNDSTS)
		}
		_, err := e.Send(&pdu_resp)
		return err
	}

	esme.StartControlLoop()

	smsc_esme := smsc.ESMEs.Load().([]*ESME)[0]
	deliverSm := NewDeliverSM().WithMessage("Hello")
	_, err := smsc_esme.Send(&deliverSm)
	if err != nil {
		t.Errorf("couldn't send to server successfully: %v", err)
	}
	pdu, smsc_err := smsc_esme.receivePdu()
	if smsc_err != nil  {
		t.Errorf("Something failed: %v", smsc_err)
	}
	if pdu.Header == (Header{}) || pdu.Body.MandatoryParameter["message_id"] != "0" {
		t.Errorf("Didn't receive expected Pdu: %v", pdu)
	}
}

func messageIdsGenerator() chan string {
	message_ids := make(chan string)
	go func() {
		i := 0
		for {
			message_ids <- fmt.Sprint(i)
			i++
		}

	}()
	return message_ids
}

func GetSmscAndConnectEsme(t *testing.T) (*SMSC, *ESME, net.Conn) {
	smsc, err := GetSmscSimulatorServer()
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
