package smpp

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
)

// errUnknownCommandID marks a PDU whose command_id isn't in the command table.
var errUnknownCommandID = errors.New("unknown command_id")

type Header struct {
	CommandLength  int
	CommandId      string
	CommandStatus  string
	SequenceNumber int
}

func parseHeader(bytes []byte) (header Header, err error) {
	length, err := verifyLength(bytes)
	if err != nil {
		return
	}
	commandId, err := extractCommandID(bytes)
	if err != nil {
		return
	}
	commandStatus, err := extractCommandStatus(bytes)
	if err != nil {
		return
	}
	sequenceNumber := extractSequenceNumber(bytes)
	header = Header{
		CommandLength:  length,
		CommandId:      commandId,
		CommandStatus:  commandStatus,
		SequenceNumber: sequenceNumber,
	}
	return
}

func extractSequenceNumber(bytes []byte) (sequenceNumber int) {
	sequenceNumber = int(binary.BigEndian.Uint32(bytes[12:16]))
	return
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
	return "", fmt.Errorf("%w %s", errUnknownCommandID, commandId)
}

func verifyLength(fixture []byte) (int, error) {
	if len(fixture) < 4 {
		return 0, fmt.Errorf("invalid length parameter")
	}
	pdu_length := int(binary.BigEndian.Uint32(fixture[0:4]))
	if pdu_length < 16 {
		return 0, fmt.Errorf("invalid command_length %d: shorter than the 16-byte header", pdu_length)
	}
	if len(fixture) < pdu_length {
		return 0, fmt.Errorf("invalid PDU Length for pdu : %v", hex.EncodeToString(fixture))
	}
	return pdu_length, nil
}

func encodeHeader(obj PDU, bodyBytes []byte) ([]byte, error) {
	commandId, known := commandIdByName[obj.Header.CommandId]
	if !known {
		return nil, fmt.Errorf("unknown command_id %q, can't encode", obj.Header.CommandId)
	}
	commandStatus, known := commandStatusByName[obj.Header.CommandStatus]
	if !known {
		return nil, fmt.Errorf("unknown command_status %q, can't encode", obj.Header.CommandStatus)
	}
	commandIdBytes, _ := hex.DecodeString(commandId["hex"])
	commandStatusBytes, _ := hex.DecodeString(commandStatus["hex"])
	headerBytes := binary.BigEndian.AppendUint32(nil, uint32(16+len(bodyBytes)))
	headerBytes = append(headerBytes, commandIdBytes...)
	headerBytes = append(headerBytes, commandStatusBytes...)
	return binary.BigEndian.AppendUint32(headerBytes, uint32(obj.Header.SequenceNumber)), nil
}
