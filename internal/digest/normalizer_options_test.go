package digest

import "testing"

func TestNormalizer_DisableValidation(t *testing.T) {
	n := NewNormalizerWithOptions(true, true, false)
	// Invalid UTF-8 should pass when validation disabled
	bad := []byte{0xff, 0xfe, 0xff}
	if _, err := n.Normalize(bad); err != nil {
		t.Fatalf("did not expect error when validation disabled: %v", err)
	}
}
