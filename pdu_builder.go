package smpp

func defaultHeader() Header {
	return Header{
		CommandLength:  0,
		CommandId:      "generick_nack",
		CommandStatus:  ESME_ROK,
		SequenceNumber: 0,
	}
}

/* Sane Defaults objects */
func NewGenerickNack() PDU {
	header := defaultHeader()
	return PDU{Header: header}
}

func NewEnquireLink() PDU {
	header := defaultHeader()
	header.CommandId = "enquire_link"
	return PDU{Header: header}
}

func NewEnquireLinkResp() PDU {
	header := defaultHeader()
	header.CommandId = "enquire_link_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewBindTransmitter() PDU {
	header := defaultHeader()
	header.CommandId = "bind_transmitter"
	body := defaultBindBody()
	return PDU{Header: header, Body: body}
}

func NewBindReceiver() PDU {
	header := defaultHeader()
	header.CommandId = "bind_receiver"
	body := defaultBindBody()
	return PDU{Header: header, Body: body}
}

func NewBindTransceiver() PDU {
	header := defaultHeader()
	header.CommandId = "bind_transceiver"
	body := defaultBindBody()
	return PDU{Header: header, Body: body}
}

func NewBindTransmitterResp() PDU {
	header := defaultHeader()
	header.CommandId = "bind_transmitter_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewBindTransceiverResp() PDU {
	header := defaultHeader()
	header.CommandId = "bind_transceiver_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewBindReceiverResp() PDU {
	header := defaultHeader()
	header.CommandId = "bind_receiver_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewSubmitSM() PDU {
	header := defaultHeader()
	header.CommandId = "submit_sm"
	body := defaultSubmitSmBody()
	return PDU{Header: header, Body: body}
}

func NewSubmitSMResp() PDU {
	header := defaultHeader()
	header.CommandId = "submit_sm_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewDeliverSMResp() PDU {
	header := defaultHeader()
	header.CommandId = "deliver_sm_resp"
	body := Body{
		MandatoryParameter: map[string]interface{}{},
	}
	return PDU{Header: header, Body: body}
}

func NewDeliverSM() PDU {
	header := defaultHeader()
	header.CommandId = "deliver_sm"
	body := defaultSubmitSmBody()
	return PDU{Header: header, Body: body}
}

func NewDataSM() PDU {
	header := defaultHeader()
	header.CommandId = "data_sm"
	body := defaultSubmitSmBody()
	return PDU{Header: header, Body: body}
}

func defaultBindBody() Body {
	body := Body{
		MandatoryParameter: map[string]interface{}{
			"system_id":         "",
			"password":          "",
			"system_type":       "",
			"interface_version": 52,
			"addr_ton":          0,
			"addr_npi":          0,
			"address_range":     "",
		},
		OptionalParameters: nil,
	}
	return body
}

func defaultSubmitSmBody() Body {
	body := Body{
		MandatoryParameter: map[string]interface{}{
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
		OptionalParameters: nil,
	}
	return body

}

/* Builder Pattern associated functions */
func (p PDU) WithSystemId(s string) PDU {
	p.Body.MandatoryParameter["system_id"] = s
	return p
}

func (p PDU) WithPassword(s string) PDU {
	p.Body.MandatoryParameter["password"] = s
	return p
}

func (p PDU) WithAddressRange(s string) PDU {
	p.Body.MandatoryParameter["address_range"] = s
	return p
}

func (p PDU) WithSystemType(s string) PDU {
	p.Body.MandatoryParameter["system_type"] = s
	return p
}

func (p PDU) WithInterfaceVersion(i int) PDU {
	p.Body.MandatoryParameter["interface_version"] = i
	return p
}

func (p PDU) WithAddressNpi(i int) PDU {
	p.Body.MandatoryParameter["addr_npi"] = i
	return p
}

func (p PDU) WithAddressTon(i int) PDU {
	p.Body.MandatoryParameter["addr_ton"] = i
	return p
}

func (p PDU) WithSourceAddressNpi(i int) PDU {
	p.Body.MandatoryParameter["source_addr_npi"] = i
	return p
}

func (p PDU) WithSourceAddressTon(i int) PDU {
	p.Body.MandatoryParameter["source_addr_ton"] = i
	return p
}

func (p PDU) WithSourceAddress(s string) PDU {
	p.Body.MandatoryParameter["source_addr"] = s
	return p
}

func (p PDU) WithDestinationAddressNpi(i int) PDU {
	p.Body.MandatoryParameter["dest_addr_npi"] = i
	return p
}

func (p PDU) WithDestinationAddressTon(i int) PDU {
	p.Body.MandatoryParameter["dest_addr_ton"] = i
	return p
}

func (p PDU) WithDestinationAddress(s string) PDU {
	p.Body.MandatoryParameter["destination_addr"] = s
	return p
}

func (p PDU) WithDataCoding(i int) PDU {
	p.Body.MandatoryParameter["data_coding"] = i
	return p
}

func (p PDU) WithMessage(s string) PDU {
	p.Body.MandatoryParameter["short_message"] = s
	return p
}

func (p PDU) WithMessageId(id string) PDU {
	p.Body.MandatoryParameter["message_id"] = id
	return p
}

func (p PDU) WithSMPPError(id string) PDU {
	p.Header.CommandStatus = id
	return p
}

func (p PDU) WithCommandId(id string) PDU {
	p.Header.CommandId = id
	return p
}

func (p PDU) WithSequenceNumber(id int) PDU {
	p.Header.SequenceNumber = id
	return p
}

func (p PDU) isSystemId(id string) bool {
	return p.Body.MandatoryParameter["system_id"] == id
}

func (p PDU) isPassword(password string) bool {
	return p.Body.MandatoryParameter["password"] == password
}

func IsBindOperation(receivedPdu PDU) bool {
	switch receivedPdu.Header.CommandId {
	case "bind_transmitter",
		"bind_receiver",
		"bind_transceiver":
		return true
	}
	return false
}
