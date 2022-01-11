package smpp

import (
	"bytes"
	"log"
	"net"
	"testing"
	"time"
)

const (
	connhost = "localhost"
	connport = "0"
	connType = "tcp"
	validSystemID = "SystemId"
	validPassword = "Password"
)

func TestServerInstantiationAndConnectClient(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", LastError)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc,1)
	readBuf, LastError := readFromConnection(smsc.connections.Load().([]net.Conn)[0])
	if LastError != nil{
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
}


func TestESMEIsBound(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", LastError)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc,1)
	smsc_connection := smsc.connections.Load().([]net.Conn)[0]
	readBuf, LastError := readFromConnection(smsc_connection)
	if LastError != nil{
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	bindResponse, err := EncodePdu(NewBindTransmitterResp().WithSystemId(validSystemID))
	if err != nil {
		t.Errorf("Encoding bind response failed : %v", err)
	}
	_, err = smsc_connection.Write(bindResponse)
	if err != nil {
		t.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
	esmeReceivedBuf, err := readFromConnection(Esme.clientSocket)
	if err != nil {
		t.Errorf("Couldn't receive on the response on the ESME")
	}
	resp, err := ParsePdu(esmeReceivedBuf)
	if err != nil {
		t.Errorf("Couldn't parse received PDU")
	}
	if resp.header.commandStatus == ESME_ROK && resp.header.commandId == "bind_transmitter_resp" {
		Esme.state = "BOUND_TX"
	}else {
		t.Errorf("The answer received wasn't OK!")
	}
	if state := Esme.getConnectionState(); state != "BOUND_TX"{
		t.Errorf("We couldn't get the state for our connection ; state = %v, err = %v", state, err)
	}

}


func TestEsmeCanBindWithSmscAsAReceiver(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer smsc.Close()
	defer Esme.Close()

	LastError := Esme.bindReceiver("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if LastError != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", LastError)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc,1)
	readBuf, LastError := readFromConnection(smsc.connections.Load().([]net.Conn)[0])
	if LastError != nil{
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v", LastError)
	}
	expectedBuf, err := EncodePdu(NewBindReceiver().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
}


func TestCanWeConnectTwiceToSMSC(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr()) 
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc,2)

	readBuf2, err3 := readFromConnection(smsc.connections.Load().([]net.Conn)[1])

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf2[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc,2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}


func TestCanWeAvoidCallingAcceptExplicitlyOnEveryConnection(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr()) 
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	WaitForConnectionToBeEstablishedFromSmscSide(smsc,2)

	readBuf2, err3 := readFromConnection(smsc.connections.Load().([]net.Conn)[1])

	if err != nil || err2 != nil || err3 != nil {
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v\n err3 =%v", err, err2, err3)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf2[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
	if !assertWeHaveActiveConnections(smsc,2) {
		t.Errorf("We didn't have the expected amount of connections!")
	}
}



func TestWeCloseAllConnectionsOnShutdown(t *testing.T) {
	smsc, _, Esme := ConnectEsmeAndSmscTogether(t)
	defer Esme.Close()
	defer smsc.Close()

	WaitForConnectionToBeEstablishedFromSmscSide(smsc,1)

	smsc.Close()

	assertListenerIsClosed(smsc, t)
	assertAllRemainingConnectionsAreClosed(smsc, t)
}

func assertAllRemainingConnectionsAreClosed(smsc *SMSC, t *testing.T) {
	for _, conn := range smsc.connections.Load().([]net.Conn) {
		if err:= conn.Close(); err == nil {
			t.Errorf("At least one connection wasn't closed! %v", err)
		}
	}
}

func assertListenerIsClosed(smsc *SMSC, t *testing.T) {
	if err:= smsc.listeningSocket.Close(); err == nil {
		t.Errorf("The listening socket wasn't closed! %v", err)
	}
}

func assertWeHaveActiveConnections(smsc *SMSC, number_of_connections int) (is_right_number bool){
	if smsc.GetNumberOfConnection() == number_of_connections {
		return true
	} else {
		return false
	}
}


func readFromConnection(ConnectionSocket net.Conn) ([]byte, error) {
	readBuf := make([]byte, 4096)
	err := ConnectionSocket.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = ConnectionSocket.Read(readBuf)
	return readBuf, err
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
			log.Printf("SMSC wasn't able to accept a new connection: %v",err)
		}
	}
}

func ConnectEsmeAndSmscTogether(t *testing.T) (*SMSC, error, ESME) {
	smsc, err := StartSmscSimulatorServerAndAccept()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	return smsc, err, Esme
}

func WaitForConnectionToBeEstablishedFromSmscSide(smsc *SMSC,count int) {
	for smsc.GetNumberOfConnection() < count {
		time.Sleep(0)
	}
}