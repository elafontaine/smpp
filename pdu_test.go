package smpp

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

var invalidPduLength, _ = hex.DecodeString("0000001000")
var pduLengthMissing, _ = hex.DecodeString("000000")
var invalidCommandId, _ = hex.DecodeString("00000010000011150000000000000000")
var invalidStatusId, _ = hex.DecodeString("0000001000000015008000ff00000000")
var enquiryLinkFixture, _ = hex.DecodeString("00000010000000150000000000000000")
var enquiryLinkRespFixture, _ = hex.DecodeString("00000010800000150000000000000000")
var bindTransmitterFixture, _ = hex.DecodeString("0000001f000000020000000000000000746573740074657374000034000000")
var bindTransmitterRespFixture, _ = hex.DecodeString("000000158000000200000000000000007465737400")
var submitSmRespFixture, _ = hex.DecodeString("000000128000000400000000000000033100")
var deliverSmOptionsFixture, _ = hex.DecodeString("0000003f000000050000000000000001000000000000353535353535313233340004000000000000000000001e000631313130370004270001020425000100")

var optionalParameterReceiptMessageIdBytes, _ = hex.DecodeString("001e0006313131303700")
var optionalParameterMessageStateBytes, _ = hex.DecodeString("0427000102")
var optionalParameterDeliveryFailureReasonBytes, _ = hex.DecodeString("0425000100")

var enquiryLinkObjHeader = Header{CommandLength: 16, CommandId: "enquire_link", CommandStatus: ESME_ROK, SequenceNumber: 0}
var enquiryLinkRespObjHeader = Header{CommandLength: 16, CommandId: "enquire_link_resp", CommandStatus: ESME_ROK, SequenceNumber: 0}
var bindTransmitterObjHeader = Header{CommandLength: 31, CommandId: "bind_transmitter", CommandStatus: ESME_ROK, SequenceNumber: 0}
var bindTransmitterRespObjHeader = Header{CommandLength: 21, CommandId: "bind_transmitter_resp", CommandStatus: ESME_ROK, SequenceNumber: 0}
var submitSmRespObjHeader = Header{CommandLength: 18, CommandId: "submit_sm_resp", CommandStatus: ESME_ROK, SequenceNumber: 3}
var deliverSmRespObjHeader = Header{CommandLength: 18, CommandId: "deliver_sm_resp", CommandStatus: ESME_ROK, SequenceNumber: 3}
var deliverSmObjHeader = Header{CommandLength: 63, CommandId: "deliver_sm", CommandStatus: ESME_ROK, SequenceNumber: 1}
var dataSmObjHeader = Header{CommandLength: 63, CommandId: "data_sm", CommandStatus: ESME_ROK, SequenceNumber: 1}
var submitSmObjHeader = Header{CommandLength: 63, CommandId: "submit_sm", CommandStatus: ESME_ROK, SequenceNumber: 0}
var generickNackHeader = Header{CommandLength: 63, CommandId: "generick_nack", CommandStatus: "ESME_RINVSRCADR", SequenceNumber: 0}

var bindTransmitterObjBody = Body{
	MandatoryParameter: map[string]interface{}{
		"system_id":         "test",
		"password":          "test",
		"system_type":       "",
		"interface_version": 52,
		"addr_ton":          0,
		"addr_npi":          0,
		"address_range":     "",
	},
}
var bindTransmitterRespObjBody = Body{MandatoryParameter: map[string]interface{}{"system_id": "test"}}
var submitSmRespObjBody = Body{MandatoryParameter: map[string]interface{}{"message_id": "1"}}
var optionalReceiptMessageID = map[string]interface{}{"tag": "receipted_message_id", "length": 6, "value": "11107"}
var optionalMessageState = map[string]interface{}{"tag": "message_state", "length": 1, "value": 2}
var optionalDeliveryFailureReason = map[string]interface{}{"tag": "delivery_failure_reason", "length": 1, "value": 0}

