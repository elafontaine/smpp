package smpp

// Expose Data Structure to enable people to manipulate it.  We don't care if they don't respect SMPP protocols :)
type PDU struct {
	Header Header
	Body   Body
}

// Decoding Function, only ParsePdu should be public
func ParsePdu(bytes []byte) (pdu PDU, err error) {
	header, err := parseHeader(bytes)
	if err != nil {
		return
	}
	body, _ := parseBody(header, bytes)
	pdu = PDU{Header: header, Body: body}
	return
}

func (p *PDU) Write(b []byte) (n int, err error) {
	pdu, err := ParsePdu(b)
	if err != nil {
		return
	}
	
	p.Body = pdu.Body
	p.Header = pdu.Header
	n = p.Header.CommandLength
	return
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

func (p *PDU) Read(b []byte) (n int, err error) {
	pdu, err := EncodePdu(*p)
	if err != nil {
		return
	}
	n = copy(b, pdu)
	return
}