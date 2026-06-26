package email

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

// mockMailbox implements MailboxProvider for testing the Client wrappers and
// the send-only fallback path.
type mockMailbox struct {
	mockProvider
	listOpts ListOptions
	summ     []Summary
	full     *FullMessage
	moved    [2]string // {id, dest}
	err      error
}

func (m *mockMailbox) List(_ context.Context, opts ListOptions) ([]Summary, error) {
	m.listOpts = opts
	return m.summ, m.err
}
func (m *mockMailbox) Read(_ context.Context, _ string) (*FullMessage, error) {
	return m.full, m.err
}
func (m *mockMailbox) Search(_ context.Context, _ string, opts ListOptions) ([]Summary, error) {
	m.listOpts = opts
	return m.summ, m.err
}
func (m *mockMailbox) Move(_ context.Context, id, dest string) error {
	m.moved = [2]string{id, dest}
	return m.err
}
func (m *mockMailbox) ListAttachments(_ context.Context, _ string) ([]AttachmentMeta, error) {
	return nil, m.err
}
func (m *mockMailbox) SaveAttachments(_ context.Context, _, _ string) ([]string, error) {
	return nil, m.err
}
func (m *mockMailbox) MarkRead(_ context.Context, _ string, _ bool) error      { return m.err }
func (m *mockMailbox) SetLabels(_ context.Context, _ string, _ []string) error { return m.err }
func (m *mockMailbox) Delete(_ context.Context, _ string, _ bool) error        { return m.err }
func (m *mockMailbox) ListFolders(_ context.Context) ([]Folder, error)         { return nil, m.err }

func TestClientMailboxUnsupported(t *testing.T) {
	// A plain send-only provider is not a MailboxProvider; the Client mailbox
	// methods must return ErrUnsupported rather than panic.
	c := &Client{provider: &mockProvider{}}

	if _, err := c.List(ListOptions{}); !errors.Is(err, ErrUnsupported) {
		t.Errorf("List on send-only provider: got %v, want ErrUnsupported", err)
	}
	if _, err := c.Read("id"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Read on send-only provider: got %v, want ErrUnsupported", err)
	}
	if err := c.Move("id", "Archive"); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Move on send-only provider: got %v, want ErrUnsupported", err)
	}
	if err := c.Delete("id", false); !errors.Is(err, ErrUnsupported) {
		t.Errorf("Delete on send-only provider: got %v, want ErrUnsupported", err)
	}
}

func TestClientMailboxDelegation(t *testing.T) {
	want := []Summary{{ID: "1", Subject: "hi"}}
	mb := &mockMailbox{summ: want}
	c := &Client{provider: mb}

	got, err := c.List(ListOptions{Folder: "inbox", UnreadOnly: true, Limit: 5})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("List returned %v, want %v", got, want)
	}
	if !mb.listOpts.UnreadOnly || mb.listOpts.Folder != "inbox" || mb.listOpts.Limit != 5 {
		t.Errorf("List opts not passed through: %+v", mb.listOpts)
	}

	if err := c.Move("m1", "Clients/WOO-402"); err != nil {
		t.Fatalf("Move: %v", err)
	}
	if mb.moved != [2]string{"m1", "Clients/WOO-402"} {
		t.Errorf("Move args = %v", mb.moved)
	}
}

func TestGmailQuery(t *testing.T) {
	since := time.Unix(1_700_000_000, 0)
	tests := []struct {
		name string
		opts ListOptions
		want string
	}{
		{"empty", ListOptions{}, ""},
		{"unread", ListOptions{UnreadOnly: true}, "is:unread"},
		{"since", ListOptions{Since: since}, "after:1700000000"},
		{"both", ListOptions{UnreadOnly: true, Since: since}, "is:unread after:1700000000"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := gmailQuery(tt.opts); got != tt.want {
				t.Errorf("gmailQuery(%+v) = %q, want %q", tt.opts, got, tt.want)
			}
		})
	}
}

func TestOutlookFilter(t *testing.T) {
	since := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		opts ListOptions
		want string // "" means nil
	}{
		{"none", ListOptions{}, ""},
		{"unread", ListOptions{UnreadOnly: true}, "isRead eq false"},
		{"since", ListOptions{Since: since}, "receivedDateTime ge 2026-06-01T09:00:00Z"},
		{"both", ListOptions{UnreadOnly: true, Since: since}, "isRead eq false and receivedDateTime ge 2026-06-01T09:00:00Z"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := outlookFilter(tt.opts)
			if tt.want == "" {
				if got != nil {
					t.Errorf("outlookFilter(%+v) = %q, want nil", tt.opts, *got)
				}
				return
			}
			if got == nil || *got != tt.want {
				t.Errorf("outlookFilter(%+v) = %v, want %q", tt.opts, got, tt.want)
			}
		})
	}
}

func TestParseAddr(t *testing.T) {
	tests := []struct{ in, want string }{
		{"m@x.com", "m@x.com"},
		{"Marios <m@x.com>", "m@x.com"},
		{"  spaced <a@b.co>  ", "a@b.co"},
		{`"Doe, John" <john@x.com>`, "john@x.com"},
	}
	for _, tt := range tests {
		if got := parseAddr(tt.in); got != tt.want {
			t.Errorf("parseAddr(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestSplitAddrs(t *testing.T) {
	got := splitAddrs("a@x.com, Bob <b@y.com>")
	want := []string{"a@x.com", "b@y.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitAddrs = %v, want %v", got, want)
	}
	if splitAddrs("  ") != nil {
		t.Error("splitAddrs(blank) should be nil")
	}
}

func TestDecodeMIMEHeader(t *testing.T) {
	// RFC 2047 encoded-word for "ŽpČ" (Croatian diacritics) round-trips to UTF-8.
	enc := "=?UTF-8?B?xb1wxIw=?="
	if got := decodeMIMEHeader(enc); got != "ŽpČ" {
		t.Errorf("decodeMIMEHeader(%q) = %q, want %q", enc, got, "ŽpČ")
	}
	// Plain ASCII is returned unchanged.
	if got := decodeMIMEHeader("Plain Subject"); got != "Plain Subject" {
		t.Errorf("decodeMIMEHeader plain = %q", got)
	}
}

func TestGmailSystemLabelsMapping(t *testing.T) {
	for _, l := range []string{"INBOX", "SENT", "UNREAD", "TRASH"} {
		if !gmailSystemLabels[l] {
			t.Errorf("expected %q to be a recognised system label", l)
		}
	}
	if gmailSystemLabels["Clients/WOO-402"] {
		t.Error("user label wrongly treated as system label")
	}
}
