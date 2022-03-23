package smpp

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	smsc.Close()
}

func TestClosingOneConnectionCloseOnSMSCSide(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	smsc_connection := smsc.ESMEs.Load().([]*ESME)[0]

	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 1)
	Esme.Close()
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 0)
	for smsc_connection.GetEsmeState() != CLOSED {
		time.Sleep(200 * time.Millisecond)
	}
	closeResult := smsc_connection.clientSocket.Close()
	if smsc_connection.GetEsmeState() != CLOSED || !errors.Is(closeResult, net.ErrClosed) {
		t.Errorf("Connection didn't close cleanly!")
	}
}

func TestCanWeAvoidCallingAcceptExplicitlyOnEveryConnection(t *testing.T) {
	smsc, _, Esme := connectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr(), connType)
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 2)
	resp_pdu, err2 := Esme2.BindTransmitter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", err2)
	}
	expectedBuf := NewBindTransmitterResp().
		WithSystemId(validSystemID).
		WithSequenceNumber(1).
		WithSMPPError(ESME_ROK)
	expectedBuf.Header.CommandLength = 25
	comparePdu(*resp_pdu, expectedBuf, t)
	if !assertWeHaveActiveConnections(smsc, 2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}

func AssertSmscIsClosedAndClean(smsc *SMSC, t *testing.T) {
	assertListenerIsClosed(smsc, t)
	assertAllRemainingConnectionsAreClosed(smsc, t)
}

func assertAllRemainingConnectionsAreClosed(smsc *SMSC, t *testing.T) {
	for _, conn := range smsc.ESMEs.Load().([]*ESME) {
		if conn.GetEsmeState() != CLOSED {
			t.Error("At least one connection wasn't closed!")
		}
	}
}

func assertListenerIsClosed(smsc *SMSC, t *testing.T) {
	if err := smsc.listeningSocket.Close(); err == nil {
		t.Errorf("The listening socket wasn't closed! %v", err)
	}
}
