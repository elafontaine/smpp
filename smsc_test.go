package smpp

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, Esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	smsc.Close()
}

func TestClosingOneConnectionCloseOnSMSCSide(t *testing.T) {
	smsc, Esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	smsc_connection := smsc.Connections()[0]

	waitForConnections(t, smsc, 1)
	Esme.Close()
	waitForConnections(t, smsc, 0)
	if !eventually(func() bool { return smsc_connection.GetEsmeState() == CLOSED }) {
		t.Fatal("the SMSC side of the connection never closed")
	}
	closeResult := smsc_connection.clientSocket.Close()
	if smsc_connection.GetEsmeState() != CLOSED || !errors.Is(closeResult, net.ErrClosed) {
		t.Errorf("Connection didn't close cleanly!")
	}
}

func TestCanWeAvoidCallingAcceptExplicitlyOnEveryConnection(t *testing.T) {
	smsc, Esme := connectWithRunningSmsc(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme2, err := InstantiateEsme(connType, smsc.listener.Addr().String())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	waitForConnections(t, smsc, 2)
	resp_pdu, err2 := Esme2.BindTransmitter(contextWithin(t, time.Second), "SystemId", "Password")
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", err2)
	}
	expectedBuf := NewBindTransmitterResp().
		WithSystemId(validSystemID).
		WithSequenceNumber(1).
		WithSMPPError(ESME_ROK)
	expectedBuf.Header.CommandLength = 25
	comparePdu(resp_pdu, expectedBuf, t)
	if n := smsc.GetNumberOfConnection(); n != 2 {
		t.Errorf("the SMSC has %d connections, want 2", n)
	}
}

func AssertSmscIsClosedAndClean(smsc *SMSC, t *testing.T) {
	assertListenerIsClosed(smsc, t)
	assertAllRemainingConnectionsAreClosed(smsc, t)
}

func assertAllRemainingConnectionsAreClosed(smsc *SMSC, t *testing.T) {
	for _, conn := range smsc.Connections() {
		if conn.GetEsmeState() != CLOSED {
			t.Error("At least one connection wasn't closed!")
		}
	}
}

func assertListenerIsClosed(smsc *SMSC, t *testing.T) {
	if err := smsc.listener.Close(); err == nil {
		t.Errorf("The listening socket wasn't closed! %v", err)
	}
}
