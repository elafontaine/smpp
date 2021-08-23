package smpp

import (
	"reflect"
	"testing"
)

func TestDefaultValueForNewBindTransmitter(t *testing.T) {

	defaultBindTransmitter := &PDU{
		header: Header{
			0,
			"bind_transmitter",
			"ESME_ROK",
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
		if got := NewBindTransmitter(); !reflect.DeepEqual(got, defaultBindTransmitter) {
			t.Errorf("NewBindTransmitter() = %v, want %v", got, defaultBindTransmitter)
		}
	})
}

func TestBindTransmitterToPdu(t *testing.T) {
	bindTransmiter := NewBindTransmitter().WithSystemId("test").WithPassword("test")
	binaryPdu, _ := EncodePdu(bindTransmiter)
	bindTransmiter.header.commandLength = len(binaryPdu)
	t.Run("Constructor pattern for binds", func(t *testing.T) {
		if got := bindTransmiter; !reflect.DeepEqual(got, bindTransmitterObj) {
			t.Errorf("The constructor pattern isn't creating expected PDU object! %v, want %v", got, bindTransmitterObj)
		}
	})
}

func TestDefaultValueForNewBindReceiver(t *testing.T) {

	defaultBindReceiver := &PDU{
		header: Header{
			0,
			"bind_receiver",
			"ESME_ROK",
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

	defaultBindTransceiver := &PDU{
		header: Header{
			0,
			"bind_transceiver",
			"ESME_ROK",
			0,
		},
		body: defaultBindBody(),
	}

	t.Run("instantiating bind_transceiver", func(t *testing.T) {
		if got := NewBindTransceiver(); !reflect.DeepEqual(got, defaultBindTransceiver) {
			t.Errorf("NewBindTransceiver() = %v, want %v", got, defaultBindTransceiver)
		}
	})
}