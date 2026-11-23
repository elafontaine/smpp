package smpp

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
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

// parseBody decodes the body. On malformed input it returns an error, never panics, along with
// whatever it decoded before the problem.
func parseBody(header Header, pdu_bytes []byte) (body Body, err error) {
	data := pdu_bytes[16:header.CommandLength]
	body = Body{MandatoryParameter: map[string]interface{}{}}
	if len(data) == 0 {
		return body, nil // e.g. a response with a non-zero command_status: SMPP lets it omit the body
	}
	r := bytes.NewReader(data)
	body.MandatoryParameter, err = extractMandatoryParameters(header, r)
	if err != nil {
		return body, err
	}
	if rest := data[len(data)-r.Len():]; len(rest) > 0 {
		body.OptionalParameters, err = extractOptionalParameters(rest)
	}
	return body, err
}

func extractOptionalParameters(optionalParameterBytes []byte) (params []map[string]interface{}, err error) {
	rest := optionalParameterBytes
	for len(rest) > 0 {
		var param map[string]interface{}
		param, err = extractSpecificOptionalParameter(rest)
		if err != nil {
			return params, err
		}
		params = append(params, param)
		rest = rest[4+param["length"].(int):] // tag (2) + length (2) + value
	}
	return params, nil
}

// extractSpecificOptionalParameter decodes the TLV at the start of parameterBytes. A tag missing
// from the table is kept as raw bytes under its hex code (e.g. "0x1405").
func extractSpecificOptionalParameter(parameterBytes []byte) (map[string]interface{}, error) {
	if len(parameterBytes) < 4 {
		return nil, fmt.Errorf("optional parameters: %d bytes left, a TLV needs at least 4", len(parameterBytes))
	}
	tagHex := hex.EncodeToString(parameterBytes[0:2])
	definition := optionalParameterTagByHex[tagHex]
	tag := "0x" + tagHex
	if name, ok := definition["name"].(string); ok {
		tag = name
	}
	length := int(binary.BigEndian.Uint16(parameterBytes[2:4]))
	if length > len(parameterBytes)-4 {
		return nil, fmt.Errorf("optional parameter %s: length %d but only %d bytes are left", tag, length, len(parameterBytes)-4)
	}
	raw := parameterBytes[4 : 4+length]
	var value interface{}
	switch definition["type"] {
	case "string":
		value = string(bytes.TrimSuffix(raw, []byte{0}))
	case "integer":
		if length != 1 && length != 2 && length != 4 {
			return nil, fmt.Errorf("optional parameter %s: integer of %d bytes isn't supported", tag, length)
		}
		value = readFixedWidthInteger(raw)
	default: // octet strings (message_payload...), bitmasks, tags unknown to the table
		value = string(raw)
	}
	return map[string]interface{}{
		"tag":    tag,
		"length": length,
		"value":  value,
	}, nil
}

func extractMandatoryParameters(header Header, r *bytes.Reader) (map[string]interface{}, error) {
	mandatoryParameterMap := map[string]interface{}{}
	for _, mandatory_params := range mandatoryParameterLists[header.CommandId] {
		name := mandatory_params["name"].(string)
		fail := func(format string, a ...interface{}) error {
			return fmt.Errorf("%s of %s pdu: %s", name, header.CommandId, fmt.Sprintf(format, a...))
		}
		switch mandatory_params["type"].(string) {
		case "string":
			if r.Len() == 0 {
				return mandatoryParameterMap, fail("missing, the body ends early")
			}
			value, err := readCString(r)
			if err != nil {
				return mandatoryParameterMap, fail("missing NUL terminator")
			}
			mandatoryParameterMap[name] = value
		case "integer", "hex":
			raw := make([]byte, mandatory_params["max"].(int))
			if _, err := io.ReadFull(r, raw); err != nil {
				return mandatoryParameterMap, fail("missing, the body ends early")
			}
			mandatoryParameterMap[name] = readFixedWidthInteger(raw)
		case "xstring": // short_message, the only field whose length is another field (sm_length)
			smLength, _ := mandatoryParameterMap["sm_length"].(int)
			if smLength > r.Len() {
				return mandatoryParameterMap, fail("sm_length is %d but only %d bytes are left", smLength, r.Len())
			}
			shortMessage := make([]byte, smLength)
			_, _ = io.ReadFull(r, shortMessage) // can't fail: smLength <= r.Len()
			dataCoding, _ := mandatoryParameterMap["data_coding"].(int)
			text, err := DecodeText(dataCoding, shortMessage)
			if err != nil {
				text = string(shortMessage) // undecodable text: keep the raw bytes
			}
			mandatoryParameterMap["short_message"] = text
		}
	}
	return mandatoryParameterMap, nil
}

