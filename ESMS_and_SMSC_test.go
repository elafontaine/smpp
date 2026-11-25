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
			smsc, _, Esme := connectEsmeAndSmscTogether(t)
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
			if Esme.state.GetState() != smsc.Connections()[0].state.GetState() {
				t.Errorf("The state isn't the same on the SMSC connection and ESME")
			}
		})
	}
}

func TestReactionFromSmscOnFirstPDUForDefaultBehaviour(t *testing.T) {
	bindReceiver := NewBindReceiver().WithSystemId(validSystemID).WithPassword(validPassword)
	bindTransceiver := NewBindTransceiver().WithSystemId(validSystemID).WithPassword(validPassword)
	bindTransmitter := NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword)
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
		{"TestEsmeCanBindWithSmscAsAReceiver", args{&bindReceiver}, NewBindReceiverResp().WithSystemId(validSystemID)},
		{"TestEsmeCanBindWithSmscAsATransmitter", args{&bindTransmitter}, NewBindTransmitterResp().WithSystemId(validSystemID)},
		{"TestEsmeCanBindWithSmscAsATransceiver", args{&bindTransceiver}, NewBindTransceiverResp().WithSystemId(validSystemID)},
		{"TestSMSCRejectWithWrongUserName", args{&bindWrongUserName}, NewBindReceiverResp().WithSMPPError(ESME_RBINDFAIL).WithSystemId(invalidUserName)},
		{"TestSubmitSMOnNonBoundedBindIsReturningInvalidBindStatus", args{&SubmitSMUnbound}, NewSubmitSMResp().WithSMPPError(ESME_RINVBNDSTS).WithMessageId("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, _, Esme := connectEsmeAndSmscTogether(t)
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
			smsc, _, Esme := connectEsmeAndSmscTogether(t)
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

func assertWeHaveActiveConnections(smsc *SMSC, number_of_connections int) (is_right_number bool) {
	if smsc.GetNumberOfConnection() == number_of_connections {
		return true
	} else {
		return false
	}
}

func GetSmscSimulatorServer() (smsc *SMSC, err error) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	smsc = NewSMSC(serverSocket, validSystemID, validPassword)
	return smsc, err
}

func StartSmscSimulatorServerAndAccept() (smsc *SMSC, err error) {
	smsc, err = GetSmscSimulatorServer()
	smsc.Start()
	return smsc, err
}

func connectEsmeAndSmscTogether(t *testing.T) (*SMSC, net.Conn, *ESME) {
	smsc, err := StartSmscSimulatorServerAndAccept()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(connType, smsc.listener.Addr().String())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 1)
	smsc_connection := smsc.Connections()[0].clientSocket
	return smsc, smsc_connection, Esme
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC, count int) {
	for smsc.GetNumberOfConnection() != count {
		time.Sleep(0)
	}
}
