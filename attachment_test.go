package email

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWriteUniqueAttachment_NoCollision: a free name is written unchanged and
// returned as-is.
func TestWriteUniqueAttachment_NoCollision(t *testing.T) {
	dir := t.TempDir()
	got, err := writeUniqueAttachment(dir, "invoice.pdf", []byte("A"), 0o600)
	if err != nil {
		t.Fatalf("writeUniqueAttachment: %v", err)
	}
	want := filepath.Join(dir, "invoice.pdf")
	if got != want {
		t.Errorf("path = %q, want %q", got, want)
	}
	if b, _ := os.ReadFile(got); string(b) != "A" {
		t.Errorf("content = %q, want %q", b, "A")
	}
}

// TestWriteUniqueAttachment_Numbering walks a sequence of same-named writes and
// asserts the OneDrive-style suffix progression, that every distinct file is
// preserved (no clobber), and that the returned paths are the actual numbered
// ones — for names with and without an extension.
func TestWriteUniqueAttachment_Numbering(t *testing.T) {
	tests := []struct {
		name      string
		input     string   // attachment filename, written repeatedly
		wantBases []string // expected basenames for successive writes
	}{
		{
			name:      "extension",
			input:     "invoice.pdf",
			wantBases: []string{"invoice.pdf", "invoice (2).pdf", "invoice (3).pdf"},
		},
		{
			name:      "no extension",
			input:     "README",
			wantBases: []string{"README", "README (2)", "README (3)"},
		},
		{
			name:      "multi-dot",
			input:     "archive.tar.gz",
			wantBases: []string{"archive.tar.gz", "archive.tar (2).gz", "archive.tar (3).gz"},
		},
		{
			// .eml is the filing path's name shape: two emails with the same
			// subject in one matter must number, not clobber. (G2)
			name:      "eml",
			input:     "RE_ BAR-557 costs.eml",
			wantBases: []string{"RE_ BAR-557 costs.eml", "RE_ BAR-557 costs (2).eml", "RE_ BAR-557 costs (3).eml"},
		},
		{
			name:      "dotfile",
			input:     ".env",
			wantBases: []string{".env", " (2).env", " (3).env"}, // filepath.Ext(".env") == ".env"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for i, wantBase := range tt.wantBases {
				content := []byte{byte('0' + i)} // distinct bytes per write
				got, err := writeUniqueAttachment(dir, tt.input, content, 0o600)
				if err != nil {
					t.Fatalf("write %d: %v", i, err)
				}
				want := filepath.Join(dir, wantBase)
				if got != want {
					t.Fatalf("write %d path = %q, want %q", i, got, want)
				}
				// The returned path must hold *this* write's bytes, proving the
				// earlier files were not overwritten.
				if b, _ := os.ReadFile(got); string(b) != string(content) {
					t.Errorf("write %d content = %q, want %q", i, b, content)
				}
			}
			// All distinct files must coexist on disk.
			for i, wantBase := range tt.wantBases {
				p := filepath.Join(dir, wantBase)
				b, err := os.ReadFile(p)
				if err != nil {
					t.Errorf("expected %q to exist: %v", p, err)
					continue
				}
				if string(b) != string([]byte{byte('0' + i)}) {
					t.Errorf("file %q content = %q, want %q", p, b, []byte{byte('0' + i)})
				}
			}
		})
	}
}

// TestWriteUniqueAttachment_StripsDir: a name carrying a path component is
// reduced to its base so attachments cannot escape destDir.
func TestWriteUniqueAttachment_StripsDir(t *testing.T) {
	dir := t.TempDir()
	got, err := writeUniqueAttachment(dir, "../../etc/passwd", []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("writeUniqueAttachment: %v", err)
	}
	want := filepath.Join(dir, "passwd")
	if got != want {
		t.Errorf("path = %q, want %q (must stay inside destDir)", got, want)
	}
}

// TestWriteUniqueAttachment_Mode: the file is created with the requested perm.
func TestWriteUniqueAttachment_Mode(t *testing.T) {
	dir := t.TempDir()
	got, err := writeUniqueAttachment(dir, "f.bin", []byte("x"), 0o600)
	if err != nil {
		t.Fatalf("writeUniqueAttachment: %v", err)
	}
	info, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("mode = %o, want 0600", perm)
	}
}
