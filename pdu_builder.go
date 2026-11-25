package smpp

// newPdu is a PDU of command commandId with status ESME_ROK and sequence_number 0.
func newPdu(commandId string, body Body) PDU {
	return PDU{Header: Header{CommandId: commandId, CommandStatus: ESME_ROK}, Body: body}
}

// emptyBody is the body of a PDU whose fields are set later, if any (most responses).
func emptyBody() Body {
	return Body{MandatoryParameter: map[string]interface{}{}}
}

/* Sane Defaults objects */
func NewGenericNack() PDU         { return newPdu("generic_nack", Body{}) }
func NewEnquireLink() PDU         { return newPdu("enquire_link", Body{}) }
func NewEnquireLinkResp() PDU     { return newPdu("enquire_link_resp", emptyBody()) }
func NewBindTransmitter() PDU     { return newPdu("bind_transmitter", defaultBindBody()) }
func NewBindReceiver() PDU        { return newPdu("bind_receiver", defaultBindBody()) }
func NewBindTransceiver() PDU     { return newPdu("bind_transceiver", defaultBindBody()) }
func NewBindTransmitterResp() PDU { return newPdu("bind_transmitter_resp", emptyBody()) }
func NewBindTransceiverResp() PDU { return newPdu("bind_transceiver_resp", emptyBody()) }
func NewBindReceiverResp() PDU    { return newPdu("bind_receiver_resp", emptyBody()) }
func NewSubmitSM() PDU            { return newPdu("submit_sm", defaultMessageBody()) }
func NewSubmitSMResp() PDU        { return newPdu("submit_sm_resp", emptyBody()) }
func NewDeliverSMResp() PDU       { return newPdu("deliver_sm_resp", emptyBody()) }
func NewDeliverSM() PDU           { return newPdu("deliver_sm", defaultMessageBody()) }
func NewDataSM() PDU              { return newPdu("data_sm", defaultMessageBody()) }

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

// defaultMessageBody is the default body of the message delivery PDUs: submit_sm, deliver_sm and data_sm.
func defaultMessageBody() Body {
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
func (p PDU) WithDefaults(defaults map[string]interface{}) PDU {
	for k, v := range defaults {
		p.Body.MandatoryParameter[k] = v
	}
	return p
}

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

// WithMessage sets short_message as text. The encoder converts it with the PDU's data_coding
// and sets sm_length to the encoded size.
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
