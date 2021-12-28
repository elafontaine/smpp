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
	if err != nil {
		return header, err
	}
	sequenceNumber, err := extractSequenceNumber(bytes)
	header = Header{
		commandLength:  length,
		commandId:      commandId,
		commandStatus:  commandStatus,
		sequenceNumber: sequenceNumber,
	}
	return header, err

}

func extractSequenceNumber(bytes []byte) (sequenceNumber int, err error) {
	sequenceNumber = int(binary.BigEndian.Uint32(bytes[12:16]))
	return sequenceNumber, err
}

func extractCommandStatus(bytes []byte) (string, error) {
	commandStatus := hex.EncodeToString(bytes[8:12])
	if value, ok := commandStatusByHex[commandStatus]; ok {
		return value["name"], nil
	}
	return commandStatus, fmt.Errorf("unknown command status %s", commandStatus)
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

func encodeHeader(obj PDU, bodyBytes []byte) (headerBytes []byte, err error) {
	headerBytes = []byte{}
	commandIdBytes, _ := hex.DecodeString(commandIdByName[obj.header.commandId]["hex"])
	headerBytes = append(headerBytes, commandIdBytes...)
	commandStatusBytes, _ := hex.DecodeString(commandStatusByName[obj.header.commandStatus]["hex"])
	headerBytes = append(headerBytes, commandStatusBytes...)
	sequence_number_buffer := make([]byte, 4)
	binary.BigEndian.PutUint32(sequence_number_buffer, uint32(obj.header.sequenceNumber))
	headerBytes = append(headerBytes, sequence_number_buffer...)
	length := len(bodyBytes) + 16
	lengthBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(lengthBytes, uint32(length))
	headerBytes = append(lengthBytes, headerBytes...)
	return headerBytes, err
}
