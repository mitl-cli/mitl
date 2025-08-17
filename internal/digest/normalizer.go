package digest

import (
	"bytes"
	"fmt"
	"strings"
	"unicode/utf8"
)

// Normalizer provides content normalization for deterministic hashing.
// It handles line ending normalization, BOM removal, and encoding validation
// to ensure consistent digest generation across different platforms and editors.
type Normalizer struct {
	// stripBOM removes byte order mark from content
	stripBOM bool
	// normalizeLineEndings converts CRLF to LF
	normalizeLineEndings bool
	// validateUTF8 ensures content is valid UTF-8
	validateUTF8 bool
}

// NewNormalizer creates a new normalizer with default settings.
// Default configuration:
// - Strips BOM markers
// - Normalizes line endings (CRLF → LF)
// - Validates UTF-8 encoding
func NewNormalizer() *Normalizer {
	return &Normalizer{
		stripBOM:             true,
		normalizeLineEndings: true,
		validateUTF8:         true,
	}
}

// NewNormalizerWithOptions creates a normalizer with custom settings.
func NewNormalizerWithOptions(stripBOM, normalizeLineEndings, validateUTF8 bool) *Normalizer {
	return &Normalizer{
		stripBOM:             stripBOM,
		normalizeLineEndings: normalizeLineEndings,
		validateUTF8:         validateUTF8,
	}
}

// Normalize applies all enabled normalizations to the input content.
// Returns normalized content and any validation errors.
func (n *Normalizer) Normalize(content []byte) ([]byte, error) {
	result := content

	// Strip BOM if enabled
	if n.stripBOM {
		result = n.stripByteOrderMark(result)
	}

	// Validate UTF-8 if enabled
	if n.validateUTF8 {
		if !utf8.Valid(result) {
			return nil, fmt.Errorf("content is not valid UTF-8")
		}
	}

	// Normalize line endings if enabled
	if n.normalizeLineEndings {
		result = n.normalizeLineEndingsImpl(result)
	}

	return result, nil
}

// stripByteOrderMark removes UTF-8, UTF-16BE, UTF-16LE, UTF-32BE, and UTF-32LE BOMs.
func (n *Normalizer) stripByteOrderMark(content []byte) []byte {
	// UTF-8 BOM: EF BB BF
	if len(content) >= 3 && bytes.Equal(content[:3], []byte{0xEF, 0xBB, 0xBF}) {
		return content[3:]
	}

	// UTF-16BE BOM: FE FF
	if len(content) >= 2 && bytes.Equal(content[:2], []byte{0xFE, 0xFF}) {
		return content[2:]
	}

	// UTF-16LE BOM: FF FE
	if len(content) >= 2 && bytes.Equal(content[:2], []byte{0xFF, 0xFE}) {
		return content[2:]
	}

	// UTF-32BE BOM: 00 00 FE FF
	if len(content) >= 4 && bytes.Equal(content[:4], []byte{0x00, 0x00, 0xFE, 0xFF}) {
		return content[4:]
	}

	// UTF-32LE BOM: FF FE 00 00
	if len(content) >= 4 && bytes.Equal(content[:4], []byte{0xFF, 0xFE, 0x00, 0x00}) {
		return content[4:]
	}

	return content
}

// normalizeLineEndingsImpl converts CRLF and CR to LF for consistent line endings.
// This ensures the same content produces the same hash regardless of the platform
// where the file was created or edited.
func (n *Normalizer) normalizeLineEndingsImpl(content []byte) []byte {
	// Convert string for easier manipulation
	str := string(content)

	// Replace CRLF with LF first to avoid double conversion
	str = strings.ReplaceAll(str, "\r\n", "\n")

	// Replace remaining CR with LF
	str = strings.ReplaceAll(str, "\r", "\n")

	return []byte(str)
}

// NormalizeString is a convenience method for normalizing string content.
func (n *Normalizer) NormalizeString(content string) (string, error) {
	normalized, err := n.Normalize([]byte(content))
	if err != nil {
		return "", fmt.Errorf("failed to normalize string content: %w", err)
	}
	return string(normalized), nil
}

// DefaultNormalize applies default normalization to content.
// This is a convenience function for the most common use case.
func DefaultNormalize(content []byte) ([]byte, error) {
	normalizer := NewNormalizer()
	return normalizer.Normalize(content)
}

// DefaultNormalizeString applies default normalization to string content.
func DefaultNormalizeString(content string) (string, error) {
	normalizer := NewNormalizer()
	return normalizer.NormalizeString(content)
}
