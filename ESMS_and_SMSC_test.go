package smpp

import (
	"bytes"
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

var (
	WarningSmppLogger *log.Logger
	InfoSmppLogger    *log.Logger
	ErrorSmppLogger   *log.Logger
)

func init() {
	InfoSmppLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	WarningSmppLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime|log.Lshortfile)
	ErrorSmppLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, smsc_connection, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	LastError := Esme.bindTransmitter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", LastError)
	}
	secondPdu := NewSubmitSM()
	sequence_number, LastError := Esme.send(&secondPdu)
	if sequence_number != 2 {
		t.Errorf("Sending sequence number isn't as expected !")
	}
	if LastError != nil {
		t.Errorf("Error writing : %v", LastError)
	}
	readBuf, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1))
	if !bytes.Equal(readBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive the expected first PDU we sent")
	}
	readSecondPdu, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("We didn't read the second PFU we sent correctly")
	}
	expectedBytes, LastError := EncodePdu(secondPdu.WithSequenceNumber(2))
	if !bytes.Equal(readSecondPdu, expectedBytes) || LastError != nil {
		t.Errorf("We didn't receive expected PDU (sequence Number wrong?)")
	}
}

func TestEsmeCanBindAsDifferentTypesWithSmsc(t *testing.T) {
	t.Parallel()
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
			smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			LastError := tt.args.bind_pdu(Esme, validSystemID, validPassword)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			err := handleOperations(&smsc.ESMEs.Load().([]ESME)[0])
			if err != nil {
				t.Errorf("Error handling the binding operation on SMSC : %v", err)
			}
			_, err = waitForBindResponse(Esme)
			if err != nil {
				t.Errorf("Error handling the answer from SMSC : %v", err)
			}
			if state := Esme.getConnectionState(); state != tt.wantBoundAs {
				t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, err)
			}
			if Esme.state != smsc.ESMEs.Load().([]ESME)[0].state {
				t.Errorf("The state isn't the same on the SMSC connection and ESME")
			}
		})
	}
}

func TestReactionFromSmscOnFirstPDU(t *testing.T) {
	t.Parallel()
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
			smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			_, LastError := Esme.send(tt.args.bind_pdu)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			err := handleOperations(&smsc.ESMEs.Load().([]ESME)[0])
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

func TestReactionFromBindedEsmeAsTransmitter(t *testing.T) {
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
			NewSubmitSMResp().WithSequenceNumber(1).WithMessageId("1"),
		},
		{
			"Send SubmitSM when bind as receiver return SubmitSMResp but invalid bind status",
			args{NewSubmitSM().WithMessage("Hello"), BOUND_RX},
			NewSubmitSMResp().WithSequenceNumber(1).WithSMPPError(ESME_RINVBNDSTS).WithMessageId(""),
		},
		{
			"Send enquiry_link when bind as transmitter should return response",
			args{NewEnquiryLink(), BOUND_TX},
			NewEnquiryLinkResp().WithSequenceNumber(1),
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt.name), func(t *testing.T) {
			smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)
			
			Esme.state = tt.args.bind_state
			smsc.ESMEs.Load().([]ESME)[0].state = tt.args.bind_state
			go handleConnection(&smsc.ESMEs.Load().([]ESME)[0])
			_, LastError := Esme.send(&tt.args.send_pdu)
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
			if LastError != nil {
				t.Errorf("Couldn't parse received bytes : %v", LastError)
			}
			comparePdu(actualPdu, expectedPdu, t)
		})
	}
}

func ReplyToSubmitSM(e ESME, receivedPdu PDU) (err error) {
	submit_sm_resp_bytes, _ := EncodePdu(NewSubmitSMResp().WithMessageId("1").WithSequenceNumber(1))
	_, LastError := e.clientSocket.Write(submit_sm_resp_bytes)
	if LastError != nil {
		return fmt.Errorf("Couldn't write to esme socket: %v", LastError)
	}
	return nil
}

func TestCanWeConnectTwiceToSMSC(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindReceiver("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 2)

	readBuf2, err3 := readPduBytesFromConnection(smsc.ESMEs.Load().([]ESME)[1].clientSocket, time.Now().Add(1*time.Second))

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindReceiver().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1))
	if !bytes.Equal(readBuf2, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc, 2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}

func TestCanWeAvoidCallingAcceptExplicitlyOnEveryConnection(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmitter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 2)

	readBuf2, err3 := readPduBytesFromConnection(smsc.ESMEs.Load().([]ESME)[1].clientSocket, time.Now().Add(1*time.Second))

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword).WithSequenceNumber(1))
	if !bytes.Equal(readBuf2, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc, 2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}

