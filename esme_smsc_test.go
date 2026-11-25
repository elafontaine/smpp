package smpp

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"
)

const (
	connhost        = "localhost"
	connport        = "0"
	connType        = "tcp"
	validSystemID   = "SystemId"
	validPassword   = "Password"
	invalidUserName = "InvalidUser"
)

func TestEsmeCanBindAsDifferentTypesWithSmsc(t *testing.T) {
	type args struct {
		bind_pdu func(e *ESME, ctx context.Context, sys, pass string) (PDU, error)
	}
	tests := []struct {
		name        string
		args        args
		wantBoundAs string
	}{
		{"TestEsmeCanBindWithSmscAsAReceiver", args{(*ESME).BindReceiver}, BOUND_RX},
		{"TestEsmeCanBindWithSmscAsATransmitter", args{(*ESME).BindTransmitter}, BOUND_TX},
		{"TestEsmeCanBindWithSmscAsATransceiver", args{(*ESME).BindTransceiver}, BOUND_TRX},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, Esme := connectWithRunningSmsc(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			resp_pdu, LastError := tt.args.bind_pdu(Esme, contextWithin(t, time.Second), validSystemID, validPassword)
			if LastError != nil {
				t.Errorf("Error sending to or handling the answer from SMSC : %v", LastError)
			}
			if resp_pdu.Header.CommandStatus != ESME_ROK {
				t.Errorf("Answer wasn't successful ! %v", resp_pdu)
			}
			if state := Esme.GetEsmeState(); state != tt.wantBoundAs {
				t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, LastError)
			}
			if Esme.GetEsmeState() != smsc.Connections()[0].GetEsmeState() {
				t.Errorf("The state isn't the same on the SMSC connection and ESME")
			}
		})
	}
}

func TestReactionFromSmscOnFirstPDUForDefaultBehaviour(t *testing.T) {
	bindWrongUserName := NewBindReceiver().WithSystemId(invalidUserName).WithPassword(validPassword)
	SubmitSMUnbound := NewSubmitSM()
	type args struct {
		bind_pdu *PDU
	}
	tests := []struct {
		name         string
		args         args
		wantSMSCResp PDU
	}{
		{"TestSMSCRejectWithWrongUserName", args{&bindWrongUserName}, NewBindReceiverResp().WithSMPPError(ESME_RBINDFAIL).WithSystemId(invalidUserName)},
		{"TestSubmitSMOnNonBoundedBindIsReturningInvalidBindStatus", args{&SubmitSMUnbound}, NewSubmitSMResp().WithSMPPError(ESME_RINVBNDSTS).WithMessageId("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, Esme := connectWithRunningSmsc(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			pduResp, err := Esme.Send(contextWithin(t, time.Second), *tt.args.bind_pdu)
			if err != nil {
				t.Fatalf("Couldn't get an answer from the SMSC: %v", err)
			}
			tt.wantSMSCResp.Header.SequenceNumber = pduResp.Header.SequenceNumber
			tt.wantSMSCResp.Header.CommandLength = pduResp.Header.CommandLength
			comparePdu(pduResp, tt.wantSMSCResp, t)
		})
	}
}

func TestReactionFromBindedEsmeAsSpecifiedBindState(t *testing.T) {
	type args struct {
		send_pdu   PDU
		bind_state string
	}
	tests := []struct {
		name         string
		args         args
		wantSMSCResp PDU
	}{
		{
			"Send SubmitSM when bind as transmitter return SubmitSMResp",
			args{NewSubmitSM().WithMessage("Hello"), BOUND_TX},
			NewSubmitSMResp().WithMessageId("1"),
		},
		{
			"Send SubmitSM when bind as receiver return SubmitSMResp but invalid bind status",
			args{NewSubmitSM().WithMessage("Hello"), BOUND_RX},
			NewSubmitSMResp().WithSMPPError(ESME_RINVBNDSTS).WithMessageId(""),
		},
		{
			"Send enquiry_link when bind as transmitter should return response",
			args{NewEnquireLink(), BOUND_TX},
			NewEnquireLinkResp(),
		},
		{
			"Send deliver_sm when bind as transmitter should return response",
			args{NewDeliverSM(), BOUND_TX},
			NewDeliverSMResp().WithMessageId("").WithSMPPError(ESME_RINVBNDSTS),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.name), func(t *testing.T) {
			smsc, Esme := connectWithRunningSmsc(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			Esme.state.SetState(tt.args.bind_state)
			smsc.Connections()[0].state.SetState(tt.args.bind_state)
			actualPdu, LastError := Esme.Send(contextWithin(t, time.Second), tt.args.send_pdu)
			if LastError != nil {
				t.Fatalf("Failed to get an answer : %v", LastError)
			}

			expectedPdu := tt.wantSMSCResp
			expectedPdu.Header.CommandLength = actualPdu.Header.CommandLength
			expectedPdu.Header.SequenceNumber = actualPdu.Header.SequenceNumber
			comparePdu(actualPdu, expectedPdu, t)
		})
	}
}

func CloseAndAssertClean(s *SMSC, e *ESME, t *testing.T) {
	e.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		t.Errorf("SMSC Shutdown() error = %v", err)
	}

	AssertSmscIsClosedAndClean(s, t)
}

// newSmsc returns an SMSC listening on localhost, not accepting connections yet.
func newSmsc(t *testing.T) *SMSC {
	t.Helper()
	listener, err := net.Listen(connType, connhost+":"+connport)
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	return NewSMSC(listener, validSystemID, validPassword)
}

// connectWithRunningSmsc starts an SMSC and connects an ESME to it: the SMSC side answers as it
// would in use.
func connectWithRunningSmsc(t *testing.T) (*SMSC, *ESME) {
	t.Helper()
	smsc := newSmsc(t)
	smsc.Start()
	esme, err := InstantiateEsme(connType, smsc.listener.Addr().String())
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	waitForConnections(t, smsc, 1)
	return smsc, esme
}

// connectWithSilentSmsc connects an ESME to an SMSC that isn't started: the connection is
// accepted by hand and its SMSC side reads nothing. The test drives that side through the
// returned socket.
func connectWithSilentSmsc(t *testing.T) (*SMSC, *ESME, net.Conn) {
	t.Helper()
	smsc := newSmsc(t)
	esme, err := InstantiateEsme(connType, smsc.listener.Addr().String())
	if err != nil {
		t.Fatalf("connecting: %v", err)
	}
	smscSide, err := smsc.acceptNewConnectionFromSMSC()
	if err != nil {
		t.Fatalf("accepting: %v", err)
	}
	return smsc, esme, smscSide.clientSocket
}

func waitForConnections(t *testing.T, smsc *SMSC, count int) {
	t.Helper()
	if !eventually(func() bool { return smsc.GetNumberOfConnection() == count }) {
		t.Fatalf("the SMSC has %d connections, want %d", smsc.GetNumberOfConnection(), count)
	}
}
