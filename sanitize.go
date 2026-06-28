// sanitize.go - Filesystem-safety helpers shared by the provider .eml/raw-MIME
// filing paths. Pure functions, no network, no provider knowledge: they turn a
// message subject into a single safe path element with a guaranteed ".eml"
// suffix, so the consumer (e.g. dl) can pass a raw subject and the library owns
// "make this a safe .eml name". Sits beside writeUniqueAttachment in spirit;
// both providers reuse it.
package email

import (
	"strings"
	"unicode/utf8"
)

// emlStemMaxBytes caps the sanitized filename stem. 180 bytes leaves headroom
// under the local 255-byte name limit for the OneDrive-style numeric suffix
// (" (2)") plus the ".eml" extension. The rclone/OneDrive PATH cap (not the
// local name limit) is the real ceiling for deep matter folders; a write that
// still overflows is surfaced as an error by writeUniqueAttachment, never
// silently truncated.
const emlStemMaxBytes = 180

// reservedFilenameRunes are characters illegal or unsafe in a single path
// element across Windows/OneDrive and POSIX. Each is replaced with "_".
const reservedFilenameRunes = `/\:*?"<>|`

// ensureEMLSuffix sanitizes a message subject into a safe single-path-element
// filename and guarantees a single ".eml" extension:
//   - path separators and reserved/control chars -> "_"
//   - leading/trailing dots and spaces trimmed (Windows/OneDrive safety)
//   - empty result -> "message"
//   - stem capped at emlStemMaxBytes on a UTF-8 boundary (no split rune)
//   - a pre-existing ".eml" (case-insensitive) is not doubled
//
// It does NOT touch the filesystem and does NOT resolve collisions — pass its
// output to writeUniqueAttachment, which numbers duplicates atomically.
func ensureEMLSuffix(subject string) string {
	// 1. Replace reserved runes and control chars with "_".
	var b strings.Builder
	b.Grow(len(subject))
	for _, r := range subject {
		switch {
		case r < 0x20, r == 0x7f: // control chars (incl. tab/newline) and DEL
			b.WriteByte('_')
		case strings.ContainsRune(reservedFilenameRunes, r):
			b.WriteByte('_')
		default:
			b.WriteRune(r)
		}
	}
	s := b.String()

	// 2. Trim leading/trailing dots and spaces (Windows/OneDrive reject trailing
	//    dots/spaces; a leading dot would make a hidden/dotfile).
	s = strings.Trim(s, " .")

	// 3. Strip a pre-existing .eml so we can cap the stem and re-append exactly one.
	if len(s) >= 4 && strings.EqualFold(s[len(s)-4:], ".eml") {
		s = s[:len(s)-4]
		// Re-trim in case stripping exposed trailing space/dot ("foo .eml").
		s = strings.Trim(s, " .")
	}

	// 4. Empty after sanitization -> a stable default stem.
	if s == "" {
		s = "message"
	}

	// 5. Cap the stem on a rune boundary so we never split a multibyte char.
	if len(s) > emlStemMaxBytes {
		s = truncateUTF8(s, emlStemMaxBytes)
		// Capping may have exposed a trailing space/dot; re-trim, and re-default
		// if the cap left us empty (pathological all-trimmed input).
		s = strings.Trim(s, " .")
		if s == "" {
			s = "message"
		}
	}

	return s + ".eml"
}

// truncateUTF8 returns s truncated to at most n bytes, never splitting a rune.
func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	// Back off to the last complete rune boundary at or before n.
	for n > 0 && !utf8.RuneStart(s[n]) {
		n--
	}
	return s[:n]
}
