package smpp

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDefaultValueForNewBindTransmitterAndDefaultBindBody(t *testing.T) {
	t.Parallel()
	defaultBindTransmitter := PDU{
		header: Header{
			0,
			"bind_transmitter",
			ESME_ROK,
			0,
		},
		body: Body{
			mandatoryParameter: map[string]interface{}{
				"system_id":         "",
				"password":          "",
				"system_type":       "",
				"interface_version": 52,
				"addr_ton":          0,
				"addr_npi":          0,
				"address_range":     "",
			},
			optionalParameters: nil,
		},
	}

	t.Run("instantiating bind_transmitter", func(t *testing.T) {
		comparePdu(NewBindTransmitter(),
			defaultBindTransmitter, t)
	})
}

func TestDefaultValueForNewBindReceiver(t *testing.T) {
	t.Parallel()

	defaultBindReceiver := PDU{
		header: Header{
			0,
			"bind_receiver",
			ESME_ROK,
			0,
		},
		body: defaultBindBody(),
	}

	t.Run("instantiating bind_receiver", func(t *testing.T) {
		if got := NewBindReceiver(); !reflect.DeepEqual(got, defaultBindReceiver) {
			t.Errorf("NewBindReceiver() = %v, want %v", got, defaultBindReceiver)
		}
	})
}

func TestDefaultValueForNewBindTransceiver(t *testing.T) {
	t.Parallel()

	defaultBindTransceiver := PDU{
		header: Header{
			0,
			"bind_transceiver",
			ESME_ROK,
			0,
		},
		body: defaultBindBody(),
	}

	t.Run("instantiating bind_transceiver", func(t *testing.T) {
		comparePdu(NewBindTransceiver(), defaultBindTransceiver, t)
	})
}

func TestBindTransmitterWithBuilderPatternToPdu(t *testing.T) {
	t.Parallel()
	expectedBindTransmitterObj := bindTransmitterObj
	expectedBindTransmitterObj.body = defaultBindBody()
	expectedBindTransmitterObj.body.mandatoryParameter["address_range"] = "44601"
	expectedBindTransmitterObj.body.mandatoryParameter["system_type"] = "VMS"
	expectedBindTransmitterObj.body.mandatoryParameter["password"] = "test"
	expectedBindTransmitterObj.body.mandatoryParameter["system_id"] = "test"
	expectedBindTransmitterObj.body.mandatoryParameter["interface_version"] = 34
	expectedBindTransmitterObj.body.mandatoryParameter["addr_ton"] = 2
	expectedBindTransmitterObj.body.mandatoryParameter["addr_npi"] = 1
	expectedBindTransmitterObj.header.commandLength = 39

	bindTransmiter := NewBindTransmitter().
		WithSystemId("test").
		WithPassword("test").
		WithAddressRange("44601").
		WithSystemType("VMS").
		WithInterfaceVersion(34).
		WithAddressTon(2).
		WithAddressNpi(1)
	binaryPdu, _ := EncodePdu(bindTransmiter)
	bindTransmiter.header.commandLength = len(binaryPdu)

	t.Run("Constructor pattern for binds", func(t *testing.T) {
		comparePdu(bindTransmiter, expectedBindTransmitterObj, t)
	})
}

func TestNewBindTransmitterResp(t *testing.T) {
	t.Parallel()
	expectedBindTransmitterResp := bindTransmitterRespObj
	expectedBindTransmitterResp.header.commandLength = 0

	actualResponse := NewBindTransmitterResp().WithSystemId("test")
	t.Run("instantiating bind_transmitter_resp", func(t *testing.T) {
		comparePdu(actualResponse,
			expectedBindTransmitterResp, t)
	})
}

func TestNewBindTransceiverResp(t *testing.T) {
	t.Parallel()
	expectedBindTransceiverResp := bindTransmitterRespObj
	expectedBindTransceiverResp.header.commandLength = 0
	expectedBindTransceiverResp.header.commandId = "bind_transceiver_resp"

	actualResponse := NewBindTransceiverResp().WithSystemId("test")
	t.Run("instantiating bind_transceiver_resp", func(t *testing.T) {
		comparePdu(actualResponse,
			expectedBindTransceiverResp, t)
	})
}

func TestNewBindReceiverResp(t *testing.T) {
	t.Parallel()
	expectedBindReceiverResp := bindTransmitterRespObj
	expectedBindReceiverResp.header.commandLength = 0
	expectedBindReceiverResp.header.commandId = "bind_receiver_resp"

	actualResponse := NewBindReceiverResp().WithSystemId("test")
	t.Run("instantiating bind_receiver_resp", func(t *testing.T) {
		comparePdu(actualResponse,
			expectedBindReceiverResp, t)
	})
}

func TestSubmitSMDefaultValues(t *testing.T) {
	t.Parallel()
	defaultSubmitSMObj := PDU{
		header: Header{
			commandLength:  0,
			commandId:      "submit_sm",
			commandStatus:  ESME_ROK,
			sequenceNumber: 0,
		},
		body: defaultSubmitSmBody(),
	}
	t.Run("Constructor pattern for submit_sm", func(t *testing.T) {
		comparePdu(NewSubmitSM(), defaultSubmitSMObj, t)
	})
}

