package smpp

import (
	"bytes"
	"net"
	"testing"
	"time"
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
