package smpp

import (
	"bytes"
	"net"
	"testing"
	"time"
	"github.com/stretchr/testify/assert"
)

func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, smsc_connection, Esme := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	LastError := Esme.bindTransmitter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", LastError)
	}
	firstPdu := NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1)
	secondPdu := NewSubmitSM().WithSequenceNumber(2)
	sequence_number, LastError := Esme.send(&secondPdu)
	if sequence_number != 2 {
		t.Errorf("Sending sequence number isn't as expected !")
	}
	if LastError != nil {
		t.Errorf("Error writing : %v", LastError)
	}
	AssertReceivedPduIsSameAsExpected(smsc_connection, t, firstPdu)
	AssertReceivedPduIsSameAsExpected(smsc_connection, t, secondPdu)
}

func TestClosingEsmeCloseSocketAndDoesntBlock(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme.Close()
	time.Sleep(1)
	Esme.Close()
}

func TestSendingPduIncreaseSequenceNumberAcrossGoroutines(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	all_seq_numbers := make(chan int)
	seq_numbers_expected := []int{}
	seq_numbers_actual := []int{}
	iterations := 10
	for i:= 0; i<= iterations; i++ {
		go func(){
			enquireLink1 := NewEnquireLink()
			actual_seq_num, err1 := Esme.send(&enquireLink1)
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

func AssertReceivedPduIsSameAsExpected(smsc_connection net.Conn, t *testing.T, expectedPDU PDU) {
	ActualPdu, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("We didn't read the PDU we sent correctly : %v", LastError)
	}
	expectedBytes, LastError := EncodePdu(expectedPDU)
	if !bytes.Equal(ActualPdu, expectedBytes) || LastError != nil {
		t.Errorf("We didn't receive expected PDU (sequence Number wrong?) : %v", LastError)
	}
}
