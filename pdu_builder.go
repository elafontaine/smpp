package smpp

func NewBindTransmitter() *PDU {
	header := Header{
		commandLength:  0,
		commandId:      "bind_transmitter",
		commandStatus:  "ESME_ROK",
		sequenceNumber: 0,
	}
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
	return &PDU{header: header, body: body}
}

func (p PDU) withSystemId(s string) PDU {
	p.body.mandatoryParameter["system_id"] = s
	return p
}

func (p PDU) withPassword(s string) PDU {
	p.body.mandatoryParameter["password"] = s
	return p
}
