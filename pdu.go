package smpp

import (
	"bufio"
	"bytes"
)

type Body struct {
	mandatoryParameter map[string]interface{}
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
	r := bytes.NewReader(pdu_bytes[16:])
	scan := bufio.NewReader(r)
	body = Body{mandatoryParameter: map[string]interface{}{}}
	body.mandatoryParameter = extractMandatoryParameters(header, scan)
	return body, err
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
	}
	return mandatoryParameterMap
}
