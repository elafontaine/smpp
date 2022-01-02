package smpp

import (
	"bytes"
	"fmt"
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
	smsc, err := StartSmscSimulatorServer()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	defer smsc.Close()

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()
	err2 := Esme.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	err = smsc.AcceptNewConnectionFromSMSC()  
	readBuf, err2 := readFromConnection(smsc.connections.Load().([]net.Conn)[0])
	if err != nil || err2 != nil{
		t.Errorf("Couldn't read on a newly established Connection: \n err =%v\n err2 =%v", err, err2)
	}
	expectedBuf, err := EncodePdu(NewBindTransmitter().WithSystemId(validSystemID).WithPassword(validPassword))
	tempReadBuf := readBuf[0:len(expectedBuf)]
	if !bytes.Equal(tempReadBuf, expectedBuf) || err != nil {
		t.Errorf("We didn't receive what we sent")
	}
}


func TestCanWeConnectTwiceToSMSC(t *testing.T) {
	smsc, err := StartSmscSimulatorServer()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	defer smsc.Close()

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr()) 
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	err = smsc.AcceptNewConnectionFromSMSC()
	err2 = smsc.AcceptNewConnectionFromSMSC()

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
	smsc, err := StartSmscSimulatorServerAndAccept()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}
	defer smsc.Close()

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()

	Esme2, err := InstantiateEsme(smsc.listeningSocket.Addr()) 
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme2.Close()
	err2 := Esme2.bindTransmiter("SystemId", "Password")  //Should we expect the bind_transmitter to return only when the bind is done and valid? 
	if err2 != nil {
		t.Errorf("Couldn't write to the socket PDU: %s", err)
	}
	for smsc.GetNumberOfConnection() < 2 {
		time.Sleep(100 * time.Millisecond)
	}
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
	smsc, err := StartSmscSimulatorServer()
	if err != nil {
		t.Errorf("failed to start listening socket: %v", err)
	}
	
	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()
	_ = smsc.AcceptNewConnectionFromSMSC()

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


func readFromConnection(serverConnectionSocket net.Conn) ([]byte, error) {
	readBuf := make([]byte, 4096)
	err := serverConnectionSocket.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = serverConnectionSocket.Read(readBuf)
	return readBuf, err
}

func InstantiateEsme(serverAddress net.Addr) (esme ESME, err error) {
	clientSocket, err := net.Dial(connType, serverAddress.String())
	esme = ESME{clientSocket}
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
			println(fmt.Errorf("SMSC wasn't able to accept a new connection: %v",err))
		}
	}
}