func TestSubmitSMWithBuilderPatternToPdu(t *testing.T) {
	t.Parallel()
	expectedSubmitSm := submitSmObj
	expectedSubmitSm.body = defaultSubmitSmBody()
	expectedSubmitSm.body.mandatoryParameter["source_addr"] = "1234"
	expectedSubmitSm.body.mandatoryParameter["source_addr_ton"] = 2
	expectedSubmitSm.body.mandatoryParameter["source_addr_npi"] = 1
	expectedSubmitSm.body.mandatoryParameter["destination_addr"] = "12345"
	expectedSubmitSm.body.mandatoryParameter["dest_addr_ton"] = 2
	expectedSubmitSm.body.mandatoryParameter["dest_addr_npi"] = 1
	expectedSubmitSm.body.mandatoryParameter["data_coding"] = 8
	expectedSubmitSm.body.mandatoryParameter["short_message"] = "Hello"
	expectedSubmitSm.header.commandLength = 0

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

func TestDeliverSmInstantiation(t *testing.T) {
	t.Parallel()
	expectedDeliverSM := deliverSmObj
	expectedDeliverSM.body = defaultSubmitSmBody()
	expectedDeliverSM.header.commandLength = 0
	expectedDeliverSM.header.sequenceNumber = 0

	actualDeliverSM := NewDeliverSM()

	t.Run("Constructor Pattern for deliverSM ", func(t *testing.T) {
		comparePdu(actualDeliverSM, expectedDeliverSM, t)
	})
}

func TestDataSmInstantiation(t *testing.T) {
	t.Parallel()
	expectedDataSm := dataSmObj
	expectedDataSm.body = defaultSubmitSmBody()
	expectedDataSm.header.commandLength = 0
	expectedDataSm.header.sequenceNumber = 0

	actualDataSM := NewDataSM()

	t.Run("Constructor Pattern for dataSM ", func(t *testing.T) {
		comparePdu(actualDataSM, expectedDataSm, t)
	})
}

func TestSubmitSmRespInstantiation(t *testing.T) {
	t.Parallel()
	expectedSubmitSmResp := submitSmRespObj
	expectedSubmitSmResp.header.commandLength = 0
	expectedSubmitSmResp.header.sequenceNumber = 0

	actualSubmitSmResp := NewSubmitSMResp().
		WithMessageId("1")

	t.Run("Constructor Pattern for SubmitSMResp ", func(t *testing.T) {
		comparePdu(actualSubmitSmResp, expectedSubmitSmResp, t)
	})
}

func TestDeliverSmRespInstantiation(t *testing.T) {
	t.Parallel()
	expectedDeliverSmResp := deliverSmRespObj
	expectedDeliverSmResp.header.commandLength = 0
	expectedDeliverSmResp.header.sequenceNumber = 0
	expectedDeliverSmResp.body = Body{
		mandatoryParameter: map[string]interface{}{},
		optionalParameters: nil,
	}
	expectedDeliverSmResp.body.mandatoryParameter["message_id"] = "2"

	actualDeliverSmResp := NewDeliverSMResp().
		WithMessageId("2")

	t.Run("Constructor Pattern for DeliverSmResp ", func(t *testing.T) {
		comparePdu(actualDeliverSmResp, expectedDeliverSmResp, t)
	})
}

func TestGenericNACK(t *testing.T) {
	t.Parallel()
	expectedDeliverSmResp := PDU{header: generickNackHeader}
	expectedDeliverSmResp.header.commandLength = 0
	expectedDeliverSmResp.header.sequenceNumber = 0

	actualDeliverSmResp := NewGenerickNack().withSMPPError("ESME_RINVSRCADR")

	t.Run("Constructor Pattern for DeliverSmResp ", func(t *testing.T) {
		comparePdu(actualDeliverSmResp, expectedDeliverSmResp, t)
	})
}

func TestPDUObjectsShouldBeDifferentInMemoryToAvoidSharedObjects(t *testing.T) {
	t.Parallel()
	NotTheExpectedPdu := NewBindTransmitter().WithSystemId("SystemId").WithPassword("Password")
	expectedBuf, err := EncodePdu(NotTheExpectedPdu)
	if err != nil {
		t.Errorf("Couldn't make a simple PDU...")
	}
	actualPdu := NewBindTransmitter().WithSystemId("SystemID2")
	actualBuf, err := EncodePdu(actualPdu)
	if err != nil {
		t.Errorf("Couldn't make a simple PDU...")
	}
	if bytes.Equal(actualBuf, expectedBuf) {
		t.Errorf("But PDUs are considered equal... something isn't working")
	}
}

func comparePdu(actualPdu PDU, expectedPdu PDU, t *testing.T) {
	if got := actualPdu; !reflect.DeepEqual(got, expectedPdu) {
		if !reflect.DeepEqual(got.header, expectedPdu.header) {
			t.Errorf("Difference in the header structure")
		}
		if !reflect.DeepEqual(got.body, expectedPdu.body) {
			t.Errorf("Difference in the body structure")
		}
		t.Errorf("The constructor pattern isn't creating the expected PDU object! %v, want %v", got, expectedPdu)
	}
}
