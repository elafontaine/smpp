package smpp

import (
	"bufio"
	"bytes"
	"io"
	"log"
)

type Body struct {
	mandatoryParameter map[string]interface{}
	optionalParameters []map[string]interface{}
}

type PDU struct {
	header Header
	body   Body
}

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
