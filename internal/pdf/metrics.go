package pdf

// Glyph metrics for the base-14 fonts.
//
// Helvetica, Helvetica-Bold and Helvetica-Oblique are three of the fourteen
// fonts every conforming PDF reader is required to carry, so the file never
// has to embed a font program. That is the whole reason this package can stay
// on the standard library: embedding a real typeface would mean parsing woff2
// (Brotli) and rebuilding a TrueType subset, and none of that ships with Go.
//
// The trade is that the PDF is set in Helvetica rather than the site's Public
// Sans. The text is still real, selectable, copyable text — which is what
// applicant tracking systems parse — so the loss is purely aesthetic.
//
// Widths are advance widths in 1/1000 of an em, indexed by WinAnsiEncoding
// code, taken from Adobe's AFM metrics for the base-14 set. Codes that encode
// no glyph are 0. Helvetica-Oblique is metrically identical to Helvetica, so
// it shares the table.

// helveticaWidths is the advance-width table for Helvetica and
// Helvetica-Oblique. Each row is sixteen codes, starting at the code in the
// comment.
var helveticaWidths = [256]uint16{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x00
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x10
	278, 278, 355, 556, 556, 889, 667, 191, 333, 333, 389, 584, 278, 333, 278, 278, // 0x20
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 278, 278, 584, 584, 584, 556, // 0x30
	1015, 667, 667, 722, 722, 667, 611, 778, 722, 278, 500, 667, 556, 833, 722, 778, // 0x40
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 278, 278, 278, 469, 556, // 0x50
	333, 556, 556, 500, 556, 556, 278, 556, 556, 222, 222, 500, 222, 833, 556, 556, // 0x60
	556, 556, 333, 500, 278, 556, 500, 722, 500, 500, 500, 334, 260, 334, 584, 0, // 0x70
	556, 0, 222, 556, 333, 1000, 556, 556, 333, 1000, 667, 333, 1000, 0, 611, 0, // 0x80
	0, 222, 222, 333, 333, 350, 556, 1000, 333, 1000, 500, 333, 944, 0, 500, 667, // 0x90
	278, 333, 556, 556, 556, 556, 260, 556, 333, 737, 370, 556, 584, 333, 737, 333, // 0xA0
	400, 584, 333, 333, 333, 556, 537, 278, 333, 333, 365, 556, 834, 834, 834, 611, // 0xB0
	667, 667, 667, 667, 667, 667, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278, // 0xC0
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611, // 0xD0
	556, 556, 556, 556, 556, 556, 889, 500, 556, 556, 556, 556, 278, 278, 278, 278, // 0xE0
	556, 556, 556, 556, 556, 556, 556, 584, 611, 556, 556, 556, 556, 500, 556, 500, // 0xF0
}

// helveticaBoldWidths is the advance-width table for Helvetica-Bold.
var helveticaBoldWidths = [256]uint16{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x00
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, // 0x10
	278, 333, 474, 556, 556, 889, 722, 238, 333, 333, 389, 584, 278, 333, 278, 278, // 0x20
	556, 556, 556, 556, 556, 556, 556, 556, 556, 556, 333, 333, 584, 584, 584, 611, // 0x30
	975, 722, 722, 722, 722, 667, 611, 778, 722, 278, 556, 722, 611, 833, 722, 778, // 0x40
	667, 778, 722, 667, 611, 722, 667, 944, 667, 667, 611, 333, 278, 333, 584, 556, // 0x50
	333, 556, 611, 556, 611, 556, 333, 611, 611, 278, 278, 556, 278, 889, 611, 611, // 0x60
	611, 611, 389, 556, 333, 611, 556, 778, 556, 556, 500, 389, 280, 389, 584, 0, // 0x70
	556, 0, 278, 556, 500, 1000, 556, 556, 333, 1000, 667, 333, 1000, 0, 611, 0, // 0x80
	0, 278, 278, 500, 500, 350, 556, 1000, 333, 1000, 556, 333, 944, 0, 556, 667, // 0x90
	278, 333, 556, 556, 556, 556, 280, 556, 333, 737, 370, 556, 584, 333, 737, 333, // 0xA0
	400, 584, 333, 333, 333, 611, 556, 278, 333, 333, 365, 556, 834, 834, 834, 611, // 0xB0
	722, 722, 722, 722, 722, 722, 1000, 722, 667, 667, 667, 667, 278, 278, 278, 278, // 0xC0
	722, 722, 778, 778, 778, 778, 778, 584, 778, 722, 722, 722, 722, 667, 667, 611, // 0xD0
	556, 556, 556, 556, 556, 556, 889, 556, 556, 556, 556, 556, 278, 278, 278, 278, // 0xE0
	611, 611, 611, 611, 611, 611, 611, 584, 611, 611, 611, 611, 611, 556, 611, 556, // 0xF0
}

// widthsFor returns the metric table backing a font.
func widthsFor(f Font) *[256]uint16 {
	if f == Bold {
		return &helveticaBoldWidths
	}
	return &helveticaWidths
}

// baseFontName is the PostScript name written into the font dictionary.
func baseFontName(f Font) string {
	switch f {
	case Bold:
		return "Helvetica-Bold"
	case Italic:
		return "Helvetica-Oblique"
	default:
		return "Helvetica"
	}
}
