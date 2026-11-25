package smpp

// Expose Data Structure to enable people to manipulate it.  We don't care if they don't respect SMPP protocols :)
type PDU struct {
	Header Header
	Body   Body
}

// ParsePdu decodes one whole PDU.
func ParsePdu(bytes []byte) (PDU, error) {
	header, err := parseHeader(bytes)
	if err != nil {
		return PDU{}, err
	}
	body, err := parseBody(header, bytes)
	return PDU{Header: header, Body: body}, err
}

// EncodePdu encodes a PDU, computing command_length.
func EncodePdu(obj PDU) ([]byte, error) {
	bodyBytes, err := encodeBody(obj)
	if err != nil {
		return nil, err
	}
	headerBytes, err := encodeHeader(obj, bodyBytes)
	if err != nil {
		return nil, err
	}
	return append(headerBytes, bodyBytes...), nil
}
