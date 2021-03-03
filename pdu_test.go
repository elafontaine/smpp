package smpp

import (
	"encoding/hex"
	"errors"
	"reflect"
	"testing"
)

var invalidPduLength, _ = hex.DecodeString("0000001000")
var pduLengthMissing, _ = hex.DecodeString("000000")
var invalidCommandId, _ = hex.DecodeString("00000010000011150000000000000000")
var enquiryLinkFixture, _ = hex.DecodeString("00000010000000150000000000000000")
var enquiryLinkRespFixture, _ = hex.DecodeString("00000010800000150000000000000000")
var bindTransmitterFixture, _ = hex.DecodeString("0000001f000000020000000000000000746573740074657374000034000000")
var bindTransmitterRespFixture, _ = hex.DecodeString("000000158000000200000000000000007465737400")
var submitSmRespFixture, _ = hex.DecodeString("000000128000000400000000000000033100")
var deliverSmOptionsFixture, _ = hex.DecodeString("0000003f000000050000000000000001000000000000353134393432363635300004000000000000000000001e000631313130370004270001020425000100")

var enquiryLinkObjHeader = Header{commandLength: 16, commandId: "enquire_link", commandStatus: "ESME_ROK", sequenceNumber: 0}
var enquiryLinkRespObjHeader = Header{commandLength: 16, commandId: "enquire_link_resp", commandStatus: "ESME_ROK", sequenceNumber: 0}
var bindTransmitterObjHeader = Header{commandLength: 31, commandId: "bind_transmitter", commandStatus: "ESME_ROK", sequenceNumber: 0}
var bindTransmitterRespObjHeader = Header{commandLength: 21, commandId: "bind_transmitter_resp", commandStatus: "ESME_ROK", sequenceNumber: 0}
var submitSmRespObjHeader = Header{commandLength: 18, commandId: "submit_sm_resp", commandStatus: "ESME_ROK", sequenceNumber: 0}
var deliverSmRespObjHeader = Header{commandLength: 63, commandId: "deliver_sm", commandStatus: "ESME_ROK", sequenceNumber: 1}

var bindTransmitterObjBody = Body{
	mandatoryParameter: map[string]interface{}{
		"system_id":         "test",
		"password":          "test",
		"system_type":       "",
		"interface_version": 52,
		"addr_ton":          0,
		"addr_npi":          0,
		"address_range":     "",
	},
}
var bindTransmitterRespObjBody = Body{mandatoryParameter: map[string]interface{}{"system_id": "test"}}
var submitSmRespObjBody = Body{mandatoryParameter: map[string]interface{}{"message_id": "1"}}
var deliverSmObjBody = Body{
	mandatoryParameter: map[string]interface{}{
		"service_type":            "",
		"source_addr_ton":         0,
		"source_addr_npi":         0,
		"source_addr":             "",
		"dest_addr_ton":           0,
		"dest_addr_npi":           0,
		"destination_addr":        "5149426650",
		"esm_class":               4,
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
	optionalParameters: []map[string]interface{}{
		{"tag": "receipted_message_id", "length": 6, "value": "11107"},
		{"tag": "message_state", "length": 1, "value": 2},
		{"tag": "delivery_failure_reason", "length": 1, "value": 0},
	},
}

var submitSmRespObj = PDU{header: submitSmRespObjHeader, body: submitSmRespObjBody}
var bindTransmitterObj = PDU{header: bindTransmitterObjHeader, body: bindTransmitterObjBody}

func Test_parseHeaders(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantPdu Header
	}{
		{"parse_enquire_link_header", args{bytes: enquiryLinkFixture}, enquiryLinkObjHeader},
		{"parse_enquire_link_resp_header", args{bytes: enquiryLinkRespFixture}, enquiryLinkRespObjHeader},
		{"parse_bind_transmitter_header", args{bytes: bindTransmitterFixture}, bindTransmitterObjHeader},
		{"parse_bind_transmitter_resp_header", args{bytes: bindTransmitterRespFixture}, bindTransmitterRespObjHeader},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := parseHeader(tt.args.bytes)
			if got != tt.wantPdu {
				t.Errorf("parseHeader() got = %v, wantPdu %v", got, tt.wantPdu)
				return
			}
		})
	}
}

func Test_parsePduBody(t *testing.T) {
	type args struct {
		bytes  []byte
		header Header
	}
	tests := []struct {
		name     string
		args     args
		wantBody Body
	}{
		{"parse_bind_transmitter", args{header: bindTransmitterObjHeader, bytes: bindTransmitterFixture}, bindTransmitterObjBody},
		{"parse_bind_transmitter_resp", args{header: bindTransmitterRespObjHeader, bytes: bindTransmitterRespFixture}, bindTransmitterRespObjBody},
		{"parse_submit_sm_resp", args{bytes: submitSmRespFixture, header: submitSmRespObjHeader}, submitSmRespObjBody},
		{"parse_deliver_sm_with_options", args{bytes: deliverSmOptionsFixture, header: deliverSmRespObjHeader}, deliverSmObjBody},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := parseBody(tt.args.header, tt.args.bytes)
			eq := reflect.DeepEqual(got, tt.wantBody)
			if !eq {
				t.Errorf("parseBody() got = %v, want Body %v", got, tt.wantBody)
				return
			}
		})
	}
}

func Test_parsePdu(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantPdu PDU
	}{
		{"parse_bind_transmitter", args{bytes: bindTransmitterFixture}, bindTransmitterObj},
		{"parse_submit_sm_resp", args{bytes: submitSmRespFixture}, submitSmRespObj},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ParsePdu(tt.args.bytes)
			eq := reflect.DeepEqual(got, tt.wantPdu)
			if !eq {
				t.Errorf("parsePdu() got = %v, wantPdu %v", got, tt.wantPdu)
				return
			}
		})
	}
}

func Test_parseHeaderInvalidPdu(t *testing.T) {
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