var deliverSmObjBody = Body{
	MandatoryParameter: map[string]interface{}{
		"service_type":            "",
		"source_addr_ton":         0,
		"source_addr_npi":         0,
		"source_addr":             "",
		"dest_addr_ton":           0,
		"dest_addr_npi":           0,
		"destination_addr":        "5555551234",
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
	OptionalParameters: []map[string]interface{}{
		optionalReceiptMessageID,
		optionalMessageState,
		optionalDeliveryFailureReason,
	},
}
var submitSmRespObj = PDU{Header: submitSmRespObjHeader, Body: submitSmRespObjBody}
var deliverSmRespObj = PDU{Header: deliverSmRespObjHeader, Body: submitSmRespObjBody}
var deliverSmObj = PDU{Header: deliverSmObjHeader, Body: deliverSmObjBody}
var dataSmObj = PDU{Header: dataSmObjHeader, Body: deliverSmObjBody}
var submitSmObj = PDU{Header: submitSmObjHeader, Body: deliverSmObjBody}
var bindTransmitterObj = PDU{Header: bindTransmitterObjHeader, Body: bindTransmitterObjBody}
var bindTransmitterRespObj = PDU{Header: bindTransmitterRespObjHeader, Body: bindTransmitterRespObjBody}
var enquiryLinkObj = PDU{Header: enquiryLinkObjHeader, Body: Body{MandatoryParameter: map[string]interface{}{}}}
var enquiryLinkRespObj = PDU{Header: enquiryLinkRespObjHeader, Body: Body{MandatoryParameter: map[string]interface{}{}}}

var missingBodySubmitSMPdu = PDU{
	Header: Header{
		SequenceNumber: 1,
		CommandId:      "submit_sm",
		CommandStatus:  ESME_ROK,
		CommandLength:  0,
	},
	Body: Body{},
}
var missingBodyDeliverSMPdu = PDU{
	Header: Header{
		SequenceNumber: 1,
		CommandId:      "deliver_sm",
		CommandStatus:  ESME_ROK,
		CommandLength:  0,
	},
	Body: Body{},
}
var missingBodySubmitSMPduButWithServiceType = PDU{
	Header: Header{
		SequenceNumber: 1,
		CommandId:      "submit_sm",
		CommandStatus:  ESME_ROK,
		CommandLength:  0,
	},
	Body: Body{MandatoryParameter: map[string]interface{}{
		"service_type": "",
	},
	},
}
var missingHeaderPdu = PDU{
	Header: Header{},
	Body:   Body{},
}

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
	t.Parallel()
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
		{"parse_deliver_sm_with_options", args{bytes: deliverSmOptionsFixture, header: deliverSmObjHeader}, deliverSmObjBody},
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
	t.Parallel()
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
		{"parse_enquire_link", args{bytes: enquiryLinkFixture}, enquiryLinkObj},
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
	t.Parallel()
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
		{"Unknown status ID received", args{bytes: invalidStatusId}, errors.New("unknown command status 008000ff")},
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

func TestEncodePdu(t *testing.T) {
	t.Parallel()
	type args struct {
		pdu_obj PDU
	}
	tests := []struct {
		name    string
		args    args
		wantPdu []byte
	}{
		{"encode enquiry object into bytes", args{pdu_obj: enquiryLinkObj}, enquiryLinkFixture},
		{"encode enquiry resp object into bytes", args{pdu_obj: enquiryLinkRespObj}, enquiryLinkRespFixture},
		{"parse_submit_sm_resp", args{pdu_obj: submitSmRespObj}, submitSmRespFixture},
		{"bind_transmitter object into bytes", args{pdu_obj: bindTransmitterObj}, bindTransmitterFixture},
		{"bind_transmitter_resp object into bytes", args{pdu_obj: bindTransmitterRespObj}, bindTransmitterRespFixture},
		{"deliver_sm object with optional parameter into bytes", args{pdu_obj: deliverSmObj}, deliverSmOptionsFixture},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := EncodePdu(tt.args.pdu_obj)
			eq := reflect.DeepEqual(got, tt.wantPdu)
			if !eq {
				t.Errorf("EncodePdu() got = %v, wantPdu %v", got, tt.wantPdu)
				return
			}
		})
	}
}

func Test_encodeSpecificOptionalParameter(t *testing.T) {
	t.Parallel()
	type args struct {
		optionalParam map[string]interface{}
	}
	tests := []struct {
		name                    string
		args                    args
		wantOptionalParamsBytes []byte
		wantErr                 bool
	}{
		{"Encode message state optional parameter", args{optionalParam: optionalMessageState}, optionalParameterMessageStateBytes, false},
		{"Encode delivery failure reason optional parameter", args{optionalParam: optionalDeliveryFailureReason}, optionalParameterDeliveryFailureReasonBytes, false},
		{"Encode receipt message ID optional parameter", args{optionalParam: optionalReceiptMessageID}, optionalParameterReceiptMessageIdBytes, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOptionalParamsBytes, err := encodeSpecificOptionalParameter(tt.args.optionalParam)
			if (err != nil) != tt.wantErr {
				t.Errorf("encodeSpecificOptionalParameter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotOptionalParamsBytes, tt.wantOptionalParamsBytes) {
				t.Errorf("encodeSpecificOptionalParameter() gotOptionalParamsBytes = %v, want %v", gotOptionalParamsBytes, tt.wantOptionalParamsBytes)
			}
		})
	}
}

