package smpp

import (
	"log"
	"strings"
	"testing"
	"time"
)

// Every request the ESME can't process gets a generic_nack with the request's sequence_number,
// and the connection stays up.
func TestUnprocessableRequestsAreAnsweredWithGenericNack(t *testing.T) {
	dataSm, _ := EncodePdu(NewDataSM().WithSequenceNumber(9))
	deliverSm, _ := EncodePdu(NewDeliverSM().WithSequenceNumber(11))
	tests := []struct {
		name       string
		pdu        []byte
		wantStatus string
		wantSeq    int
	}{
		{"known request with a malformed body", mustHex("00000011" + "00000002" + "00000000" + "00000007" + "78"), ESME_RINVCMDLEN, 7},
		{"known request with no handler", dataSm, ESME_RINVCMDID, 9},
		{"deliver_sm to a plain ESME: no simulator handler", deliverSm, ESME_RINVCMDID, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			smsc, esme, smscConnection := connectWithSilentSmsc(t)
			defer CloseAndAssertClean(smsc, esme, t)
			esme.StartControlLoop()

			if _, err := smscConnection.Write(tt.pdu); err != nil {
				t.Fatalf("writing the PDU: %v", err)
			}
			raw, err := readPduBytesFromConnection(smscConnection, time.Now().Add(time.Second))
			if err != nil {
				t.Fatalf("no answer: %v", err)
			}
			answer, err := ParsePdu(raw)
			if err != nil {
				t.Fatalf("ParsePdu() error = %v", err)
			}
			want := Header{CommandLength: 16, CommandId: "generic_nack", CommandStatus: tt.wantStatus, SequenceNumber: tt.wantSeq}
			if answer.Header != want {
				t.Errorf("answer header = %+v, want %+v", answer.Header, want)
			}
			if state := esme.GetEsmeState(); state == CLOSED {
				t.Errorf("ESME state = %s, want the connection kept", state)
			}
		})
	}
}

// A response is never answered, not even one nobody waits for.
func TestUnexpectedResponsesAreNotAnswered(t *testing.T) {
	smsc, esme, smscConnection := connectWithSilentSmsc(t)
	defer CloseAndAssertClean(smsc, esme, t)
	esme.StartControlLoop()

	unexpected, _ := EncodePdu(NewEnquireLinkResp().WithSequenceNumber(77))
	if _, err := smscConnection.Write(unexpected); err != nil {
		t.Fatalf("writing the PDU: %v", err)
	}
	if raw, err := readPduBytesFromConnection(smscConnection, time.Now().Add(200*time.Millisecond)); err == nil {
		t.Errorf("got an answer %x, want none", raw)
	}
}

// A hand-built PDU the encoder can't encode gives an error: never a panic, never corrupt bytes.
func TestEncodePduRejectsWhatItCantEncode(t *testing.T) {
	t.Parallel()
	withField := func(pdu PDU, name string, value interface{}) PDU {
		pdu.Body.MandatoryParameter[name] = value
		return pdu
	}
	tests := []struct {
		name    string
		pdu     PDU
		wantErr string
	}{
		{"int in a string field", withField(NewBindTransmitter(), "system_id", 5), "system_id of bind_transmitter pdu"},
		{"string in an integer field", withField(NewSubmitSM(), "data_coding", "8"), "data_coding of submit_sm pdu"},
		{"short_message neither text nor bytes", withField(NewSubmitSM(), "short_message", 42), "short_message of submit_sm pdu"},
		{"unknown command_id", PDU{Header: Header{CommandId: "no_such_command", CommandStatus: ESME_ROK}}, "no_such_command"},
		{"unknown command_status", NewEnquireLink().WithSMPPError("NO_SUCH_STATUS"), "NO_SUCH_STATUS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("EncodePdu() panicked: %v", r)
				}
			}()
			got, err := EncodePdu(tt.pdu)
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("EncodePdu() = %x, error %v; want an error mentioning %q", got, err, tt.wantErr)
			}
		})
	}
}

// The library logs through these; a nil one would panic on the first log line.
func TestLoggersHaveDefaults(t *testing.T) {
	for name, logger := range map[string]*log.Logger{
		"DebugSmppLogger": DebugSmppLogger, "InfoSmppLogger": InfoSmppLogger,
		"WarningSmppLogger": WarningSmppLogger, "ErrorSmppLogger": ErrorSmppLogger,
	} {
		if logger == nil {
			t.Errorf("%s is nil", name)
		}
	}
	InfoSmppLogger.Printf("logging with the default logger must not panic")
}
