package smpp

import (
	"errors"
	"net"
	"testing"
	"time"
)

func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	smsc.Close()
}

func TestClosingOneConnectionCloseOnSMSCSide(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)
	smsc_connection := smsc.ESMEs.Load().([]*ESME)[0]
	smsc.ensureCleanUpOfEsmes(smsc_connection)

	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 1)
	Esme.Close()
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 0)
	for smsc_connection.getEsmeState() != CLOSED {
		time.Sleep(200 * time.Millisecond)
	}
	closeResult := smsc_connection.clientSocket.Close()
	if smsc_connection.getEsmeState() != CLOSED || !errors.Is(closeResult, net.ErrClosed) {
		t.Errorf("Connection didn't close cleanly!")
	}

}

func AssertSmscIsClosedAndClean(smsc *SMSC, t *testing.T) {
	assertListenerIsClosed(smsc, t)
	assertAllRemainingConnectionsAreClosed(smsc, t)
}

func assertAllRemainingConnectionsAreClosed(smsc *SMSC, t *testing.T) {
	for _, conn := range smsc.ESMEs.Load().([]*ESME) {
		if conn.getEsmeState() != CLOSED {
			t.Error("At least one connection wasn't closed!")
		}
	}
}

func assertListenerIsClosed(smsc *SMSC, t *testing.T) {
	if err := smsc.listeningSocket.Close(); err == nil {
		t.Errorf("The listening socket wasn't closed! %v", err)
	}
}