package smpp

import (
	"encoding/hex"
	"errors"
	"testing"
)

func Test(t *testing.T) {
	command_id := "00000001"
	expected_command := "bind_receiver"
	if got := commandIdByHex[command_id]["name"]; got != expected_command {
		t.Errorf("expected %v, to be %v", command_id, expected_command)
	}
}

var invalidPduLength, _ = hex.DecodeString("0000001000")
var invalidCommandId, _ = hex.DecodeString("00000010000011150000000000000000")
var enquiryLinkFixture, _ = hex.DecodeString("00000010000000150000000000000000")
var enquiryLinkRespFixture, _ = hex.DecodeString("00000010800000150000000000000000")
var bindTransmitterFixture, _ = hex.DecodeString("0000001f000000020000000000000000746573740074657374000034000000")

func TestEnquiryLinkParsing(t *testing.T) {
	enquiryLinkObj := PDU{
		header: Header{
			16,
			"enquire_link",
			"ESME_ROK",
			0,
		},
	}

	if got, _ := ParsePdu(enquiryLinkFixture); got != enquiryLinkObj {
		t.Errorf("didn't get enquire_link object %v, to be %v", got, enquiryLinkObj)
	}
}

func TestParsePduEnquiryLinkResp(t *testing.T) {
	enquiryLinkRespObj := PDU{
		header: Header{
			16,
			"enquire_link_resp",
			"ESME_ROK",
			0,
		},
	}

	if got, _ := ParsePdu(enquiryLinkRespFixture); got != enquiryLinkRespObj {
		t.Errorf("didn't get enquire_link_resp object %v, to be %v", got, enquiryLinkRespObj)
	}
}

func TestBindTransmitterParsing(t *testing.T) {
	bindTransmitterObj := PDU{
		header: Header{
			commandLength:  31,
			commandId:      "bind_transmitter",
			commandStatus:  "ESME_ROK",
			sequenceNumber: 0,
		},
	}
	if got, _ := ParsePdu(bindTransmitterFixture); got != bindTransmitterObj {
		t.Errorf("didn't get bind_transmitter object %v, to be %v", got, bindTransmitterFixture)
	}
}

func TestInvalidPduThrowsRelevantErrors(t *testing.T) {
	err := errors.New("invalid PDU Length for pdu : 0000001000")
	if _, got := ParsePdu(invalidPduLength); got.Error() != err.Error() {
		t.Errorf("didn't get expected error object : %v, to be %v", got, err)
	}
}
func TestInvalidCommandIdThrowsRelevantErrors(t *testing.T) {
	err := errors.New("unknown command_id 00001115")
	if _, got := ParsePdu(invalidCommandId); got.Error() != err.Error() {
		t.Errorf("didn't get expected error object : %v, to be %v", got, err)
	}
}
