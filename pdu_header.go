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

func parseHeader(bytes []byte) (header Header, err error) {
	length, err := verifyLength(bytes)
	if err != nil {
		return header, err
	}
	commandId, err := extractCommandID(bytes)
	if err != nil {
		return header, err
	}
	commandStatus, err := extractCommandStatus(bytes)
	header = Header{
		commandLength:  length,
		commandId:      commandId,
		commandStatus:  commandStatus,
		sequenceNumber: 0,
	}
	return header, err

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

func verifyLength(fixture []byte) (int, error) {
	if len(fixture) > 3 {
		pdu_length := int(binary.BigEndian.Uint32(fixture[0:4]))
		if len(fixture) < pdu_length {
			return 0, fmt.Errorf("invalid PDU Length for pdu : %v", hex.EncodeToString(fixture))
		}
		return pdu_length, nil
	}

	return 0, fmt.Errorf("invalid length parameter")
}
