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
	Pdu, _ := EncodePdu(NewSubmitSM())
	_, LastError = Esme.clientSocket.Write(Pdu)
	if LastError != nil {
		t.Errorf("Error writing : %v", LastError)
	}
	readBuf, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	if !bytes.Equal(readBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive the first PDU we sent")
	}
	readSecondPdu, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if !bytes.Equal(readSecondPdu, Pdu) || LastError != nil {
		t.Errorf("We didn't read the second PFU we sent correctly")
	}
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
			smsc, smsc_connection, Esme, _ := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			LastError := tt.args.bind_pdu(Esme, validSystemID, validPassword)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			err := handleBindOperation(&smsc_connection)
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
		})
	}
}

func TestReactionFromSmscOnFirstPDU(t *testing.T) {
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
		{"TestSMSCRejectWithWrongUserName", args{&bindWrongUserName}, NewBindReceiverResp().withSMPPError(ESME_RBINDFAIL).WithSystemId(invalidUserName)},
		{"TestSubmitSMOnNonBoundedBindIsReturningInvalidBindStatus", args{&SubmitSMUnbound}, NewSubmitSMResp().withSMPPError(ESME_RINVBNDSTS).WithMessageId("")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, smsc_connection, Esme, _ := ConnectEsmeAndSmscTogether(t)
			defer CloseAndAssertClean(smsc, Esme, t)

			LastError := Esme.send(tt.args.bind_pdu)

			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			err := handleBindOperation(&smsc_connection)
			if err != nil {
				t.Logf("Error handling the binding operation on SMSC : %v", err)
			}
			pduResp, err := waitForBindResponse(Esme)
			tt.wantSMSCResp.header.sequenceNumber = pduResp.header.sequenceNumber
			tt.wantSMSCResp.header.commandLength = pduResp.header.commandLength
			if err != nil && pduResp.header.commandStatus != tt.wantSMSCResp.header.commandStatus {
				t.Errorf("Error handling the answer from SMSC : %v", err)
			}
			comparePdu(*pduResp, tt.wantSMSCResp, t)
		})
	}
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

	readBuf2, err3 := readPduBytesFromConnection(smsc.connections.Load().([]net.Conn)[1], time.Now().Add(1*time.Second))

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindReceiver().WithSystemId(validSystemID).WithPassword(validPassword))
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

	readBuf2, err3 := readPduBytesFromConnection(smsc.connections.Load().([]net.Conn)[1], time.Now().Add(1*time.Second))

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	if !bytes.Equal(readBuf2, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc, 2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}

func TestSmscCanRefuseConnectionHavingWrongCredentials(t *testing.T) {
	smsc, smsc_connection, esme, _ := ConnectEsmeAndSmscTogether(t)
	defer CloseAndAssertClean(smsc, esme, t)

	go handleBindOperation(&smsc_connection)

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
	for _, conn := range smsc.connections.Load().([]net.Conn) {
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
	esme = ESME{clientSocket, OPEN}
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
	smsc_connection := smsc.connections.Load().([]net.Conn)[0]
	return smsc, smsc_connection, &Esme, err
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC, count int) {
	for smsc.GetNumberOfConnection() < count {
		time.Sleep(0)
	}
}

func handleConnection(conn *net.Conn) {
	err := handleBindOperation(conn)
	if err != nil {
		InfoSmppLogger.Printf("Issue on Connection %v\n", conn)
	}
}

func handleBindOperation(smsc_connection *net.Conn) (formated_error error) {
	readBuf, LastError := readPduBytesFromConnection(*smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		return fmt.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	receivedPdu, err := ParsePdu(readBuf)
	if err != nil {
		return fmt.Errorf("Couldn't parse received PDU!")
	}
	bindResponsePdu := receivedPdu.WithCommandId(receivedPdu.header.commandId + "_resp")
	ABindOperation := IsBindOperation(receivedPdu)
	if !ABindOperation {
		formated_error = fmt.Errorf("We didn't received expected bind operation")
		bindResponsePdu = bindResponsePdu.WithMessageId("").withSMPPError(ESME_RINVBNDSTS)
	}
	if ABindOperation {
		if !receivedPdu.isSystemId(validSystemID) || !receivedPdu.isPassword(validPassword) {
			bindResponsePdu.header.commandStatus = ESME_RBINDFAIL
			InfoSmppLogger.Printf("We didn't received expected credentials")
		}
	}
	bindResponse, err := EncodePdu(bindResponsePdu)
	if err != nil {
		return fmt.Errorf("Encoding bind response failed : %v", err)
	}
	_, err = (*smsc_connection).Write(bindResponse)
	if err != nil {
		return fmt.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
	return formated_error
}
