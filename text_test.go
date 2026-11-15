package smpp

import (
	"bytes"
	"encoding/hex"
	"testing"
)

var textCases = []struct {
	name       string
	dataCoding int
	text       string
	wireHex    string
}{
	{"default alphabet (GSM 03.38), ASCII letters", DataCodingDefault, "hello", "68656c6c6f"},
	{"default alphabet, characters away from their ASCII positions", DataCodingDefault, "@£$¥_", "0001020311"},
	{"default alphabet, extension table characters are escaped", DataCodingDefault, "€[", "1b651b3c"},
	{"IA5 (ASCII)", DataCodingIA5, "hello", "68656c6c6f"},
	{"Latin-1", DataCodingLatin1, "café", "636166e9"},
	{"UCS-2, two Chinese characters", DataCodingUCS2, "你好", "4f60597d"},
	{"UCS-2, outside the BMP becomes a surrogate pair", DataCodingUCS2, "😀", "d83dde00"},
	{"binary, bytes unchanged", DataCodingBinary, "\x00\xff", "00ff"},
}

func TestEncodeText(t *testing.T) {
	t.Parallel()
	for _, tt := range textCases {
		t.Run(tt.name, func(t *testing.T) {
			want, _ := hex.DecodeString(tt.wireHex)
			got, err := EncodeText(tt.dataCoding, tt.text)
			if err != nil {
				t.Fatalf("EncodeText() error = %v", err)
			}
			if !bytes.Equal(got, want) {
				t.Errorf("EncodeText() = %x, want %x", got, want)
			}
		})
	}
}

func TestDecodeText(t *testing.T) {
	t.Parallel()
	for _, tt := range textCases {
		t.Run(tt.name, func(t *testing.T) {
			wire, _ := hex.DecodeString(tt.wireHex)
			got, err := DecodeText(tt.dataCoding, wire)
			if err != nil {
				t.Fatalf("DecodeText() error = %v", err)
			}
			if got != tt.text {
				t.Errorf("DecodeText() = %q, want %q", got, tt.text)
			}
		})
	}
}

func TestEncodeTextErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dataCoding int
		text       string
		wantErr    string
	}{
		{"Chinese in the default alphabet", DataCodingDefault, "你", "character '你' can't be encoded with data_coding 0"},
		{"accent in IA5", DataCodingIA5, "é", "character 'é' can't be encoded with data_coding 1"},
		{"Chinese in Latin-1", DataCodingLatin1, "你", "character '你' can't be encoded with data_coding 3"},
		{"unsupported data_coding", 5, "x", "data_coding 5 isn't supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncodeText(tt.dataCoding, tt.text)
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("EncodeText() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestDecodeTextErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		dataCoding int
		wireHex    string
		wantErr    string
	}{
		{"UCS-2 with an odd number of bytes", DataCodingUCS2, "4f6059", "UCS-2 text has an odd number of bytes (3)"},
		{"IA5 byte above 0x7f", DataCodingIA5, "80", "byte 0x80 isn't valid with data_coding 1"},
		{"default alphabet byte above 0x7f", DataCodingDefault, "80", "byte 0x80 isn't valid with data_coding 0"},
		{"default alphabet escape at the end", DataCodingDefault, "411b", "escape (0x1b) without a following character"},
		{"default alphabet unknown escape", DataCodingDefault, "1b41", "escape sequence 0x1b 0x41 isn't in the extension table"},
		{"unsupported data_coding", 5, "41", "data_coding 5 isn't supported"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wire, _ := hex.DecodeString(tt.wireHex)
			_, err := DecodeText(tt.dataCoding, wire)
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("DecodeText() error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
