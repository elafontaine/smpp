package smpp

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
)

type Header struct {
	commandLength  int
	commandId      string
	commandStatus  string
	sequenceNumber int
}

type PDU struct {
	header Header
}

func ParsePdu(bytes []byte) (pdu PDU, err error) {
	pdu, err3 := parseHeader(bytes)
	return pdu, err3

}

func parseHeader(bytes []byte) (pdu PDU, err error) {
	p, err := verifyLength(bytes, pdu)
	if err != nil {
		return p, err
	}
	commandId, err := extractCommandID(bytes)
	if err != nil {
		return p, err
	}
	commandStatus, err := extractCommandStatus(bytes)
	pdu = PDU{
		header: Header{
			commandLength:  16,
			commandId:      commandId,
			commandStatus:  commandStatus,
			sequenceNumber: 0,
		},
	}

	return pdu, err

}

func extractCommandStatus(bytes []byte) (string, error) {
	commandStatus := hex.EncodeToString(bytes[8:12])
	if value, ok := commandStatusByHex[commandStatus]; ok {
		return value["name"], nil
	}
	return "", fmt.Errorf("unknown command status %s", commandStatus)
}

func extractCommandID(bytes []byte) (string, error) {
	commandId := hex.EncodeToString(bytes[4:8])
	if value, ok := commandIdByHex[commandId]; ok {
		return value["name"], nil
	}
	return "", fmt.Errorf("unknown command_id %s", commandId)
}

func verifyLength(fixture []byte, pdu PDU) (PDU, error) {
	if len(fixture) > 3 {
		pdu_length := int(binary.BigEndian.Uint32(fixture[0:4]))
		if len(fixture) < pdu_length {
			return pdu, fmt.Errorf("invalid PDU Length for pdu : %v", hex.EncodeToString(fixture))
		}
	}
	return PDU{}, nil
}
