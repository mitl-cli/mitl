package digest

import (
	"bytes"
	"testing"
)

func TestNormalizer_LineEndingsAndBOM(t *testing.T) {
	n := NewNormalizer()
	// CRLF should normalize consistently
	lf, err := n.Normalize([]byte("a\r\nb\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	lf2, err := n.Normalize([]byte("a\nb\n"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(lf, lf2) {
		t.Errorf("expected normalized outputs to match; got %q vs %q", string(lf), string(lf2))
	}
	// BOM removal
	withBOM := []byte{0xEF, 0xBB, 0xBF, 'x'}
	out, err := n.Normalize(withBOM)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "x" {
		t.Errorf("expected BOM to be removed; got %q", string(out))
	}
}
