package smpp

import (
	"fmt"
	"log"
	"net"
	"os"
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

func init() {
	InfoSmppLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningSmppLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorSmppLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestEsmeCanBindAsDifferentTypesWithSmsc(t *testing.T) {
	type args struct {
		bind_pdu func(e *ESME, sys, pass string) error
	}
	tests := []struct {
		name        string
		args        args
		wantBoundAs string
	}{
		{"TestEsmeCanBindWithSmscAsAReceiver", args{(*ESME).bindReceiver}, BOUND_RX},
		{"TestEsmeCanBindWithSmscAsATransmitter", args{(*ESME).bindTransmitter}, BOUND_TX},
		{"TestEsmeCanBindWithSmscAsATransceiver", args{(*ESME).bindTransceiver}, BOUND_TRX},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			LastError := tt.args.bind_pdu(Esme, validSystemID, validPassword)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			LastError = handleOperations(smsc.ESMEs.Load().([]*ESME)[0])
			if LastError != nil {
				t.Errorf("Error handling the binding operation on SMSC : %v", LastError)
			}
			_, LastError = waitForBindResponse(Esme)
			if LastError != nil {
				t.Errorf("Error handling the answer from SMSC : %v", LastError)
			}
			if state := Esme.getEsmeState(); state != tt.wantBoundAs {
				t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, LastError)
			}
			if Esme.state.getState() != smsc.ESMEs.Load().([]*ESME)[0].state.getState() {
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
			smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			_, LastError := Esme.send(tt.args.bind_pdu)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			err := handleOperations(smsc.ESMEs.Load().([]*ESME)[0])
			if err != nil {
				t.Logf("Error handling the binding operation on SMSC : %v", err)
			}
			pduResp, err := waitForBindResponse(Esme)
			if err != nil {
				t.Logf("didn't receive a successful answer (might not be an issue): %v", err)
			}
			tt.wantSMSCResp.header.sequenceNumber = pduResp.header.sequenceNumber
			tt.wantSMSCResp.header.commandLength = pduResp.header.commandLength
			if err != nil && pduResp.header.commandStatus != tt.wantSMSCResp.header.commandStatus {
				t.Errorf("Error handling the answer from SMSC : %v", err)
			}
			comparePdu(*pduResp, tt.wantSMSCResp, t)
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
			args{NewEnquiryLink(), BOUND_TX},
			NewEnquiryLinkResp(),
		},
		{
			"Send deliver_sm when bind as transmitter should return response",
			args{NewDeliverSM(), BOUND_TX},
			NewDeliverSMResp().WithMessageId("").WithSMPPError(ESME_RINVBNDSTS),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.name), func(t *testing.T) {
			smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			Esme.state.setState <- tt.args.bind_state
			smsc.ESMEs.Load().([]*ESME)[0].state.setState <- tt.args.bind_state
			smsc.ensureCleanUpOfEsmes(smsc.ESMEs.Load().([]*ESME)[0])
			sequence_number, LastError := Esme.send(&tt.args.send_pdu)
			if LastError != nil {
				t.Errorf("Failed to send pdu : %v", LastError)
			}

			expectedPdu := tt.wantSMSCResp
			actualBytes, LastError := readPduBytesFromConnection(Esme.clientSocket, time.Now().Add(1*time.Second))

			if LastError != nil {
				t.Errorf("Failed to receive bytes : %v", LastError)
			}
			actualPdu, LastError := ParsePdu(actualBytes)
			expectedPdu.header.commandLength = actualPdu.header.commandLength
			expectedPdu.header.sequenceNumber = sequence_number
			if LastError != nil {
				t.Errorf("Couldn't parse received bytes : %v", LastError)
			}
			comparePdu(actualPdu, expectedPdu, t)
		})
	}
}


func CloseAndAssertClean(s *SMSC, e *ESME, t *testing.T) {
	e.Close()
	s.Close()

	AssertSmscIsClosedAndClean(s, t)
}

func assertWeHaveActiveConnections(smsc *SMSC, number_of_connections int) (is_right_number bool) {
	if smsc.GetNumberOfConnection() == number_of_connections {
		return true
	} else {
		return false
	}
}

func StartSmscSimulatorServer() (smsc *SMSC, err error) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	smsc = NewSMSC(&serverSocket, validSystemID, validPassword)
	return smsc, err
}

func StartSmscSimulatorServerAndAccept() (smsc *SMSC, err error) {
	smsc, err = StartSmscSimulatorServer()
	smsc.start()
	return smsc, err
}

func (s *SMSC) ensureCleanUpOfEsmes(e *ESME) {
	go func() {
		defer s.closeAndRemoveEsme(e)
		handleConnection(e)
	}()
}

