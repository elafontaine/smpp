package smpp

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"testing"
	"time"
)

const (
	connhost = "localhost"
	connport = "0"
	connType = "tcp"
)

func TestServerInstantiationAndConnectClient(t *testing.T) {
	serverSocket, err := net.Listen(connType, connhost+":"+connport)
	smsc, err := StartSmscSimulatorServer(serverSocket)
	defer smsc.Close()
	if err != nil {
		t.Errorf("couldn't start server successfully: %v", err)
	}

	Esme, err := InstantiateEsme(smsc.listeningSocket.Addr())
	if err != nil {
		t.Errorf("couldn't connect client to server successfully: %v", err)
	}
	defer Esme.Close()
	pdu := PDU{
		header: Header{
			sequenceNumber: 1,
			commandId:      "submit_sm",
			commandStatus:  "ESME_OK",
			commandLength:  0,
		},
		body: Body{
			mandatoryParameter: map[string]interface{}{
				"service_type":            "",
				"source_addr_ton":         0,
				"source_addr_npi":         0,
				"source_addr":             "5555551234",
				"dest_addr_ton":           0,
				"dest_addr_npi":           0,
				"destination_addr":        "5555551234",
				"esm_class":               0,
				"protocol_id":             0,
				"priority_flag":           0,
				"schedule_delivery_time":  "",
				"validity_period":         "",
				"registered_delivery":     0,
				"replace_if_present_flag": 0,
				"data_coding":             0,
				"sm_default_msg_id":       0,
				"sm_length":               0,
				"short_message":           "",
			},
		},
	}
	expectedBytes, err := EncodePdu(pdu)
	if err != nil {
		t.Errorf("Couldn't get the bytes out of the PDU: %s", err)
	}
	_, err = Esme.clientSocket.Write(expectedBytes)
	readBuf, err := AcceptNewConnectionAndReadFromSMSC(smsc)
	if err != nil {
		t.Errorf("Couldn't read on a newly established Connection: %v", err)
	}
	if bytes.Equal(readBuf, expectedBytes) {
		t.Errorf("We didn't receive what we sent")
	}
}

func AcceptNewConnectionAndReadFromSMSC(smsc SMSC) (readBuf []byte, err error) {
	serverConnectionSocket, err := smsc.listeningSocket.Accept()
	if err != nil {
		err = errors.New(fmt.Sprintf("Couldn't establish connection on the server side successfully: %v", err))
		return nil, err
	}
	readBuf = make([]byte, 4096)
	err = serverConnectionSocket.SetDeadline(time.Now().Add(1 * time.Second))
	if err != nil {
		return nil, err
	}
	_, err = serverConnectionSocket.Read(readBuf)
	if err != nil {
		return nil, err
	}
	return readBuf, err
}

func InstantiateEsme(serverAddress net.Addr) (esme ESME, err error) {
	clientSocket, err := net.Dial(connType, serverAddress.String())
	esme = ESME{clientSocket}
	return esme, err
}

func StartSmscSimulatorServer(serverSocket net.Listener) (smsc SMSC, err error) {
	smsc = SMSC{serverSocket}
	return smsc, err
}
