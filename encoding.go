package main

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// validEncodings are the accepted values for a rule's Encoding field.
var validEncodings = map[string]bool{
	"utf8":    true,
	"utf8bom": true,
	"sjis":    true,
	"utf16le": true,
	"utf16be": true,
}

// decodingReader wraps r so reads are decoded from enc to UTF-8. enc must be
// one of validEncodings, or empty (treated as utf8).
func decodingReader(enc string, r io.Reader) (io.Reader, error) {
	switch strings.ToLower(enc) {
	case "", "utf8":
		return r, nil
	case "utf8bom":
		return transform.NewReader(r, unicode.BOMOverride(unicode.UTF8.NewDecoder())), nil
	case "sjis":
		return transform.NewReader(r, japanese.ShiftJIS.NewDecoder()), nil
	case "utf16le":
		return transform.NewReader(r, unicode.UTF16(unicode.LittleEndian, unicode.IgnoreBOM).NewDecoder()), nil
	case "utf16be":
		return transform.NewReader(r, unicode.UTF16(unicode.BigEndian, unicode.IgnoreBOM).NewDecoder()), nil
	default:
		return nil, fmt.Errorf("unknown Encoding %q", enc)
	}
}
