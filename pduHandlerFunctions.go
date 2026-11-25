package smpp

func handleEnquiryLinkPduReceived(e *ESME, receivedPdu PDU) error {
	return e.Reply(NewEnquireLinkResp().WithSequenceNumber(receivedPdu.Header.SequenceNumber))
}

func handleSubmitSmPduReceived(e *ESME, receivedPdu PDU) error {
	ResponsePdu := NewSubmitSMResp().WithSequenceNumber(receivedPdu.Header.SequenceNumber)
	if e.isTransmitterState() {
		ResponsePdu = ResponsePdu.WithMessageId("1")
	} else {
		ResponsePdu = ResponsePdu.WithMessageId("").WithSMPPError(ESME_RINVBNDSTS)
	}
	return e.Reply(ResponsePdu)
}

func (s *SMSC) handleBindOperation(e *ESME, receivedPdu PDU) error {
	ResponsePdu := receivedPdu.WithCommandId(receivedPdu.Header.CommandId + "_resp")
	if !receivedPdu.isSystemId(s.SystemId) || !receivedPdu.isPassword(s.Password) {
		ResponsePdu.Header.CommandStatus = ESME_RBINDFAIL
		InfoSmppLogger.Printf("We didn't received expected credentials")
	}
	if err := e.setStateFromBindResponse(ResponsePdu); err != nil {
		InfoSmppLogger.Printf("Couldn't set the bind state on request!")
	}
	return e.Reply(ResponsePdu)
}

func handleDeliverSmPduReceived(e *ESME, receivedPdu PDU) error {
	return e.Reply(receivedPdu.
		WithCommandId(receivedPdu.Header.CommandId + "_resp").
		WithSMPPError(ESME_RINVBNDSTS).
		WithMessageId(""))
}
