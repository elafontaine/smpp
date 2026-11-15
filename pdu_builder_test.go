package smpp

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDefaultValueForNewBindTransmitterAndDefaultBindBody(t *testing.T) {
	t.Parallel()
	type args struct {
		builder_function func() PDU
	}
	tests := []struct {
		name    string
		args    args
		wantPDU string
	}{
		{"instantiating bind_transmitter and default body", args{NewBindTransmitter}, "bind_transmitter"},
		{"instantiating bind_transceiver", args{NewBindTransceiver}, "bind_transceiver"},
		{"instantiating bind_transmitter", args{NewBindReceiver}, "bind_receiver"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultBindPdu := PDU{
				Header: Header{
					0,
					tt.wantPDU,
					ESME_ROK,
					0,
				},
				Body: Body{
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
				},
			}
			comparePdu(tt.args.builder_function(), defaultBindPdu, t)

		})
	}
}

func TestBindTransmitterWithBuilderPatternToPdu(t *testing.T) {
	t.Parallel()
	expectedBindTransmitterObj := bindTransmitterObj
	expectedBindTransmitterObj.Body = defaultBindBody()
	expectedBindTransmitterObj.Body.MandatoryParameter["address_range"] = "44601"
	expectedBindTransmitterObj.Body.MandatoryParameter["system_type"] = "VMS"
	expectedBindTransmitterObj.Body.MandatoryParameter["password"] = "test"
	expectedBindTransmitterObj.Body.MandatoryParameter["system_id"] = "test"
	expectedBindTransmitterObj.Body.MandatoryParameter["interface_version"] = 34
	expectedBindTransmitterObj.Body.MandatoryParameter["addr_ton"] = 2
	expectedBindTransmitterObj.Body.MandatoryParameter["addr_npi"] = 1
	expectedBindTransmitterObj.Header.CommandLength = 39

	bindTransmiter := NewBindTransmitter().
		WithSystemId("test").
		WithPassword("test").
		WithAddressRange("44601").
		WithSystemType("VMS").
		WithInterfaceVersion(34).
		WithAddressTon(2).
		WithAddressNpi(1)
	binaryPdu, _ := EncodePdu(bindTransmiter)
	bindTransmiter.Header.CommandLength = len(binaryPdu)

	t.Run("Constructor pattern for binds", func(t *testing.T) {
		comparePdu(bindTransmiter, expectedBindTransmitterObj, t)
	})
}

func TestNewBindTransmitterResp(t *testing.T) {
	t.Parallel()
	expectedBindTransmitterResp := bindTransmitterRespObj
	expectedBindTransmitterResp.Header.CommandLength = 0

	actualResponse := NewBindTransmitterResp().WithSystemId("test")
	t.Run("instantiating bind_transmitter_resp", func(t *testing.T) {
		comparePdu(actualResponse, expectedBindTransmitterResp, t)
	})
}

func TestNewBindTransceiverResp(t *testing.T) {
	t.Parallel()
	expectedBindTransceiverResp := bindTransmitterRespObj
	expectedBindTransceiverResp.Header.CommandLength = 0
	expectedBindTransceiverResp.Header.CommandId = "bind_transceiver_resp"

	actualResponse := NewBindTransceiverResp().WithSystemId("test")
	t.Run("instantiating bind_transceiver_resp", func(t *testing.T) {
		comparePdu(actualResponse, expectedBindTransceiverResp, t)
	})
}

func TestNewBindReceiverResp(t *testing.T) {
	t.Parallel()
	expectedBindReceiverResp := bindTransmitterRespObj
	expectedBindReceiverResp.Header.CommandLength = 0
	expectedBindReceiverResp.Header.CommandId = "bind_receiver_resp"

	actualResponse := NewBindReceiverResp().WithSystemId("test")
	t.Run("instantiating bind_receiver_resp", func(t *testing.T) {
		comparePdu(actualResponse, expectedBindReceiverResp, t)
	})
}

func TestSubmitSMWithBuilderPatternToPdu(t *testing.T) {
	t.Parallel()
	expectedSubmitSm := submitSmObj
	expectedSubmitSm.Body = defaultMessageBody()
	expectedSubmitSm.Body.MandatoryParameter["source_addr"] = "1234"
	expectedSubmitSm.Body.MandatoryParameter["source_addr_ton"] = 2
	expectedSubmitSm.Body.MandatoryParameter["source_addr_npi"] = 1
	expectedSubmitSm.Body.MandatoryParameter["destination_addr"] = "12345"
	expectedSubmitSm.Body.MandatoryParameter["dest_addr_ton"] = 2
	expectedSubmitSm.Body.MandatoryParameter["dest_addr_npi"] = 1
	expectedSubmitSm.Body.MandatoryParameter["data_coding"] = 8
	expectedSubmitSm.Body.MandatoryParameter["short_message"] = "Hello"
	expectedSubmitSm.Header.CommandLength = 0

	actualSubmitSm := NewSubmitSM().
		WithSourceAddress("1234").
		WithSourceAddressTon(2).
		WithSourceAddressNpi(1).
		WithDestinationAddress("12345").
		WithDestinationAddressNpi(1).
		WithDestinationAddressTon(2).
		WithDataCoding(8).
		WithMessage("Hello")

	t.Run("Constructor pattern for binds", func(t *testing.T) {
		comparePdu(actualSubmitSm, expectedSubmitSm, t)
	})
}

