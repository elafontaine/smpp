package smpp

import (
	"reflect"
	"testing"
)

func TestNewBindTransmitter(t *testing.T) {

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
				"interface_version": "34",
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
	bindTransmiter := NewBindTransmitter().withSystemId("test").withPassword("test")
	binaryPdu, _ := EncodePdu(bindTransmiter)
	bindTransmiter.header.commandLength = len(binaryPdu)
	t.Run("Constructor pattern for binds", func(t *testing.T) {
		if got := bindTransmiter; !reflect.DeepEqual(got, bindTransmitterObj) {
			t.Errorf("The constructor pattern isn't creating expected PDU object! %v, want %v", got, bindTransmitterObj)
		}
	})
}