func ConnectEsmeAndSmscTogether(t *testing.T) (*SMSC, net.Conn, *ESME) {
	smsc, err := StartSmscSimulatorServerAndAccept()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr(), connType)
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 1)
	smsc_connection := smsc.ESMEs.Load().([]*ESME)[0].clientSocket
	return smsc, smsc_connection, Esme
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC, count int) {
	for smsc.GetNumberOfConnection() != count {
		time.Sleep(0)
	}
}

func handleConnection(e *ESME) {
	for e.getEsmeState() != CLOSED {
		err := handleOperations(e)
		if err != nil {
			InfoSmppLogger.Printf("Issue on Connection: %v\n", err)
		}
		time.Sleep(0)
	}
}

func handleOperations(e *ESME) (formated_error error) {
	receivedPdu, formated_error := e.receivePdu()
	if formated_error != nil {
		return formated_error
	}

	ABindOperation := IsBindOperation(receivedPdu)
	if e.getEsmeState() == OPEN && !ABindOperation {
		ResponsePdu := receivedPdu.WithCommandId(receivedPdu.header.commandId + "_resp")
		formated_error = fmt.Errorf("We didn't received expected bind operation")
		ResponsePdu = ResponsePdu.WithMessageId("").WithSMPPError(ESME_RINVBNDSTS)
		bindResponse, err := EncodePdu(ResponsePdu)
		if err != nil {
			return fmt.Errorf("Encoding bind response failed : %v", err)
		}
		_, err = (e.clientSocket).Write(bindResponse)
		if err != nil {
			return fmt.Errorf("Couldn't write to the ESME from SMSC : %v", err)
		}
	}
	if receivedPdu.header.commandId == "deliver_sm" {
		ResponsePdu := receivedPdu.WithCommandId(receivedPdu.header.commandId + "_resp")
		formated_error = fmt.Errorf("We received a deliver_sm on a SMSC which isn't supposed to happen.")
		ResponsePdu = ResponsePdu.WithSMPPError(ESME_RINVBNDSTS).WithMessageId("")
		bindResponse, err := EncodePdu(ResponsePdu)
		if err != nil {
			return fmt.Errorf("Encoding bind response failed : %v", err)
		}
		_, err = (e.clientSocket).Write(bindResponse)
		if err != nil {
			return fmt.Errorf("Couldn't write to the ESME from SMSC : %v", err)
		}
	}
	if ABindOperation {
		formated_error = handleBindOperation(receivedPdu, e)
	}
	if receivedPdu.header.commandId == "submit_sm" {
		if e.isTransmitterState() {
			formated_error = ReplyToSubmitSM(*e, receivedPdu)
		} else {
			ResponsePdu := NewSubmitSMResp().WithSequenceNumber(receivedPdu.header.sequenceNumber)
			ResponsePdu = ResponsePdu.WithMessageId("").WithSMPPError(ESME_RINVBNDSTS)
			_, formated_error = e.send(&ResponsePdu)

		}
	}
	if receivedPdu.header.commandId == "enquire_link" {
		ResponsePdu := NewEnquiryLinkResp().WithSequenceNumber(receivedPdu.header.sequenceNumber)
		_, formated_error = e.send(&ResponsePdu)
	}
	return formated_error
}

func (e *ESME) isTransmitterState() bool {
	currentState := e.getEsmeState()
	return (currentState == BOUND_TX || currentState == BOUND_TRX)
}

func (e *ESME) isReceiverState() bool {
	currentState := e.getEsmeState()
	return (currentState == BOUND_RX || currentState == BOUND_TRX)
}



func handleBindOperation(receivedPdu PDU, e *ESME) error {
	ResponsePdu := receivedPdu.WithCommandId(receivedPdu.header.commandId + "_resp")
	if !receivedPdu.isSystemId(validSystemID) || !receivedPdu.isPassword(validPassword) {
		ResponsePdu.header.commandStatus = ESME_RBINDFAIL
		InfoSmppLogger.Printf("We didn't received expected credentials")
	}
	bindResponse, err := EncodePdu(ResponsePdu)
	if err != nil {
		return fmt.Errorf("Encoding bind response failed : %v", err)
	}
	err = SetESMEStateFromSMSCResponse(&ResponsePdu, e)
	if err != nil {
		InfoSmppLogger.Printf("Couldn't set the bind state on request!")
	}
	_, err = (e.clientSocket).Write(bindResponse)
	if err != nil {
		return fmt.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
	return nil
}

func ReplyToSubmitSM(e ESME, receivedPdu PDU) (err error) {
	submit_sm_resp_bytes, _ := EncodePdu(NewSubmitSMResp().WithMessageId("1").WithSequenceNumber(1))
	_, LastError := e.clientSocket.Write(submit_sm_resp_bytes)
	if LastError != nil {
		return fmt.Errorf("Couldn't write to esme socket: %v", LastError)
	}
	return nil
}
