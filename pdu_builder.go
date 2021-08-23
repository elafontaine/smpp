package smpp

func NewBindTransmitter() *PDU {
	header := Header{
		commandLength:  0,
		commandId:      "bind_transmitter",
		commandStatus:  "ESME_ROK",
		sequenceNumber: 0,
	}
	body := defaultBindBody()
	return &PDU{header: header, body: body}
}

func NewBindReceiver() *PDU {
	header := Header{
		commandLength:  0,
		commandId:      "bind_receiver",
		commandStatus:  "ESME_ROK",
		sequenceNumber: 0,
	}
	body := defaultBindBody()
	return &PDU{header: header, body: body}
}

func NewBindTransceiver() *PDU {
	header := Header{
		commandLength:  0,
		commandId:      "bind_transceiver",
		commandStatus:  "ESME_ROK",
		sequenceNumber: 0,
	}
	body := defaultBindBody()
	return &PDU{header: header, body: body}
}

func (p PDU) WithSystemId(s string) PDU {
	p.body.mandatoryParameter["system_id"] = s
	return p
}

func (p PDU) WithPassword(s string) PDU {
	p.body.mandatoryParameter["password"] = s
	return p
}

func (p PDU) WithAddressRange(s string) PDU {
	p.body.mandatoryParameter["address_range"] = s
	return p
}

func (p PDU) WithSystemType(s string) PDU {
	p.body.mandatoryParameter["system_type"] = s
	return p
}

func defaultBindBody() Body {
	body := Body{
		mandatoryParameter: map[string]interface{}{
			"system_id":         "",
			"password":          "",
			"system_type":       "",
			"interface_version": 52,
			"addr_ton":          0,
			"addr_npi":          0,
			"address_range":     "",
		},
		optionalParameters: nil,
	}
	return body
}
