package smpp

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
)
type Error string
func (e Error) Error() string {
	return string(e)
}

const missingHeaderError = Error("PDU object malformed, missing headers")

// Expose Data Structure to enable people to manipulate it.
// We don't care if they don't respect SMPP protocols :)
// We use what we can (at least, try to)
type Body struct {
	MandatoryParameter map[string]interface{}
	OptionalParameters []map[string]interface{}
}

// Decoding Function (only ParsePdu should be public)
func parseBody(header Header, pdu_bytes []byte) (body Body, err error) {
	r := bytes.NewReader(pdu_bytes[16:header.CommandLength])
	scan := bufio.NewReader(r)
	body = Body{MandatoryParameter: map[string]interface{}{}}
	body.MandatoryParameter = extractMandatoryParameters(header, scan)
	optionalParameterBytes := make([]byte, scan.Buffered())
	bytesLeft, err := io.ReadFull(scan, optionalParameterBytes)
	if bytesLeft > 0 {
		body.OptionalParameters = extractOptionalParameters(optionalParameterBytes)
	}
	return body, err
}

func extractOptionalParameters(optionalParameterBytes []byte) (params []map[string]interface{}) {
	for i := 0; i < len(optionalParameterBytes); i++ {
		param, _ := extractSpecificOptionalParameter(optionalParameterBytes[i:])
		params = append(params, param)
		i += incrementBasedOnTlv(param)
	}
	return params
}

func incrementBasedOnTlv(param map[string]interface{}) int {
	return 2 + 2 + param["length"].(int) - 1 // tag (2) + length  (2) + value (value of length) - 1 (as we have post-increments)
}

func extractSpecificOptionalParameter(parameterBytes []byte) (nbOfBytes map[string]interface{}, err error) {
	identityTag := optionalParameterTagByHex[hex.EncodeToString(parameterBytes[0:2])]
	tag := identityTag["name"]
	length := int(binary.BigEndian.Uint16(parameterBytes[2:4]))
	var value interface{}
	if identityTag["type"] == "string" {
		lastBytePosition := length + 4
		lastByteIsNulByte := byte(0) == parameterBytes[lastBytePosition-1]
		if lastByteIsNulByte {
			value = string(parameterBytes[4 : lastBytePosition-1])
		} else {
			value = string(parameterBytes[4:lastBytePosition])
		}
	}
	if identityTag["type"] == "integer" {
		value = int(parameterBytes[4])
	}
	return map[string]interface{}{
		"tag":    tag,
		"length": length,
		"value":  value,
	}, err
}

func extractMandatoryParameters(header Header, scan *bufio.Reader) map[string]interface{} {
	mandatoryParameterMap := map[string]interface{}{}
	for _, mandatory_params := range mandatoryParameterLists[header.CommandId] {

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
			dataCoding, _ := mandatoryParameterMap["data_coding"].(int)
			text, err := DecodeText(dataCoding, shortMessage)
			if err != nil {
				text = string(shortMessage) // undecodable: keep the raw bytes (decode errors aren't reported yet)
			}
			mandatoryParameterMap["short_message"] = text
		}
	}
	return mandatoryParameterMap
}

// Encoding functions, only EncodePdu should be public
func encodeBody(obj PDU) (bodyBytes []byte, err error) {
	if obj.Header.CommandId == "" {
		return nil, missingHeaderError
	}
	var mandatoryParamsBytes []byte
	var optionalParamsBytes []byte

	mandatoryParamsBytes, err = encodeMandatoryParameters(obj)
	bodyBytes = append(bodyBytes, mandatoryParamsBytes...)

	if len(obj.Body.OptionalParameters) > 0 {
		optionalParamsBytes, err = encodeOptionalParameters(obj)
		bodyBytes = append(bodyBytes, optionalParamsBytes...)
	}
	if err != nil {
		return nil, err
	}
	return bodyBytes, err
}

func encodeOptionalParameters(obj PDU) (optionalParamsBytes []byte, err error) {
	for _, optionalParam := range obj.Body.OptionalParameters {
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
		integerByte := byte(int64(optionalParam["value"].(int)))
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
	shortMessage, smLength, err := encodeShortMessage(obj)
	if err != nil {
		return nil, err
	}
	for _, mandatoryParam := range mandatoryParameterLists[obj.Header.CommandId] {
		value, ok := obj.Body.MandatoryParameter[mandatoryParam["name"].(string)]
		if mandatoryParam["name"].(string) == "sm_length" {
			value, ok = smLength, smLength != nil
		}
		if ok {
			if mandatoryParam["type"].(string) == "string" {
				fieldBytes := []byte(value.(string))
				bodyBytes = append(bodyBytes, append(fieldBytes, 0)...)
			}

			if mandatoryParam["type"].(string) == "integer" || mandatoryParam["type"].(string) == "hex" {
				width := mandatoryParam["max"].(int)
				bodyBytes, err = appendFixedWidthInteger(bodyBytes, value.(int), width)
				if err != nil {
					err = fmt.Errorf("%v of %v pdu is %v, %w, can't encode", mandatoryParam["name"].(string), obj.Header.CommandId, value, err)
					return
				}
			}

			if mandatoryParam["type"].(string) == "xstring" {
				bodyBytes = append(bodyBytes, shortMessage...)
			}
		} else {
			err = fmt.Errorf("%v of %v pdu missing, can't encode", mandatoryParam["name"].(string), obj.Header.CommandId)
			return
		}
	}
	return
}

// encodeShortMessage returns short_message's bytes and the sm_length to send with them:
//   - text (string) is encoded with the PDU's data_coding, and sm_length is the encoded size,
//     whatever the map says;
//   - a []byte is sent as given, with the map's sm_length, so a peer can be tested with an
//     inconsistent PDU.
func encodeShortMessage(obj PDU) (message []byte, smLength interface{}, err error) {
	switch value := obj.Body.MandatoryParameter["short_message"].(type) {
	case string:
		if value == "" { // empty in every data_coding, even one EncodeText doesn't support
			return nil, 0, nil
		}
		dataCoding, _ := obj.Body.MandatoryParameter["data_coding"].(int)
		message, err = EncodeText(dataCoding, value)
		if err != nil {
			return nil, nil, fmt.Errorf("short_message of %v pdu: %w, can't encode", obj.Header.CommandId, err)
		}
		return message, len(message), nil
	case []byte:
		return value, obj.Body.MandatoryParameter["sm_length"], nil
	}
	return nil, obj.Body.MandatoryParameter["sm_length"], nil
}

// appendFixedWidthInteger appends value as a big-endian unsigned integer of width bytes,
// the way SMPP encodes its integer fields.
func appendFixedWidthInteger(buf []byte, value int, width int) ([]byte, error) {
	if value < 0 || (width < 8 && value >= 1<<(8*width)) {
		return buf, fmt.Errorf("doesn't fit in %d byte(s)", width)
	}
	switch width {
	case 1:
		return append(buf, byte(value)), nil
	case 2:
		return binary.BigEndian.AppendUint16(buf, uint16(value)), nil
	case 4:
		return binary.BigEndian.AppendUint32(buf, uint32(value)), nil
	}
	return buf, fmt.Errorf("unsupported integer width %d", width)
}
