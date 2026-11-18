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
var submitSmHelloFixture, _ = hex.DecodeString(
	"00000030000000040000000000000001" + // header: length 48, submit_sm, ESME_ROK, sequence 1
		"00" + "0000" + "00" + // service_type "", source_addr_ton, source_addr_npi, source_addr ""
		"0000" + "3535353535353132333400" + // dest_addr_ton, dest_addr_npi, destination_addr "5555551234"
		"000000" + "00" + "00" + // esm_class, protocol_id, priority_flag, schedule_delivery_time "", validity_period ""
		"00000000" + "05" + "68656c6c6f") // registered_delivery, replace_if_present_flag, data_coding, sm_default_msg_id, sm_length 5, short_message "hello"
var submitSmNihaoUcs2Fixture, _ = hex.DecodeString(
	"0000002f000000040000000000000001" + // header: length 47, submit_sm, ESME_ROK, sequence 1
		"00" + "0000" + "00" + // service_type "", source_addr_ton, source_addr_npi, source_addr ""
		"0000" + "3535353535353132333400" + // dest_addr_ton, dest_addr_npi, destination_addr "5555551234"
		"000000" + "00" + "00" + // esm_class, protocol_id, priority_flag, schedule_delivery_time "", validity_period ""
		"00000800" + "04" + // registered_delivery, replace_if_present_flag, data_coding 8 (UCS-2), sm_default_msg_id, sm_length 4
		"4f60597d") // short_message "你好": 2 characters, 2 bytes each in UCS-2

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
var genericNackHeader = Header{CommandLength: 63, CommandId: "generic_nack", CommandStatus: "ESME_RINVSRCADR", SequenceNumber: 0}

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
var submitSmHelloObj = PDU{
	Header: Header{CommandLength: 48, CommandId: "submit_sm", CommandStatus: ESME_ROK, SequenceNumber: 1},
	Body:   messageBodyWith(map[string]interface{}{"destination_addr": "5555551234", "sm_length": 5, "short_message": "hello"}),
}

var submitSmNihaoUcs2Obj = PDU{
	Header: Header{CommandLength: 47, CommandId: "submit_sm", CommandStatus: ESME_ROK, SequenceNumber: 1},
	Body:   messageBodyWith(map[string]interface{}{"destination_addr": "5555551234", "data_coding": 8, "sm_length": 4, "short_message": "你好"}),
}

