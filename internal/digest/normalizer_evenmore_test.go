package digest

import (
	"testing"
)

func TestNormalizer_StringHelpersAndInvalidUTF8(t *testing.T) {
	n := NewNormalizer()
	s, err := n.NormalizeString("a\r\nb\n")
	if err != nil || s == "" {
		t.Fatalf("unexpected: %q %v", s, err)
	}
	s2, err := DefaultNormalizeString("x\r\n")
	if err != nil || s2 == "" {
		t.Fatalf("unexpected default normalize string: %q %v", s2, err)
	}
	// Invalid UTF-8
	bad := []byte{0xff, 0xfe, 0xff}
	_, err = NewNormalizerWithOptions(false, false, true).Normalize(bad)
	if err == nil {
		t.Fatalf("expected error for invalid utf-8")
	}
}