// readCString reads up to and including a NUL byte, and returns what came before it.
func readCString(r *bytes.Reader) (string, error) {
	var out []byte
	for {
		c, err := r.ReadByte()
		if err != nil {
			return "", err
		}
		if c == 0 {
			return string(out), nil
		}
		out = append(out, c)
	}
}

// readFixedWidthInteger reads a big-endian unsigned integer (1, 2 or 4 bytes in SMPP).
func readFixedWidthInteger(raw []byte) (value int) {
	for _, c := range raw {
		value = value<<8 | int(c)
	}
	return value
}

// Encoding functions, only EncodePdu should be public
func encodeBody(obj PDU) (bodyBytes []byte, err error) {
	if obj.Header.CommandId == "" {
		return nil, missingHeaderError
	}
	var mandatoryParamsBytes []byte
	var optionalParamsBytes []byte

	mandatoryParamsBytes, err = encodeMandatoryParameters(obj)
	if err != nil {
		return nil, err
	}
	bodyBytes = append(bodyBytes, mandatoryParamsBytes...)

	if len(obj.Body.OptionalParameters) > 0 {
		optionalParamsBytes, err = encodeOptionalParameters(obj)
		if err != nil {
			return nil, err
		}
		bodyBytes = append(bodyBytes, optionalParamsBytes...)
	}
	return bodyBytes, nil
}

func encodeOptionalParameters(obj PDU) (optionalParamsBytes []byte, err error) {
	for _, optionalParam := range obj.Body.OptionalParameters {
		var specificOptionalParamsBytes []byte
		specificOptionalParamsBytes, err = encodeSpecificOptionalParameter(optionalParam)
		if err != nil {
			return nil, err
		}
		optionalParamsBytes = append(optionalParamsBytes, specificOptionalParamsBytes...)
	}
	return optionalParamsBytes, nil
}

// encodeSpecificOptionalParameter encodes one TLV. The value decides the format:
//   - int: a big-endian integer, as wide as the table says (1 byte when it doesn't say);
//   - string: text, NUL-terminated when the table says the TLV is a string;
//   - []byte: sent as given.
func encodeSpecificOptionalParameter(optionalParam map[string]interface{}) ([]byte, error) {
	name, _ := optionalParam["tag"].(string)
	definition, ok := optionalParameterTagByName[name]
	if !ok {
		return nil, fmt.Errorf("unknown optional parameter %v, can't encode", optionalParam["tag"])
	}
	tag, err := hex.DecodeString(definition["hex"].(string))
	if err != nil {
		return nil, err
	}
	var value []byte
	switch v := optionalParam["value"].(type) {
	case int:
		value, err = appendFixedWidthInteger(nil, v, optionalParameterIntegerWidth(definition))
		if err != nil {
			return nil, fmt.Errorf("optional parameter %s is %d, %w, can't encode", name, v, err)
		}
	case string:
		value = []byte(v)
		if definition["type"] == "string" {
			value = append(value, 0)
		}
	case []byte:
		value = v
	default:
		return nil, fmt.Errorf("optional parameter %s: a value of type %T can't be encoded", name, v)
	}
	if len(value) > 0xffff {
		return nil, fmt.Errorf("optional parameter %s: %d bytes, more than a TLV can hold", name, len(value))
	}
	out := append(tag, byte(len(value)>>8), byte(len(value)))
	return append(out, value...), nil
}

// optionalParameterIntegerWidth gives an integer TLV's size: the table's "min", or 1 byte.
func optionalParameterIntegerWidth(definition map[string]interface{}) int {
	if width, ok := definition["min"].(int); ok {
		return width
	}
	return 1
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
				text, isString := value.(string)
				if !isString {
					return nil, fmt.Errorf("%v of %v pdu is a %T, want a string, can't encode", mandatoryParam["name"].(string), obj.Header.CommandId, value)
				}
				bodyBytes = append(bodyBytes, append([]byte(text), 0)...)
			}

			if mandatoryParam["type"].(string) == "integer" || mandatoryParam["type"].(string) == "hex" {
				integer, isInt := value.(int)
				if !isInt {
					return nil, fmt.Errorf("%v of %v pdu is a %T, want an int, can't encode", mandatoryParam["name"].(string), obj.Header.CommandId, value)
				}
				width := mandatoryParam["max"].(int)
				bodyBytes, err = appendFixedWidthInteger(bodyBytes, integer, width)
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
	case nil: // no short_message: a PDU without one, or one where it's missing
		return nil, obj.Body.MandatoryParameter["sm_length"], nil
	default:
		return nil, nil, fmt.Errorf("short_message of %v pdu is a %T, want a string or []byte, can't encode", obj.Header.CommandId, value)
	}
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
