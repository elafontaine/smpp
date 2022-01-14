package smpp

import (
	"bytes"
	"encoding/binary"
	"fmt"
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
	smsc, _, Esme ,_ := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindTransmiter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", LastError)
	}
	Pdu, _ := EncodePdu(NewSubmitSM())
	_, LastError = Esme.clientSocket.Write(Pdu)
	if LastError != nil {
		t.Errorf("Error writing : %v", LastError)
	}
	readBuf, LastError := readPduBytesFromConnection(smsc.connections.Load().([]net.Conn)[0], time.Now().Add(1*time.Second))
	if LastError != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	if !bytes.Equal(readBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive the first PDU we sent")
	}
	readSecondPdu, LastError := readPduBytesFromConnection(smsc.connections.Load().([]net.Conn)[0], time.Now().Add(1*time.Second))
	if !bytes.Equal(readSecondPdu, Pdu) || LastError != nil {
		t.Errorf("We didn't read the second PFU we sent correctly")
	}
}

func TestESMEIsBound(t *testing.T) {
	smsc, smsc_connection, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindTransmiter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", LastError)
	}
	handleBindOperation(smsc_connection, t)
	esmeReceivedBuf, err := readPduBytesFromConnection(Esme.clientSocket, time.Now().Add(1*time.Second))
	if err != nil {
		t.Errorf("Couldn't receive on the response on the ESME")
	}
	resp, err := ParsePdu(esmeReceivedBuf)
	if err != nil {
		t.Errorf("Couldn't parse received PDU")
	}
	if resp.header.commandStatus == ESME_ROK && resp.header.commandId == "bind_transmitter_resp" {
		Esme.state = BOUND_TX
	} else {
		t.Errorf("The answer received wasn't OK!")
	}
	if state := Esme.getConnectionState(); state != BOUND_TX {
		t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, err)
	}

}

func TestEsmeCanBindWithSmscAsAReceiver(t *testing.T) {
	smsc, smsc_connection, Esme , _ := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindReceiver("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %v", LastError)
	}
	handleBindOperation(smsc_connection, t)
	err := handleBindResponse(&Esme)
	if err != nil {
		t.Errorf("Error handling the answer from SMSC : %v", err)
	}
	if state := Esme.getConnectionState(); state != BOUND_RX {
		t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, err)
	}
}

func handleBindResponse(Esme *ESME) error {
	esmeReceivedBuf, err := readPduBytesFromConnection(Esme.clientSocket, time.Now().Add(1*time.Second))
	if err != nil {
		return err
	}
	resp, err := ParsePdu(esmeReceivedBuf)
	if err != nil {
		return err
	}
	if resp.header.commandStatus == ESME_ROK && resp.header.commandId == "bind_receiver_resp" {
		Esme.state = BOUND_RX
	} else {
		err = fmt.Errorf("The answer received wasn't OK or not the type we expected!")
	}
	return err
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
	err2 := Esme2.bindTransmiter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
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

func TestCanWeAvoidCallingAcceptExplicitlyOnEveryConnection(t *testing.T) {
	smsc, _, Esme, _ := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmiter("SystemId", "Password") //Should we expect the bind_transmitter to return only when the bind is done and valid?
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
	smsc, _, Esme , _ := ConnectEsmeAndSmscTogether(t)
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

func readPduBytesFromConnection(ConnectionSocket net.Conn, timeout time.Time) ([]byte, error) {
	buffer := bytes.Buffer{}
	err := ConnectionSocket.SetDeadline(timeout)
	if err != nil {
		return nil, err
	}
	readLengthBuffer := make([]byte, 4)
	_, err = ConnectionSocket.Read(readLengthBuffer)
	if err == nil {
		length := int(binary.BigEndian.Uint32(readLengthBuffer))
		if length <= 4 {
			return nil, fmt.Errorf("Received malformed packet : %v", readLengthBuffer)
		}
		readBuf := make([]byte, length-4)
		_, err = ConnectionSocket.Read(readBuf)
		buffer.Write(readLengthBuffer)
		buffer.Write(readBuf)
	}

	return buffer.Bytes(), err
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
	bindResponse, err := EncodePdu(bindResponsePdu)
	if err != nil {
		t.Errorf("Encoding bind response failed : %v", err)
	}
	_, err = smsc_connection.Write(bindResponse)
	if err != nil {
		t.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
}

func IsBindOperation(receivedPdu PDU) bool {
	switch receivedPdu.header.commandId {
	case "bind_transmitter",
		"bind_receiver",
		"bind_transceiver":
		return true
	}
	return false
}
