package email

import (
	"context"
	"errors"
	"strings"
	"testing"
	"unicode/utf8"
)

// TestEnsureEMLSuffix pins the subject -> safe .eml filename rules: reserved/
// control chars replaced, dots/spaces trimmed, empty -> "message", a pre-existing
// .eml not doubled, and an over-long subject capped to a valid .eml on a rune
// boundary. Pure function, no network. (G1)
func TestEnsureEMLSuffix(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"simple", "Invoice March", "Invoice March.eml"},
		{"path separators (colon replaced)", "RE: Foo/Bar", "RE_ Foo_Bar.eml"},
		{"backslash and colon", `C:\path:thing`, "C__path_thing.eml"},
		{"reserved glob/quote", `a*b?c"d<e>f|g`, "a_b_c_d_e_f_g.eml"},
		{"empty", "", "message.eml"},
		// Reserved runes become "_"; underscores are a legitimate stem (only an
		// EMPTY post-sanitization result defaults to "message").
		{"only separators+spaces", "  ///  ", "___.eml"},
		{"trailing dots/spaces", "report... ", "report.eml"},
		{"leading dot not a dotfile", ".hidden", "hidden.eml"},
		{"already .eml not doubled", "thread.eml", "thread.eml"},
		{"already .EML case-insensitive", "thread.EML", "thread.eml"},
		{"eml with trailing space stem", "note .eml", "note.eml"},
		{"control chars stripped", "a\tb\nc", "a_b_c.eml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ensureEMLSuffix(tt.in)
			if got != tt.want {
				t.Errorf("ensureEMLSuffix(%q) = %q; want %q", tt.in, got, tt.want)
			}
			if !strings.HasSuffix(got, ".eml") {
				t.Errorf("result %q must end in .eml", got)
			}
			if !utf8.ValidString(got) {
				t.Errorf("result %q is not valid UTF-8", got)
			}
		})
	}
}

// TestEnsureEMLSuffix_LongSubjectCapped: a 300-rune subject yields a name whose
// stem is capped at emlStemMaxBytes, the whole name stays well under 255 bytes,
// it ends in exactly one .eml, and no rune was split (valid UTF-8). (G1)
func TestEnsureEMLSuffix_LongSubjectCapped(t *testing.T) {
	long := strings.Repeat("A", 300)
	got := ensureEMLSuffix(long)
	if !strings.HasSuffix(got, ".eml") {
		t.Fatalf("missing .eml suffix: %q", got)
	}
	stem := strings.TrimSuffix(got, ".eml")
	if len(stem) > emlStemMaxBytes {
		t.Errorf("stem = %d bytes; want <= %d", len(stem), emlStemMaxBytes)
	}
	if len(got) >= 255 {
		t.Errorf("full name = %d bytes; must stay under the 255-byte limit", len(got))
	}
	if !utf8.ValidString(got) {
		t.Errorf("result %q is not valid UTF-8 (rune split?)", got)
	}
}

// TestEnsureEMLSuffix_MultibyteCapNoSplit: capping a multibyte subject must not
// split a rune mid-byte. Use a subject of 3-byte runes long enough to force the
// cap. (G1)
func TestEnsureEMLSuffix_MultibyteCapNoSplit(t *testing.T) {
	// "あ" is 3 bytes; 100 of them = 300 bytes, well over emlStemMaxBytes.
	long := strings.Repeat("あ", 100)
	got := ensureEMLSuffix(long)
	if !utf8.ValidString(got) {
		t.Fatalf("multibyte cap split a rune: %q", got)
	}
	stem := strings.TrimSuffix(got, ".eml")
	if len(stem) > emlStemMaxBytes {
		t.Errorf("stem = %d bytes; want <= %d", len(stem), emlStemMaxBytes)
	}
}

// TestGmailSaveMessageRawUnsupported locks the YAGNI decision: Gmail's
// SaveMessageRaw is a stub returning ErrUnsupported (satisfies the compile-time
// MailboxProvider assertion without an untested base64url impl). (G3)
func TestGmailSaveMessageRawUnsupported(t *testing.T) {
	g := &gmailProvider{} // nil service is never touched; stub returns before use
	_, err := g.SaveMessageRaw(context.Background(), "id", t.TempDir(), "subject")
	if !errors.Is(err, ErrUnsupported) {
		t.Errorf("gmail SaveMessageRaw: got %v, want ErrUnsupported", err)
	}
}
