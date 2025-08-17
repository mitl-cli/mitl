package digest

import "testing"

func TestNormalizer_UTF8BOM(t *testing.T) {
	n := NewNormalizer()
	// UTF-8 BOM EF BB BF
	in := []byte{0xEF, 0xBB, 0xBF, 'x'}
	out, err := n.Normalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == len(in) {
		t.Fatalf("expected BOM removed")
	}
	// When stripBOM disabled, length should remain
	n2 := NewNormalizerWithOptions(false, true, true)
	out2, err := n2.Normalize(in)
	if err != nil {
		t.Fatal(err)
	}
	if len(out2) != len(in) {
		t.Fatalf("expected BOM retained when disabled")
	}
}
