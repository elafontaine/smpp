package smpp

import (
	"encoding/hex"
	"errors"
	"testing"
)

var invalidPduLength, _ = hex.DecodeString("0000001000")
var pduLengthMissing, _ = hex.DecodeString("000000")
var invalidCommandId, _ = hex.DecodeString("00000010000011150000000000000000")
var enquiryLinkFixture, _ = hex.DecodeString("00000010000000150000000000000000")
var enquiryLinkRespFixture, _ = hex.DecodeString("00000010800000150000000000000000")
var bindTransmitterFixture, _ = hex.DecodeString("0000001f000000020000000000000000746573740074657374000034000000")

func TestEnquiryLinkParsing(t *testing.T) {
	enquiryLinkObjHeader := PDU{
		header: Header{16, "enquire_link", "ESME_ROK", 0},
	}

	if got, _ := parseHeader(enquiryLinkFixture); got != enquiryLinkObjHeader {
		t.Errorf("didn't get enquire_link object %v, to be %v", got, enquiryLinkObjHeader)
	}
}

func TestParsePduEnquiryLinkResp(t *testing.T) {
	enquiryLinkRespObjHeader := PDU{
		header: Header{16, "enquire_link_resp", "ESME_ROK", 0},
	}

	if got, _ := parseHeader(enquiryLinkRespFixture); got != enquiryLinkRespObjHeader {
		t.Errorf("didn't get enquire_link_resp object %v, to be %v", got, enquiryLinkRespObjHeader)
	}
}

func TestBindTransmitterParsing(t *testing.T) {
	bindTransmitterObjHeader := PDU{
		header: Header{commandLength: 31, commandId: "bind_transmitter", commandStatus: "ESME_ROK", sequenceNumber: 0},
	}
	if got, _ := parseHeader(bindTransmitterFixture); got != bindTransmitterObjHeader {
		t.Errorf("didn't get bind_transmitter object %v, to be %v", got, bindTransmitterFixture)
	}
}

func Test_parseHeader(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{"PduInvalidLength", args{bytes: pduLengthMissing}, errors.New("invalid length parameter")},
		{"InvalidCommandId", args{bytes: invalidCommandId}, errors.New("unknown command_id 00001115")},
		{"InvalidPdu", args{bytes: invalidPduLength}, errors.New("invalid PDU Length for pdu : 0000001000")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseHeader(tt.args.bytes)
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("parseHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