func Test_extractOptionalParameters(t *testing.T) {
	t.Parallel()
	type args struct {
		optionalParameterBytes []byte
	}
	tests := []struct {
		name string
		args args
		want map[string]interface{}
	}{
		{"Decode message state", args{optionalParameterBytes: optionalParameterMessageStateBytes}, optionalMessageState},
		{"Decode ReceiptMessageId", args{optionalParameterBytes: optionalParameterReceiptMessageIdBytes}, optionalReceiptMessageID},
		{"Decode DeliveryFailureReason", args{optionalParameterBytes: optionalParameterDeliveryFailureReasonBytes}, optionalDeliveryFailureReason},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, _ := extractSpecificOptionalParameter(tt.args.optionalParameterBytes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractOptionalParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInvalidPduEncodingCases(t *testing.T) {
	t.Parallel()
	type args struct {
		pdu_obj PDU
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{"missingBodySubmitSMPdu raise mandatory fields are missings", args{missingBodySubmitSMPdu}, errors.New("service_type of submit_sm pdu missing, can't encode")},
		{"missingBodySubmitSMPduButWithServiceType raise mandatory fields are missings", args{missingBodySubmitSMPduButWithServiceType}, errors.New("source_addr_ton of submit_sm pdu missing, can't encode")},
		{"missingHeader raise header missing error", args{missingHeaderPdu}, missingHeaderError},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodePdu(tt.args.pdu_obj)
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("EncodePdu() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestInvalidPduEncodingCasesBody(t *testing.T) {
	t.Parallel()
	type args struct {
		pdu_obj PDU
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
	}{
		{"missingBodySubmitSMPdu raise mandatory fields are missings", args{missingBodySubmitSMPdu}, errors.New("service_type of submit_sm pdu missing, can't encode")},
		{"missingBodyDeliverSMPdu raise mandatory fields are missings", args{missingBodyDeliverSMPdu}, errors.New("service_type of deliver_sm pdu missing, can't encode")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := encodeBody(tt.args.pdu_obj)
			if err.Error() != tt.wantErr.Error() {
				t.Errorf("encodeBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func Test_PduWriter(t *testing.T) {
	t.Parallel()
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantPdu PDU
		err     error
	}{
		{
			name:    "Write to the right size buffer",
			args:    args{bytes: bytes.Clone(bindTransmitterFixture)},
			wantPdu: bindTransmitterObj,
		},
		{
			name:    "Wrong content should be returning an error",
			args:    args{bytes: bytes.Clone(bindTransmitterFixture[:len(bindTransmitterFixture)-20])},
			wantPdu: PDU{},
			err:     fmt.Errorf("invalid PDU Length for pdu : %v", hex.EncodeToString(bindTransmitterFixture[:len(bindTransmitterFixture)-20])),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualPdu := PDU{}
			_, err := actualPdu.Write(tt.args.bytes)
			eq := reflect.DeepEqual(actualPdu, tt.wantPdu)
			if !eq {
				t.Errorf("Write not working: got = %v, wantPdu = %v", actualPdu, tt.wantPdu)
			}
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("Error wasn't the one expected: got %v, wanted err %v", err, tt.err)
			}
		})
	}
}

func TestPduReader(t *testing.T) {
	t.Parallel()
	type args struct {
		Pdu PDU
	}
	tests := []struct {
		name      string
		args      args
		wantBytes []byte
		err       error
	}{
		{
			name:      "Read to the right size buffer",
			args:      args{Pdu: bindTransmitterObj},
			wantBytes: bindTransmitterFixture,
		},
		{
			name:      "Wrong content should be returning an error",
			args:      args{Pdu: missingHeaderPdu},
			wantBytes: []byte{},
			err:       missingHeaderError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualBytes := make([]byte, len(tt.wantBytes))
			_, err := tt.args.Pdu.Read(actualBytes)
			eq := reflect.DeepEqual(actualBytes, tt.wantBytes)
			if !eq {
				t.Errorf("Read not working: got = %v, wantPdu = %v", actualBytes, tt.wantBytes)
			}
			if err != nil && err.Error() != tt.err.Error() {
				t.Errorf("Error wasn't the one expected: got %v, wanted err %v", err, tt.err)
			}
		})
	}
}
