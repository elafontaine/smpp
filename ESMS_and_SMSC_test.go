package smpp

import (
	"bytes"
	"log"
	"net"
	"testing"
	"time"
)

const (
	connhost      = "localhost"
	connport      = "0"
	connType      = "tcp"
	validSystemID = "SystemId"
	validPassword = "Password"
)

func TestSendingBackToBackPduIsInterpretedOkOnSmsc(t *testing.T) {
	smsc, smsc_connection, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

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
		bind_pdu func(e *ESME,sys, pass string) error
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
			defer smsc.Close()
			defer Esme.Close()

			LastError := tt.args.bind_pdu(&Esme,validSystemID,validPassword)
			
			if LastError != nil {
				t.Errorf("Couldn't write to the socket PDU: %v", LastError)
			}
			handleBindOperation(smsc_connection, t)
			err := handleBindResponse(&Esme)
			if err != nil {
				t.Errorf("Error handling the answer from SMSC : %v", err)
			}
			if state := Esme.getConnectionState(); state != tt.wantBoundAs {
				t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, err)
			}
		})
	}

}

func TestCanWeConnectTwiceToSMSC(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

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
	defer Esme.Close()
	defer smsc.Close()

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
	tempReadBuf := readBuf2[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc, 2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}

func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

	smsc.Close()

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
	esme = ESME{clientSocket, "OPEN"}
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
	for s.State != "CLOSED" {
		err := s.AcceptNewConnectionFromSMSC()
		if err != nil {
			log.Printf("SMSC wasn't able to accept a new connection: %v", err)
			break
		}
	}
}

func ConnectEsmeAndSmscTogether(t *testing.T) (*SMSC, net.Conn, ESME, error) {
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
	return smsc, smsc_connection, Esme, err
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC, count int) {
	for smsc.GetNumberOfConnection() < count {
		time.Sleep(0)
	}
}

func handleBindOperation(smsc_connection net.Conn, t *testing.T) {
	readBuf, LastError := readPduBytesFromConnection(smsc_connection, time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	receivedPdu, err := ParsePdu(readBuf)
	NotABindOperation := !IsBindOperation(receivedPdu)
	if NotABindOperation || err != nil {
		t.Errorf("We didn't received expected bind operation")
	}
	if !receivedPdu.isSystemId(validSystemID) || !receivedPdu.isPassword(validPassword) {
		t.Errorf("We didn't received expected credentials")
	}
	bindResponsePdu := NewBindTransmitterResp().WithSystemId(validSystemID)
	if receivedPdu.header.commandId == "bind_receiver" {
		bindResponsePdu.header.commandId = "bind_receiver_resp"
	}
	if receivedPdu.header.commandId == "bind_transceiver" {
		bindResponsePdu.header.commandId = "bind_transceiver_resp"
	}
	bindResponse, err := EncodePdu(bindResponsePdu)
	if err != nil {
		t.Errorf("Encoding bind response failed : %v", err)
	}
	_, err = smsc_connection.Write(bindResponse)
	if err != nil {
		t.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
}