func TestSubmitSmRespInstantiation(t *testing.T) {
	t.Parallel()
	expectedSubmitSmResp := submitSmRespObj
	expectedSubmitSmResp.Header.CommandLength = 0
	expectedSubmitSmResp.Header.SequenceNumber = 0

	actualSubmitSmResp := NewSubmitSMResp().
		WithMessageId("1")

	t.Run("Constructor Pattern for SubmitSMResp ", func(t *testing.T) {
		comparePdu(actualSubmitSmResp, expectedSubmitSmResp, t)
	})
}

func TestDeliverSmRespInstantiation(t *testing.T) {
	t.Parallel()
	expectedDeliverSmResp := deliverSmRespObj
	expectedDeliverSmResp.Header.CommandLength = 0
	expectedDeliverSmResp.Header.SequenceNumber = 0
	expectedDeliverSmResp.Body = Body{
		MandatoryParameter: map[string]interface{}{},
		OptionalParameters: nil,
	}
	expectedDeliverSmResp.Body.MandatoryParameter["message_id"] = "2"

	actualDeliverSmResp := NewDeliverSMResp().
		WithMessageId("2")

	t.Run("Constructor Pattern for DeliverSmResp ", func(t *testing.T) {
		comparePdu(actualDeliverSmResp, expectedDeliverSmResp, t)
	})
}

func TestGenericNACK(t *testing.T) {
	t.Parallel()
	expectedDeliverSmResp := PDU{Header: generickNackHeader}
	expectedDeliverSmResp.Header.CommandLength = 0
	expectedDeliverSmResp.Header.SequenceNumber = 0

	actualDeliverSmResp := NewGenerickNack().WithSMPPError("ESME_RINVSRCADR")

	t.Run("Constructor Pattern for DeliverSmResp ", func(t *testing.T) {
		comparePdu(actualDeliverSmResp, expectedDeliverSmResp, t)
	})
}

func TestEnquiryLink(t *testing.T) {
	t.Parallel()

	expectedEnquiryLink := PDU{
		Header: Header{
			CommandId:     "enquire_link",
			CommandStatus: ESME_ROK,
		},
	}
	expectedEnquiryLink.Header.CommandLength = 0
	expectedEnquiryLink.Header.SequenceNumber = 0

	actualEnquiryLink := NewEnquireLink()
	t.Run("Constructor Pattern for EnquiryLink ", func(t *testing.T) {
		comparePdu(actualEnquiryLink, expectedEnquiryLink, t)
	})
}

func TestEnquiryLinkResp(t *testing.T) {
	t.Parallel()

	expectedEnquiryLink := PDU{
		Header: Header{
			CommandId:     "enquire_link_resp",
			CommandStatus: ESME_ROK,
		},
		Body: Body{
			MandatoryParameter: map[string]interface{}{},
		},
	}
	expectedEnquiryLink.Header.CommandLength = 0
	expectedEnquiryLink.Header.SequenceNumber = 0

	actualEnquiryLink := NewEnquireLinkResp()
	t.Run("Constructor Pattern for EnquiryLink ", func(t *testing.T) {
		comparePdu(actualEnquiryLink, expectedEnquiryLink, t)
	})
}

// The message delivery PDUs (submit_sm, deliver_sm, data_sm) share one default body. It's written
// out here, not taken from defaultMessageBody(), so a changed default makes this test fail.
func TestSubmitSmFamilyDefaultValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		builder       func() PDU
		wantCommandId string
	}{
		{"submit_sm", NewSubmitSM, "submit_sm"},
		{"deliver_sm", NewDeliverSM, "deliver_sm"},
		{"data_sm", NewDataSM, "data_sm"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			want := PDU{
				Header: Header{CommandLength: 0, CommandId: tt.wantCommandId, CommandStatus: ESME_ROK, SequenceNumber: 0},
				Body: Body{
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
				},
			}
			comparePdu(tt.builder(), want, t)
		})
	}
}

// Each builder call must return its own parameter map: changing one PDU must not change
// another one built earlier.
func TestPDUObjectsShouldBeDifferentInMemoryToAvoidSharedObjects(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		builder func() PDU
	}{
		{"bind_transmitter", NewBindTransmitter},
		{"bind_transceiver", NewBindTransceiver},
		{"bind_receiver", NewBindReceiver},
		{"submit_sm", NewSubmitSM},
		{"deliver_sm", NewDeliverSM},
		{"data_sm", NewDataSM},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			first := tt.builder()
			before := fmt.Sprint(first.Body.MandatoryParameter)

			second := tt.builder()
			second.Body.MandatoryParameter["changed_on_the_second_"+tt.name] = true

			if after := fmt.Sprint(first.Body.MandatoryParameter); after != before {
				t.Errorf("changing the second %s PDU changed the first one:\nbefore %s\nafter  %s", tt.name, before, after)
			}
		})
	}
}

func TestBuildingWithDefaults(t *testing.T) {
	destination := "5551234567"
	source := "5557654321"
	expectedPdu := NewSubmitSM().
		WithDataCoding(8).
		WithDestinationAddress(destination).
		WithSourceAddress(source)
	pduDefault := map[string]interface{}{
		"destination_addr": destination,
		"source_addr":      source,
		"data_coding":      8,
	}

	pduToApply := NewSubmitSM().WithDefaults(pduDefault)
	comparePdu(pduToApply, expectedPdu, t)

}

func comparePdu(actualPdu PDU, expectedPdu PDU, t *testing.T) {
	if got := actualPdu; !reflect.DeepEqual(got, expectedPdu) {
		if !reflect.DeepEqual(got.Header, expectedPdu.Header) {
			t.Errorf("Difference in the header structure")
		}
		if !reflect.DeepEqual(got.Body, expectedPdu.Body) {
			t.Errorf("Difference in the body structure")
		}
		t.Errorf("The constructor pattern isn't creating the expected PDU object! %v, want %v", got, expectedPdu)
	}
}
