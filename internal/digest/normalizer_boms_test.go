package digest

import "testing"

func TestNormalizer_UTF16AndUTF32BOMs(t *testing.T) {
	n := NewNormalizer()
	// UTF-16LE FF FE
	in16le := []byte{0xFF, 0xFE, 'x', 0x00}
	if out, err := n.Normalize(in16le); err != nil || len(out) >= len(in16le) {
		t.Fatalf("utf16le: %v len=%d", err, len(out))
	}
	// UTF-16BE FE FF
	in16be := []byte{0xFE, 0xFF, 0x00, 'x'}
	if out, err := n.Normalize(in16be); err != nil || len(out) >= len(in16be) {
		t.Fatalf("utf16be: %v len=%d", err, len(out))
	}
	// UTF-32LE FF FE 00 00
	in32le := []byte{0xFF, 0xFE, 0x00, 0x00, 'x'}
	if out, err := n.Normalize(in32le); err != nil || len(out) >= len(in32le) {
		t.Fatalf("utf32le: %v len=%d", err, len(out))
	}
	// UTF-32BE 00 00 FE FF
	in32be := []byte{0x00, 0x00, 0xFE, 0xFF, 'x'}
	if out, err := n.Normalize(in32be); err != nil || len(out) >= len(in32be) {
		t.Fatalf("utf32be: %v len=%d", err, len(out))
	}
}

func TestDefaultNormalize(t *testing.T) {
	out, err := DefaultNormalize([]byte("a\r\nb\r"))
	if err != nil || len(out) == 0 {
		t.Fatalf("default normalize: %v", err)
	}
}
