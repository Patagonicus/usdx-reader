package encoding

import (
	enc "golang.org/x/text/encoding"
	"golang.org/x/text/encoding/charmap"
	"golang.org/x/text/encoding/unicode"
)

var (
	UTF8 = Decoder{
		DecoderName: "UTF8",
		Decoder:     unicode.UTF8.NewDecoder(),
	}
	CP1250 = Decoder{
		DecoderName: "CP1250",
		Decoder:     charmap.Windows1250.NewDecoder(),
	}
	CP1252 = Decoder{
		DecoderName: "CP1252",
		Decoder:     charmap.Windows1252.NewDecoder(),
	}
	Auto = UTF8DetectingDecoder{
		UTF8:     unicode.UTF8.NewDecoder(),
		Fallback: charmap.Windows1250.NewDecoder(),
	}
)

type Decoder struct {
	DecoderName string
	Decoder     *enc.Decoder
}

func (d Decoder) Name() string {
	return d.DecoderName
}

func (d Decoder) Decode(s string) (string, error) {
	return d.Decoder.String(s)
}

type UTF8DetectingDecoder struct {
	UTF8     *enc.Decoder
	Fallback *enc.Decoder
}

func (d UTF8DetectingDecoder) Name() string {
	return "Auto"
}

func (d UTF8DetectingDecoder) Decode(s string) (string, error) {
	if isUTF8(s) {
		return d.UTF8.String(s)
	}
	return d.Fallback.String(s)
}

func isUTF8(s string) bool {
	// taken from USDX source code
	state := 0
	bytes := []byte(s)
	for i := 0; i < len(bytes); i++ {
		c := 0x100*state + int(bytes[i])
		switch {
		case c == 0x09 || c == 0x0A || c == 0x0D || (c >= 0x20 && c <= 0x7E):
			state = 0
		case c >= 0xC2 && c <= 0xDF:
			state = 1
		case c == 0xE0:
			state = 2
		case (c >= 0xE1 && c <= 0xEC) || c == 0xEE || c == 0xEF:
			state = 3
		case c == 0xED:
			state = 4
		case c == 0xF0:
			state = 5
		case c >= 0xF1 && c <= 0xF3:
			state = 6
		case c == 0xF4:
			state = 7
		case c >= 0x180 && c <= 0x1BF:
			state = 0
		case c >= 0x2A0 && c <= 0x2BF:
			state = 1
		case c >= 0x380 && c <= 0x3BF:
			state = 1
		case c >= 0x480 && c <= 0x49F:
			state = 1
		case c >= 0x590 && c <= 0x5BF:
			state = 3
		case c >= 0x680 && c <= 0x6BF:
			state = 3
		case c >= 0x780 && c <= 0x78F:
			state = 3
		default:
			return false
		}
	}
	return state == 0
}
