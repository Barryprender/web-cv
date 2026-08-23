package pdf

import (
	"bytes"
	"strings"
)

// WinAnsiEncoding is the single-byte encoding the font dictionaries declare.
// Codes 0x20–0x7E and 0xA0–0xFF line up with Latin-1, so only the 0x80–0x9F
// block needs an explicit table: those codes hold the typographic characters
// Windows put there, and they are exactly the ones CV copy runs into — em and
// en dashes, curly quotes, the bullet, the ellipsis.
var winAnsiHigh = map[rune]byte{
	'\u20AC': 0x80, // euro
	'\u201A': 0x82, // single low quote
	'\u0192': 0x83, // florin
	'\u201E': 0x84, // double low quote
	'\u2026': 0x85, // ellipsis
	'\u2020': 0x86, // dagger
	'\u2021': 0x87, // double dagger
	'\u02C6': 0x88, // circumflex
	'\u2030': 0x89, // per mille
	'\u0160': 0x8A, // S caron
	'\u2039': 0x8B, // single left guillemet
	'\u0152': 0x8C, // OE
	'\u017D': 0x8E, // Z caron
	'\u2018': 0x91, // left single quote
	'\u2019': 0x92, // right single quote
	'\u201C': 0x93, // left double quote
	'\u201D': 0x94, // right double quote
	'\u2022': 0x95, // bullet
	'\u2013': 0x96, // en dash
	'\u2014': 0x97, // em dash
	'\u02DC': 0x98, // small tilde
	'\u2122': 0x99, // trademark
	'\u0161': 0x9A, // s caron
	'\u203A': 0x9B, // single right guillemet
	'\u0153': 0x9C, // oe
	'\u017E': 0x9E, // z caron
	'\u0178': 0x9F, // Y dieresis
}

// asciiFallback covers characters with no WinAnsi code but an unambiguous
// plain-text stand-in. Silently dropping them would be worse: a missing
// character in a CV reads as a typo, whereas "->" reads as a deliberate
// compromise.
var asciiFallback = map[rune]string{
	'\u2192': "->",
	'\u2190': "<-",
	'\u21D2': "=>",
	'\u2264': "<=",
	'\u2265': ">=",
	'\u2260': "!=",
	'\u00A0': " ", // non-breaking space, which has its own code but is safer flattened
	'\u2009': " ",
	'\u200B': "",
	'\u2011': "-",
	'\u2015': "-",
	'\u00D7': "x",
}

// encode converts a UTF-8 string to WinAnsi bytes.
//
// Anything that maps to no code and has no fallback becomes '?', which is
// deliberately visible: an unrenderable character should show up in review
// rather than vanish from the finished CV.
func encode(s string) []byte {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 0x20 && r <= 0x7E:
			out = append(out, byte(r))
		case r >= 0xA0 && r <= 0xFF:
			out = append(out, byte(r))
		default:
			if code, ok := winAnsiHigh[r]; ok {
				out = append(out, code)
				continue
			}
			if alt, ok := asciiFallback[r]; ok {
				out = append(out, alt...)
				continue
			}
			out = append(out, '?')
		}
	}
	return out
}

// escapeString escapes the three bytes that are structural inside a PDF
// literal string. Everything else, high bytes included, is written raw.
func escapeString(b []byte) []byte {
	if !bytes.ContainsAny(b, structuralBytes) {
		return b
	}
	out := make([]byte, 0, len(b)+8)
	for _, c := range b {
		if strings.IndexByte(structuralBytes, c) >= 0 {
			out = append(out, '\\')
		}
		out = append(out, c)
	}
	return out
}

// structuralBytes are the bytes a PDF literal string cannot carry unescaped.
const structuralBytes = `()\`
