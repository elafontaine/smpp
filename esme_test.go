package smpp

import (
	"bytes"
	"net"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, err := GetSmscSimulatorServer()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr(), connType)
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	smsc.acceptNewConnectionFromSMSC()
	smsc_connection := smsc.ESMEs.Load().([]*ESME)[0].clientSocket

	
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

func TestSendingPduIncreaseSequenceNumberAcrossGoroutines(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	all_seq_numbers := make(chan int)
	seq_numbers_expected := []int{}
	seq_numbers_actual := []int{}
	iterations := 100
	for i:= 0; i<= iterations; i++ {
		go func(){
			enquireLink1 := NewEnquireLink()
			actual_seq_num, err1 := Esme.Send(&enquireLink1)
			if err1 != nil {
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
