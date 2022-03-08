package smpp

import "fmt"

func handleEnquiryLinkPduReceived(e *ESME, receivedPdu PDU) (formated_error error) {
	ResponsePdu := NewEnquiryLinkResp().WithSequenceNumber(receivedPdu.header.sequenceNumber)
	_, formated_error = e.send(&ResponsePdu)
	return formated_error
}

func handleSubmitSmPduReceived(e *ESME, receivedPdu PDU) (formated_error error) {
	if e.isTransmitterState() {
		formated_error = replyToSubmitSM(*e, receivedPdu)
	} else {
		ResponsePdu := NewSubmitSMResp().WithSequenceNumber(receivedPdu.header.sequenceNumber)
		ResponsePdu = ResponsePdu.WithMessageId("").WithSMPPError(ESME_RINVBNDSTS)
		_, formated_error = e.send(&ResponsePdu)
	}
	return formated_error
}

func replyToSubmitSM(e ESME, receivedPdu PDU) (err error) {
	submit_sm_resp_bytes := NewSubmitSMResp().WithMessageId("1").WithSequenceNumber(1)
	_, err = e.send(&submit_sm_resp_bytes)
	return err
}

func (s *SMSC) handleBindOperation(e *ESME, receivedPdu PDU) error {
	ResponsePdu := receivedPdu.WithCommandId(receivedPdu.header.commandId + "_resp")
	if !receivedPdu.isSystemId(s.SystemId) || !receivedPdu.isPassword(s.Password) {
		ResponsePdu.header.commandStatus = ESME_RBINDFAIL
		InfoSmppLogger.Printf("We didn't received expected credentials")
	}
	bindResponse, err := EncodePdu(ResponsePdu)
	if err != nil {
		return fmt.Errorf("Encoding bind response failed : %v", err)
	}
	err = SetESMEStateFromSMSCResponse(&ResponsePdu, e)
	if err != nil {
		InfoSmppLogger.Printf("Couldn't set the bind state on request!")
	}
	_, err = (e.clientSocket).Write(bindResponse)
	if err != nil {
		return fmt.Errorf("Couldn't write to the ESME from SMSC : %v", err)
	}
	return nil
}
