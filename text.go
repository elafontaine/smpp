package smpp

import (
	"fmt"
	"unicode/utf16"
	"unicode/utf8"
)

// data_coding values (SMPP 3.4, section 5.2.19) that EncodeText and DecodeText support.
const (
	// SMSC default alphabet. What that is depends on the SMSC, so no choice works everywhere:
	// this library uses UTF-8, Go's own string encoding.
	DataCodingDefault = 0
	DataCodingIA5     = 1 // IA5 (ASCII)
	DataCodingBinary  = 2 // octet unspecified (8-bit binary)
	DataCodingLatin1  = 3 // ISO-8859-1
	DataCodingBinary8 = 4 // octet unspecified (8-bit binary)
	DataCodingUCS2    = 8 // UCS-2, big-endian (UTF-16 surrogate pairs outside the BMP)
)

// EncodeText turns text into the bytes data_coding calls for, as they go in short_message or
// message_payload. Binary data codings return the string's bytes unchanged.
func EncodeText(dataCoding int, text string) ([]byte, error) {
	switch dataCoding {
	case DataCodingDefault:
		return []byte(text), nil
	case DataCodingIA5, DataCodingLatin1:
		limit := rune(0x7f)
		if dataCoding == DataCodingLatin1 {
			limit = 0xff
		}
		out := make([]byte, 0, len(text))
		for _, r := range text {
			if r > limit {
				return nil, unencodable(r, dataCoding)
			}
			out = append(out, byte(r))
		}
		return out, nil
	case DataCodingUCS2:
		units := utf16.Encode([]rune(text))
		out := make([]byte, 0, 2*len(units))
		for _, u := range units {
			out = append(out, byte(u>>8), byte(u))
		}
		return out, nil
	case DataCodingBinary, DataCodingBinary8:
		return []byte(text), nil
	}
	return nil, fmt.Errorf("data_coding %d isn't supported", dataCoding)
}

// DecodeText turns bytes received in short_message or message_payload into text, according to
// data_coding. Binary data codings return the bytes unchanged, as a string.
func DecodeText(dataCoding int, b []byte) (string, error) {
	switch dataCoding {
	case DataCodingDefault:
		if !utf8.Valid(b) {
			return "", fmt.Errorf("bytes aren't valid UTF-8 (data_coding 0)")
		}
		return string(b), nil
	case DataCodingIA5, DataCodingLatin1:
		out := make([]rune, 0, len(b))
		for _, c := range b {
			if dataCoding == DataCodingIA5 && c > 0x7f {
				return "", invalidByte(c, dataCoding)
			}
			out = append(out, rune(c))
		}
		return string(out), nil
	case DataCodingUCS2:
		if len(b)%2 != 0 {
			return "", fmt.Errorf("UCS-2 text has an odd number of bytes (%d)", len(b))
		}
		units := make([]uint16, 0, len(b)/2)
		for i := 0; i < len(b); i += 2 {
			units = append(units, uint16(b[i])<<8|uint16(b[i+1]))
		}
		return string(utf16.Decode(units)), nil
	case DataCodingBinary, DataCodingBinary8:
		return string(b), nil
	}
	return "", fmt.Errorf("data_coding %d isn't supported", dataCoding)
}

func unencodable(r rune, dataCoding int) error {
	return fmt.Errorf("character %q can't be encoded with data_coding %d", r, dataCoding)
}

func invalidByte(c byte, dataCoding int) error {
	return fmt.Errorf("byte 0x%02x isn't valid with data_coding %d", c, dataCoding)
}