func TestSmscCanRefuseConnectionHavingWrongCredentials(t *testing.T) {
	smsc, _, esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, esme, t)

	go handleConnection(&smsc.ESMEs.Load().([]ESME)[0])

	pdu_resp, err := esme.bindTransmitter2("WrongSystemId", validPassword) // this shouldn't return until we get a "OK" from SMSC
	if err != nil && pdu_resp == nil {
		t.Errorf("We didn't receive the expected error response from SMSC.")
	} else if err == nil && pdu_resp.header.commandStatus == ESME_ROK {
		t.Errorf("Error, we shouldn't succeed to bind with wrong password!")
	} else if pdu_resp.header.commandStatus != ESME_RBINDFAIL {
		t.Errorf("We didn't receive the expected error")
	}
}

func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, Esme, t)

	smsc.Close()
}

func CloseAndAssertClean(s *SMSC, e *ESME, t *testing.T) {
	e.Close()
	s.Close()

	AssertSmscIsClosedAndClean(s, t)
}

func AssertSmscIsClosedAndClean(smsc *SMSC, t *testing.T) {
	assertListenerIsClosed(smsc, t)
	assertAllRemainingConnectionsAreClosed(smsc, t)
}

func assertAllRemainingConnectionsAreClosed(smsc *SMSC, t *testing.T) {
	for _, conn := range smsc.ESMEs.Load().([]ESME) {
		if err := conn.Close(); err == nil {
			t.Errorf("At least one connection wasn't closed! %v", err)
		}
	}
}

func assertListenerIsClosed(smsc *SMSC, t *testing.T) {
	if err := smsc.listeningSocket.Close(); err == nil {
		t.Errorf("The listening socket wasn't closed! %v", err)
	}
}

func assertWeHaveActiveConnections(smsc *SMSC, number_of_connections int) (is_right_number bool) {
	if smsc.GetNumberOfConnection() == number_of_connections {
		return true
	} else {
		return false
	}
}

func InstantiateEsme(serverAddress net.Addr) (esme ESME, err error) {
	clientSocket, err := net.Dial(connType, serverAddress.String())
	esme = ESME{clientSocket, OPEN, 0}
	return esme, err
}

func StartSmscSimulatorServer() (smsc *SMSC, err error) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	smsc = NewSMSC(&serverSocket)
	return smsc, err
}

func StartSmscSimulatorServerAndAccept() (smsc *SMSC, err error) {
	smsc, err = StartSmscSimulatorServer()
	go smsc.AcceptAllNewConnection()
	return smsc, err
}

func (s *SMSC) AcceptAllNewConnection() {
	for s.State != CLOSED {
		_, err := s.AcceptNewConnectionFromSMSC()
		if err != nil {
			log.Printf("SMSC wasn't able to accept a new connection: %v", err)
			break
		}
	}
}

func ConnectEsmeAndSmscTogether(t *testing.T) (*SMSC, net.Conn, *ESME, error) {
	smsc, err := StartSmscSimulatorServerAndAccept()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc, 1)
	smsc_connection := smsc.ESMEs.Load().([]ESME)[0].clientSocket
	return smsc, smsc_connection, &Esme, err
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC, count int) {
	for smsc.GetNumberOfConnection() < count {
		time.Sleep(0)
	}
}

func handleConnection(e *ESME) {
	err := handleOperations(e)
	if err != nil {
		InfoSmppLogger.Printf("Issue on Connection: %v\n", err)
	}
}

func handleOperations(e *ESME) (formated_error error) {
	receivedPdu, formated_error := receivePduFromESME(e)
	if formated_error != nil {
		return formated_error
	}

	ABindOperation := IsBindOperation(receivedPdu)
	if e.state == OPEN && !ABindOperation {
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
	if ABindOperation {
		formated_error = handleBindOperation(receivedPdu, e)
	}
	if receivedPdu.header.commandId == "submit_sm" {
		if isTransmitterState(e) {
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

func isTransmitterState(e *ESME) bool {
	return (e.state == BOUND_TX || e.state == BOUND_TRX)
}
func isReceiverState(e *ESME) bool {
	return (e.state == BOUND_RX || e.state == BOUND_TRX)
}
func receivePduFromESME(e *ESME) (PDU, error) {
	readBuf, LastError := readPduBytesFromConnection(e.clientSocket, time.Now().Add(1*time.Second))
	if LastError != nil {
		return PDU{}, fmt.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	receivedPdu, err := ParsePdu(readBuf)
	if err != nil {
		return PDU{}, fmt.Errorf("Couldn't parse received PDU!")
	}
	return receivedPdu, nil
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
