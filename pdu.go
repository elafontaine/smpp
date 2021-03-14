package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
)

// Expose Data Structure to enable people to manipulate it.  We don't care if they don't respect SMPP protocols :)
type Body struct {
	mandatoryParameter map[string]interface{}
	optionalParameters []map[string]interface{}
}

// Expose Data Structure to enable people to manipulate it.  We don't care if they don't respect SMPP protocols :)
type PDU struct {
	header Header
	body   Body
}

// Decoding Function (only ParsePdu should be public
func ParsePdu(bytes []byte) (pdu PDU, err error) {
	header, err3 := parseHeader(bytes)
	if err3 != nil {
		return PDU{}, err3
	}
	body, _ := parseBody(header, bytes)
	pdu = PDU{header: header, body: body}
	return pdu, err3
}

func parseBody(header Header, pdu_bytes []byte) (body Body, err error) {
	r := bytes.NewReader(pdu_bytes[16:header.commandLength])
	scan := bufio.NewReader(r)
	body = Body{mandatoryParameter: map[string]interface{}{}}
	body.mandatoryParameter = extractMandatoryParameters(header, scan)
	bytesLeft := scan.Buffered()
	if bytesLeft > 0 {
		body.optionalParameters = extractOptionalParameters(header, scan)
	}
	return body, err
}

func extractOptionalParameters(header Header, scan *bufio.Reader) []map[string]interface{} {
	params := []map[string]interface{}{
		{"tag": "receipted_message_id", "length": 6, "value": "11107"},
		{"tag": "message_state", "length": 1, "value": 2},
		{"tag": "delivery_failure_reason", "length": 1, "value": 0},
	}
	return params
}

func extractMandatoryParameters(header Header, scan *bufio.Reader) map[string]interface{} {
	mandatoryParameterMap := map[string]interface{}{}
	for _, mandatory_params := range mandatoryParameterLists[header.commandId] {

		if mandatory_params["type"].(string) == "string" {
			currentBytes, _ := scan.ReadBytes(0)
			mandatoryParameterMap[mandatory_params["name"].(string)] = string(currentBytes[:len(currentBytes)-1])
		}
		if mandatory_params["type"].(string) == "integer" || mandatory_params["type"].(string) == "hex" {
			currentBytes, _ := scan.ReadByte()
			mandatoryParameterMap[mandatory_params["name"].(string)] = int(currentBytes)
		}
		if mandatory_params["type"].(string) == "xstring" {
			smLength := mandatoryParameterMap["sm_length"].(int)
			shortMessage := make([]byte, smLength)
			numberRead, _ := io.ReadFull(scan, shortMessage)
			if numberRead != smLength {
				log.Printf("Unable to read full PDU, failure not yet implemented")
			}
			mandatoryParameterMap[mandatory_params["name"].(string)] = string(shortMessage)
		}
	}
	return mandatoryParameterMap
}

// Encoding functions, only EncodePdu should be public

func EncodePdu(obj PDU) (pdu_bytes []byte, err error) {
	bodyBytes, err := encodeBody(obj)
	if err != nil {
		return nil, err
	}
	headerBytes, err := encodeHeader(obj, bodyBytes)
	if err != nil {
		return nil, err
	}
	pdu_bytes = append(headerBytes, bodyBytes...)
	return pdu_bytes, err
}

func encodeBody(obj PDU) (bodyBytes []byte, err error) {
	var mandatoryParamsBytes []byte
	var optionalParamsBytes []byte
	if len(obj.body.mandatoryParameter) > 0 {
		mandatoryParamsBytes, err = encodeMandatoryParameters(obj)
		bodyBytes = append(bodyBytes, mandatoryParamsBytes...)
	}
	if len(obj.body.optionalParameters) > 0 {
		optionalParamsBytes, err = encodeOptionalParameters(obj)
		bodyBytes = append(bodyBytes, optionalParamsBytes...)
	}
	if err != nil {
		return nil, err
	}
	return bodyBytes, err
}

func encodeOptionalParameters(obj PDU) (optionalParamsBytes []byte, err error) {
	for _, optionalParam := range obj.body.optionalParameters {
		specificOptionalParamsBytes, _ := encodeSpecificOptionalParameter(optionalParam)
		optionalParamsBytes = append(optionalParamsBytes, specificOptionalParamsBytes...)

	}
	return optionalParamsBytes, err
}

func encodeSpecificOptionalParameter(optionalParam map[string]interface{}) (optionalParamsBytes []byte, err error) {
	parameterDefinitions := optionalParameterTagByName[optionalParam["tag"].(string)]
	var tag []byte
	tag, err = hex.DecodeString(parameterDefinitions["hex"].(string))
	lengthBuffer := make([]byte, 2)
	if parameterDefinitions["type"] == "integer" || parameterDefinitions["type"] == "hex" {
		//integerBuffer := []byte
		//binary.PutUvarint(integerBuffer, uint64(optionalParam["value"].(int)))
		integerByte := byte(int64(optionalParam["value"].(int)))
		//integerBuffer,_ = tlv.Marshal(uint64(optionalParam["value"].(int)),math.MaxUint32)
		binary.BigEndian.PutUint16(lengthBuffer, uint16(1))
		optionalParamsBytes = append(optionalParamsBytes, tag...)
		optionalParamsBytes = append(optionalParamsBytes, lengthBuffer...)
		optionalParamsBytes = append(optionalParamsBytes, integerByte)

	}
	if parameterDefinitions["type"] == "string" {
		fieldBytes := []byte(optionalParam["value"].(string))
		binary.BigEndian.PutUint16(lengthBuffer, uint16(len(fieldBytes)+1))
		optionalParamsBytes = append(optionalParamsBytes, tag...)
		optionalParamsBytes = append(optionalParamsBytes, lengthBuffer...)
		optionalParamsBytes = append(optionalParamsBytes, append(fieldBytes, 0)...)
	}
	return optionalParamsBytes, err
}

func encodeMandatoryParameters(obj PDU) (bodyBytes []byte, err error) {
	for _, mandatoryParam := range mandatoryParameterLists[obj.header.commandId] {
		if mandatoryParam["type"].(string) == "string" {
			fieldBytes := []byte(obj.body.mandatoryParameter[mandatoryParam["name"].(string)].(string))
			bodyBytes = append(bodyBytes, append(fieldBytes, 0)...)
		}
		if mandatoryParam["type"].(string) == "integer" || mandatoryParam["type"].(string) == "hex" {
			integerBuffer := make([]byte, mandatoryParam["max"].(int))
			binary.PutUvarint(integerBuffer, uint64(obj.body.mandatoryParameter[mandatoryParam["name"].(string)].(int)))
			bodyBytes = append(bodyBytes, integerBuffer...)
		}
	}
	return bodyBytes, err
}
