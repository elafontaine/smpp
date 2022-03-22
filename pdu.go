package smpp

// Expose Data Structure to enable people to manipulate it.  We don't care if they don't respect SMPP protocols :)
type PDU struct {
	header Header
	body   Body
}

// Decoding Function, only ParsePdu should be public
func ParsePdu(bytes []byte) (pdu PDU, err error) {
	header, err3 := parseHeader(bytes)
	if err3 != nil {
		return PDU{}, err3
	}
	body, _ := parseBody(header, bytes)
	pdu = PDU{header: header, body: body}
	return pdu, err3
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
