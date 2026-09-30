package ditto

import (
	"strings"
	"unicode/utf8"
)

// Normalize is the shared front-end to shingling: it strips HTML tags, lowercases,
// and collapses runs of whitespace to single spaces. It is deterministic, and its
// behaviour is part of the fingerprint pipeline — any change to it must bump
// [PipelineVersion], because it shifts every fingerprint.
func Normalize(text string) string {
	return strings.Join(strings.Fields(strings.ToLower(stripTags(text))), " ")
}

// stripTags removes <...> spans, replacing each with a space so adjacent words
// don't fuse across a tag boundary. A '<' only opens a tag when it is followed by
// a tag-like byte ([A-Za-z/!?]); otherwise it is kept literally, so ordinary text
// such as "x < y", "5 < 3", or "<3" survives instead of being swallowed to the
// next '>' (or to end-of-input when there is none). It is intentionally simple —
// enough for HTML email/pages, not a conformant parser.
func stripTags(s string) string {
	if !strings.ContainsRune(s, '<') {
		return s
	}
	var b strings.Builder
	b.Grow(len(s))
	depth := 0
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		switch {
		case r == '<' && isTagStart(s[i+size:]):
			depth++
		case r == '>' && depth > 0:
			depth--
			b.WriteByte(' ')
		case depth > 0:
			// inside a tag: drop the rune
		default:
			b.WriteRune(r)
		}
		i += size
	}
	return b.String()
}

// isTagStart reports whether s begins with a byte that plausibly opens an HTML
// tag: a letter, or '/' / '!' / '?' for closing tags, comments, and declarations.
// A '<' not followed by one of these is ordinary text, not markup.
func isTagStart(s string) bool {
	if s == "" {
		return false
	}
	c := s[0]
	return c == '/' || c == '!' || c == '?' ||
		('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z')
}
