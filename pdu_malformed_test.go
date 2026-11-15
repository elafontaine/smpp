package smpp

import (
	"reflect"
	"testing"
)

// submitSmBodyNoMessage is a complete submit_sm body with an empty short_message (17 bytes).
const submitSmBodyNoMessage = "00" + "0000" + "00" + "0000" + "00" + "000000" + "00" + "00" + "00000000" + "00"

// Malformed input must give an error, never a panic. The header still comes back when it was
// readable, so the receiver can answer with the right sequence number (e.g. generic_nack).
func TestParsePduMalformedBody(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		pduHex  string
		wantErr string
		wantSeq int
	}{
		{"string field without its NUL terminator",
			"00000011000000020000000000000001" + "78",
			"system_id of bind_transmitter pdu: missing NUL terminator", 1},
		{"body ends before an integer field",
			"00000011000000040000000000000002" + "00",
			"source_addr_ton of submit_sm pdu: missing, the body ends early", 2},
		{"sm_length larger than what's left",
			"00000023000000040000000000000003" + submitSmBodyNoMessage[:len(submitSmBodyNoMessage)-2] + "05" + "6865",
			"short_message of submit_sm pdu: sm_length is 5 but only 2 bytes are left", 3},
		{"leftover bytes too short for a TLV",
			"00000023000000040000000000000004" + submitSmBodyNoMessage + "001e",
			"optional parameters: 2 bytes left, a TLV needs at least 4", 4},
		{"TLV length larger than what's left",
			"00000028000000040000000000000005" + submitSmBodyNoMessage + "001e0006313100",
			"optional parameter receipted_message_id: length 6 but only 3 bytes are left", 5},
		{"integer TLV of an unsupported size",
			"00000028000000040000000000000006" + submitSmBodyNoMessage + "020c0003010203",
			"optional parameter sar_msg_ref_num: integer of 3 bytes isn't supported", 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePdu(mustHex(tt.pduHex))
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("ParsePdu() error = %v, want %q", err, tt.wantErr)
			}
			if got.Header.SequenceNumber != tt.wantSeq {
				t.Errorf("header sequence_number = %d, want %d (the header must still come back)", got.Header.SequenceNumber, tt.wantSeq)
			}
		})
	}
}

// Inputs that look odd but are valid.
func TestParsePduUnusualButValid(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		pduHex string
		want   Body
	}{
		// SMPP 3.4: a response with a non-zero command_status may have no body at all.
		{"error submit_sm_resp without a body",
			"00000010800000040000000400000007",
			Body{MandatoryParameter: map[string]interface{}{}}},
		{"string TLV of length 0",
			"00000025000000040000000000000008" + submitSmBodyNoMessage + "001e0000",
			Body{
				MandatoryParameter: messageBodyWith(nil).MandatoryParameter,
				OptionalParameters: []map[string]interface{}{{"tag": "receipted_message_id", "length": 0, "value": ""}},
			}},
		{"vendor TLV unknown to the table: kept as raw bytes",
			"00000026000000040000000000000009" + submitSmBodyNoMessage + "14050001ff",
			Body{
				MandatoryParameter: messageBodyWith(nil).MandatoryParameter,
				OptionalParameters: []map[string]interface{}{{"tag": "0x1405", "length": 1, "value": "\xff"}},
			}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParsePdu(mustHex(tt.pduHex))
			if err != nil {
				t.Fatalf("ParsePdu() error = %v", err)
			}
			if !reflect.DeepEqual(got.Body, tt.want) {
				t.Errorf("ParsePdu() body = %v, want %v", got.Body, tt.want)
			}
		})
	}
}

// Whatever the bytes, ParsePdu must not panic, and neither may EncodePdu on what ParsePdu
// accepted. Errors are fine. Run longer with: go test -fuzz=FuzzParsePdu -fuzztime=1m
func FuzzParsePdu(f *testing.F) {
	for _, seed := range [][]byte{
		enquiryLinkFixture, bindTransmitterFixture, bindTransmitterRespFixture, submitSmRespFixture,
		deliverSmOptionsFixture, submitSmHelloFixture, submitSmNihaoUcs2Fixture,
		mustHex("00000010800000040000000400000007"),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, b []byte) {
		pdu, err := ParsePdu(b)
		if err == nil {
			_, _ = EncodePdu(pdu)
		}
	})
}
