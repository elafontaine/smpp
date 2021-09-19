package smpp

/* Sane Defaults objects */
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
func NewSubmitSM() *PDU {
	header := Header{
		commandLength:  0,
		commandId:      "submit_sm",
		commandStatus:  "ESME_ROK",
		sequenceNumber: 0,
	}
	body := defaultSubmitSmBody()
	return &PDU{header: header, body: body}
}

/* Builder Pattern associated functions */
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

func (p PDU) WithInterfaceVersion(i int) PDU {
	p.body.mandatoryParameter["interface_version"] = i
	return p
}

func (p PDU) WithAddressNpi(i int) PDU {
	p.body.mandatoryParameter["addr_npi"] = i
	return p
}
func (p PDU) WithAddressTon(i int) PDU {
	p.body.mandatoryParameter["addr_ton"] = i
	return p
}

func (p PDU) WithSourceAddressNpi(i int) PDU {
	p.body.mandatoryParameter["source_addr_npi"] = i
	return p
}
func (p PDU) WithSourceAddressTon(i int) PDU {
	p.body.mandatoryParameter["source_addr_ton"] = i
	return p
}
func (p PDU) WithSourceAddress(s string) PDU {
	p.body.mandatoryParameter["source_addr"] = s
	return p
}
func (p PDU) WithDestinationAddressNpi(i int) PDU {
	p.body.mandatoryParameter["dest_addr_npi"] = i
	return p
}
func (p PDU) WithDestinationAddressTon(i int) PDU {
	p.body.mandatoryParameter["dest_addr_ton"] = i
	return p
}
func (p PDU) WithDestinationAddress(s string) PDU {
	p.body.mandatoryParameter["destination_addr"] = s
	return p
}

func (p PDU) WithDataCoding(i int) PDU {
	p.body.mandatoryParameter["data_coding"] = i
	return p
}
func (p PDU) WithMessage(s string) PDU {
	p.body.mandatoryParameter["short_message"] = s
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

func defaultSubmitSmBody() Body {
	body := Body{
		mandatoryParameter: map[string]interface{}{
			"service_type":            "",
			"source_addr_ton":         0,
			"source_addr_npi":         0,
			"source_addr":             "",
			"dest_addr_ton":           0,
			"dest_addr_npi":           0,
			"destination_addr":        "",
			"esm_class":               0,
			"protocol_id":             0,
			"priority_flag":           0,
			"schedule_delivery_time":  "",
			"validity_period":         "",
			"registered_delivery":     0,
			"replace_if_present_flag": 0,
			"data_coding":             0,
			"sm_default_msg_id":       0,
			"sm_length":               0,
			"short_message":           "",
		},
		optionalParameters: nil,
	}
	return body

}
