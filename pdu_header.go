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

func parseHeader(bytes []byte) (Header, error) {
	length, err := verifyLength(bytes)
	if err != nil {
		return Header{}, err
	}
	commandId, known := commandIdByHex[hex.EncodeToString(bytes[4:8])]
	if !known {
		return Header{}, fmt.Errorf("%w %s", errUnknownCommandID, hex.EncodeToString(bytes[4:8]))
	}
	commandStatus, known := commandStatusByHex[hex.EncodeToString(bytes[8:12])]
	if !known {
		return Header{}, fmt.Errorf("unknown command status %s", hex.EncodeToString(bytes[8:12]))
	}
	return Header{
		CommandLength:  length,
		CommandId:      commandId["name"],
		CommandStatus:  commandStatus["name"],
		SequenceNumber: int(binary.BigEndian.Uint32(bytes[12:16])),
	}, nil
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
