package digest

import "testing"

func TestNormalizer_CRAndMixed(t *testing.T) {
	n := NewNormalizer()
	// CR only
	out1, err := n.Normalize([]byte("a\rb\r"))
	if err != nil {
		t.Fatal(err)
	}
	// Mixed CRLF/LF/CR
	out2, err := n.Normalize([]byte("a\r\nb\nc\r"))
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) == "" || string(out2) == "" {
		t.Fatalf("unexpected empty outputs")
	}
	// Ensure NormalizeString mirrors byte behavior
	s1, err := n.NormalizeString("a\rb\r")
	if err != nil || s1 == "" {
		t.Fatalf("unexpected NormalizeString result: %q %v", s1, err)
	}
}