// messageBodyWith returns a fresh default message body (submit_sm, deliver_sm, data_sm) with some fields overridden.
func messageBodyWith(fields map[string]interface{}) Body {
	body := defaultMessageBody()
	for name, value := range fields {
		body.MandatoryParameter[name] = value
	}
	return body
}

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
		{"parse_submit_sm_with_short_message", args{bytes: submitSmHelloFixture}, submitSmHelloObj},
		{"parse_submit_sm_with_ucs2_short_message", args{bytes: submitSmNihaoUcs2Fixture}, submitSmNihaoUcs2Obj},
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
		{"command_length shorter than the header", args{bytes: mustHex("0000000800000015")}, errors.New("invalid command_length 8: shorter than the 16-byte header")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParsePdu(tt.args.bytes)
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
		{"submit_sm object with UCS-2 short_message into bytes", args{pdu_obj: submitSmNihaoUcs2Obj}, submitSmNihaoUcs2Fixture},
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
		{"Encode a 2-byte integer (sar_msg_ref_num)", args{optionalParam: map[string]interface{}{"tag": "sar_msg_ref_num", "value": 0x1234}}, mustHex("020c00021234"), false},
		{"Encode a 4-byte integer (qos_time_to_live)", args{optionalParam: map[string]interface{}{"tag": "qos_time_to_live", "value": 0x01020304}}, mustHex("0017000401020304"), false},
		{"Encode an octet string (message_payload)", args{optionalParam: map[string]interface{}{"tag": "message_payload", "value": "hi"}}, mustHex("042400026869"), false},
		{"Integer too large for its 1 byte", args{optionalParam: map[string]interface{}{"tag": "sar_total_segments", "value": 256}}, nil, true},
		{"Unknown optional parameter", args{optionalParam: map[string]interface{}{"tag": "no_such_tlv", "value": 1}}, nil, true},
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
		{"Decode a 2-byte integer (sar_msg_ref_num)", args{optionalParameterBytes: mustHex("020c00021234")}, map[string]interface{}{"tag": "sar_msg_ref_num", "length": 2, "value": 0x1234}},
		{"Decode a 4-byte integer (qos_time_to_live)", args{optionalParameterBytes: mustHex("0017000401020304")}, map[string]interface{}{"tag": "qos_time_to_live", "length": 4, "value": 0x01020304}},
		{"Decode an octet string (message_payload)", args{optionalParameterBytes: mustHex("042400026869")}, map[string]interface{}{"tag": "message_payload", "length": 2, "value": "hi"}},
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
		{"missingBodyDeliverSMPdu raise mandatory fields are missings", args{missingBodyDeliverSMPdu}, errors.New("service_type of deliver_sm pdu missing, can't encode")},
		{"integer too large for its field raise an error", args{NewSubmitSM().WithDataCoding(256)}, errors.New("data_coding of submit_sm pdu is 256, doesn't fit in 1 byte(s), can't encode")},
		{"text not representable in its data_coding raise an error", args{NewSubmitSM().WithDataCoding(DataCodingIA5).WithMessage("你好")}, errors.New("short_message of submit_sm pdu: character '你' can't be encoded with data_coding 1, can't encode")},
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

// SMPP integers are fixed-width big-endian: 128 and above must still take a single byte for sm_length.
func TestEncodeShortMessageOf128BytesOrMore(t *testing.T) {
	t.Parallel()
	for _, length := range []int{127, 128, 200, 254} {
		t.Run(fmt.Sprintf("short_message of %d bytes", length), func(t *testing.T) {
			message := string(bytes.Repeat([]byte("a"), length))
			pdu := NewSubmitSM().WithMessage(message).WithSequenceNumber(1)

			encoded, err := EncodePdu(pdu)
			if err != nil {
				t.Fatalf("EncodePdu() error = %v", err)
			}
			if want := 16 + 17 + length; len(encoded) != want {
				t.Fatalf("EncodePdu() gave %d bytes, want %d", len(encoded), want)
			}
			decoded, err := ParsePdu(encoded)
			if err != nil {
				t.Fatalf("ParsePdu() error = %v", err)
			}
			if got := decoded.Body.MandatoryParameter["sm_length"]; got != length {
				t.Errorf("decoded sm_length = %v, want %d", got, length)
			}
			if got := decoded.Body.MandatoryParameter["short_message"]; got != message {
				t.Errorf("decoded short_message has %d bytes, want %d", len(got.(string)), length)
			}
		})
	}
}

// sm_length follows the encoded bytes, not the characters, nor the sm_length in the map.
func TestEncodeComputesSmLengthFromEncodedText(t *testing.T) {
	t.Parallel()
	const smLengthOffset = 16 + 16 // header + the submit_sm fields before sm_length
	pdu := NewSubmitSM().WithDataCoding(DataCodingUCS2).WithMessage("你好")
	pdu.Body.MandatoryParameter["sm_length"] = 99

	encoded, err := EncodePdu(pdu)
	if err != nil {
		t.Fatalf("EncodePdu() error = %v", err)
	}
	if got := encoded[smLengthOffset]; got != 4 {
		t.Errorf("sm_length = %d, want 4 (2 UCS-2 characters)", got)
	}
}

// A []byte short_message is sent as given, with the map's sm_length, so a peer can be
// tested with an inconsistent PDU.
func TestEncodeRawShortMessageKeepsGivenSmLength(t *testing.T) {
	t.Parallel()
	const smLengthOffset = 16 + 16
	pdu := NewSubmitSM()
	pdu.Body.MandatoryParameter["short_message"] = []byte{0xde, 0xad}
	pdu.Body.MandatoryParameter["sm_length"] = 7

	encoded, err := EncodePdu(pdu)
	if err != nil {
		t.Fatalf("EncodePdu() error = %v", err)
	}
	if got := encoded[smLengthOffset]; got != 7 {
		t.Errorf("sm_length = %d, want 7 as given", got)
	}
	if got := encoded[smLengthOffset+1:]; !bytes.Equal(got, []byte{0xde, 0xad}) {
		t.Errorf("short_message bytes = %x, want dead", got)
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

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
