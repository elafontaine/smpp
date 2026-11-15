package smpp

import (
	"fmt"
	"unicode/utf16"
)

// data_coding values (SMPP 3.4, section 5.2.19) that EncodeText and DecodeText support.
const (
	DataCodingDefault = 0 // SMSC default alphabet, taken as GSM 03.38 (one character per byte, unpacked)
	DataCodingIA5     = 1 // IA5 (ASCII)
	DataCodingBinary  = 2 // octet unspecified (8-bit binary)
	DataCodingLatin1  = 3 // ISO-8859-1
	DataCodingBinary8 = 4 // octet unspecified (8-bit binary)
	DataCodingUCS2    = 8 // UCS-2, big-endian (UTF-16 surrogate pairs outside the BMP)
)

const gsmEscape = 0x1b

// gsmBasic is the GSM 03.38 basic character set, indexed by byte value. 0x1b is the escape to
// the extension table, never a character by itself.
var gsmBasic = []rune("@£$¥èéùìòÇ\nØø\rÅåΔ_ΦΓΛΩΠΨΣΘΞ\x1bÆæßÉ !\"#¤%&'()*+,-./0123456789:;<=>?" +
	"¡ABCDEFGHIJKLMNOPQRSTUVWXYZÄÖÑÜ§¿abcdefghijklmnopqrstuvwxyzäöñüà")

// gsmExtension is the GSM 03.38 extension table: each character is sent as 0x1b + its byte.
var gsmExtension = map[byte]rune{
	0x0a: '\f', 0x14: '^', 0x28: '{', 0x29: '}', 0x2f: '\\',
	0x3c: '[', 0x3d: '~', 0x3e: ']', 0x40: '|', 0x65: '€',
}

var gsmBasicByRune, gsmExtensionByRune = func() (map[rune]byte, map[rune]byte) {
	basic := make(map[rune]byte, len(gsmBasic))
	for i, r := range gsmBasic {
		if i != gsmEscape {
			basic[r] = byte(i)
		}
	}
	extension := make(map[rune]byte, len(gsmExtension))
	for b, r := range gsmExtension {
		extension[r] = b
	}
	return basic, extension
}()

// EncodeText turns text into the bytes data_coding calls for, as they go in short_message or
// message_payload. Binary data codings return the string's bytes unchanged.
func EncodeText(dataCoding int, text string) ([]byte, error) {
	switch dataCoding {
	case DataCodingDefault:
		out := make([]byte, 0, len(text))
		for _, r := range text {
			if b, ok := gsmBasicByRune[r]; ok {
				out = append(out, b)
			} else if b, ok := gsmExtensionByRune[r]; ok {
				out = append(out, gsmEscape, b)
			} else {
				return nil, unencodable(r, dataCoding)
			}
		}
		return out, nil
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
		out := make([]rune, 0, len(b))
		for i := 0; i < len(b); i++ {
			switch {
			case b[i] > 0x7f:
				return "", invalidByte(b[i], dataCoding)
			case b[i] != gsmEscape:
				out = append(out, gsmBasic[b[i]])
			case i+1 == len(b):
				return "", fmt.Errorf("escape (0x1b) without a following character")
			default:
				i++
				r, ok := gsmExtension[b[i]]
				if !ok {
					return "", fmt.Errorf("escape sequence 0x1b 0x%02x isn't in the extension table", b[i])
				}
				out = append(out, r)
			}
		}
		return string(out), nil
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
