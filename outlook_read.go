// outlook_read.go - Outlook 365 (Microsoft Graph) implementation of the
// MailboxProvider read/management operations. The send path lives in
// outlook.go; this file adds list/read/search/move/attachments/flags/delete/
// folders. All builder and query-parameter type names here were verified
// against the generated msgraph-sdk-go source (stable across v1.59.0 the
// standalone pin and the workspace's newer pin).
package email

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

// user returns the configured mailbox to operate on, or an error if none was
// set. Read operations have no message to key off (unlike Send, which uses
// msg.From), so a concrete UserID is required.
func (o *outlookProvider) user() (string, error) {
	if o.config == nil || o.config.UserID == "" {
		return "", fmt.Errorf("outlook: OutlookConfig.UserID is required for mailbox operations")
	}
	return o.config.UserID, nil
}

// strptr / i32ptr are local pointer helpers for the SDK's optional fields.
func strptr(s string) *string { return &s }
func i32ptr(i int32) *int32   { return &i }

// summarySelect is the field set fetched for message headers.
var summarySelect = []string{
	"id", "subject", "from", "receivedDateTime", "hasAttachments",
	"isRead", "categories",
}

// List returns message headers from a folder (default inbox), newest first.
func (o *outlookProvider) List(ctx context.Context, opts ListOptions) ([]Summary, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}

	folder := opts.Folder
	if folder == "" {
		folder = "inbox"
	}

	cfg := &graphusers.ItemMailFoldersItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphusers.ItemMailFoldersItemMessagesRequestBuilderGetQueryParameters{
			Select:  summarySelect,
			Orderby: []string{"receivedDateTime desc"},
			Filter:  outlookFilter(opts),
			Top:     outlookTop(opts),
		},
	}

	resp, err := o.client.Users().ByUserId(uid).
		MailFolders().ByMailFolderId(folder).
		Messages().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook list %s/%s: %w", uid, folder, err)
	}

	var out []Summary
	for _, m := range resp.GetValue() {
		out = append(out, outlookSummary(m))
		if opts.Limit > 0 && len(out) >= opts.Limit {
			return out, nil
		}
	}
	// Follow @odata.nextLink until the limit is reached or pages are exhausted.
	next := resp.GetOdataNextLink()
	for next != nil && *next != "" {
		page, err := o.client.Users().ByUserId(uid).
			MailFolders().ByMailFolderId(folder).
			Messages().WithUrl(*next).Get(ctx, cfg)
		if err != nil {
			return out, fmt.Errorf("outlook list %s/%s (page): %w", uid, folder, err)
		}
		for _, m := range page.GetValue() {
			out = append(out, outlookSummary(m))
			if opts.Limit > 0 && len(out) >= opts.Limit {
				return out, nil
			}
		}
		next = page.GetOdataNextLink()
	}
	return out, nil
}

// Read returns one message including its body.
func (o *outlookProvider) Read(ctx context.Context, id string) (*FullMessage, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	cfg := &graphusers.ItemMessagesMessageItemRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphusers.ItemMessagesMessageItemRequestBuilderGetQueryParameters{
			Select: []string{
				"id", "subject", "from", "toRecipients", "ccRecipients",
				"receivedDateTime", "hasAttachments", "isRead", "categories", "body",
			},
		},
	}
	m, err := o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook read %s/%s: %w", uid, id, err)
	}

	full := &FullMessage{
		Summary: outlookSummary(m),
		To:      outlookRecipientAddrs(m.GetToRecipients()),
		Cc:      outlookRecipientAddrs(m.GetCcRecipients()),
	}
	if body := m.GetBody(); body != nil {
		content := derefStr(body.GetContent())
		if ct := body.GetContentType(); ct != nil && *ct == graphmodels.HTML_BODYTYPE {
			full.BodyHTML = content
		} else {
			full.BodyText = content
		}
	}
	return full, nil
}

// Search runs a Graph $search over the mailbox. Graph $search for messages
// does not require the ConsistencyLevel header (that applies to directory
// objects only) and cannot be combined with $orderby — results come back
// ranked by relevance/date.
func (o *outlookProvider) Search(ctx context.Context, query string, opts ListOptions) ([]Summary, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	// $search requires the term wrapped in double quotes.
	quoted := fmt.Sprintf("%q", query)
	cfg := &graphusers.ItemMessagesRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphusers.ItemMessagesRequestBuilderGetQueryParameters{
			Search: &quoted,
			Select: summarySelect,
			Top:    outlookTop(opts),
		},
	}
	resp, err := o.client.Users().ByUserId(uid).Messages().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook search %s %q: %w", uid, query, err)
	}
	var out []Summary
	for _, m := range resp.GetValue() {
		out = append(out, outlookSummary(m))
		if opts.Limit > 0 && len(out) >= opts.Limit {
			break
		}
	}
	return out, nil
}

// Move relocates a message to the destination folder. dest may be a well-known
// folder name (e.g. "archive", "deleteditems") or a folder id.
func (o *outlookProvider) Move(ctx context.Context, id, dest string) error {
	uid, err := o.user()
	if err != nil {
		return err
	}
	body := graphusers.NewItemMessagesItemMovePostRequestBody()
	body.SetDestinationId(strptr(dest))
	_, err = o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Move().Post(ctx, body, nil)
	if err != nil {
		return fmt.Errorf("outlook move %s/%s -> %s: %w", uid, id, dest, err)
	}
	return nil
}

// ListAttachments returns metadata for a message's file attachments.
func (o *outlookProvider) ListAttachments(ctx context.Context, id string) ([]AttachmentMeta, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	resp, err := o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Attachments().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("outlook attachments %s/%s: %w", uid, id, err)
	}
	var out []AttachmentMeta
	for _, att := range resp.GetValue() {
		fa, ok := att.(graphmodels.FileAttachmentable)
		if !ok {
			continue // skip item/reference attachments
		}
		out = append(out, AttachmentMeta{
			ID:       derefStr(fa.GetId()),
			Filename: derefStr(fa.GetName()),
			MimeType: derefStr(fa.GetContentType()),
			Size:     int64(derefI32(fa.GetSize())),
		})
	}
	return out, nil
}

// SaveAttachments writes a message's file attachments into destDir.
func (o *outlookProvider) SaveAttachments(ctx context.Context, id, destDir string) ([]string, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	resp, err := o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Attachments().Get(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("outlook attachments %s/%s: %w", uid, id, err)
	}
	if err := os.MkdirAll(destDir, 0o750); err != nil {
		return nil, fmt.Errorf("outlook save attachments: mkdir %q: %w", destDir, err)
	}
	var saved []string
	for _, att := range resp.GetValue() {
		fa, ok := att.(graphmodels.FileAttachmentable)
		if !ok {
			continue
		}
		name := derefStr(fa.GetName())
		if name == "" {
			name = derefStr(fa.GetId())
		}
		out, err := writeUniqueAttachment(destDir, name, fa.GetContentBytes(), 0o600)
		if err != nil {
			return saved, fmt.Errorf("outlook save attachment %q: %w", name, err)
		}
		saved = append(saved, out)
	}
	return saved, nil
}

// maxAttachmentCollisions caps the OneDrive-style auto-numbering search so a
// pathological directory (or a races-with-another-writer scenario) can never
// spin forever. 4096 distinct collisions for one filename in one destDir is far
// beyond anything a real mailbox produces.
const maxAttachmentCollisions = 4096

// writeUniqueAttachment writes data into destDir under a collision-free name
// derived from the attachment's filename, and returns the path actually written.
//
// It never overwrites an existing file. If destDir/<name> is free it is used
// unchanged; otherwise the numeric suffix " (2)", " (3)", ... is inserted before
// the extension (OneDrive style): "invoice.pdf" -> "invoice (2).pdf",
// "README" -> "README (2)". The extension boundary is filepath.Ext's, so dotted
// names like "archive.tar.gz" number as "archive.tar (2).gz".
//
// The create is atomic: each candidate is opened with O_CREATE|O_EXCL, so the
// existence check and the create are a single syscall. This closes the TOCTOU
// window between "does it exist?" and "write it" — including the same-process
// repeats dl triggers when it re-runs triage. On EEXIST we advance to the next
// number and retry; any other open error is returned.
func writeUniqueAttachment(destDir, name string, data []byte, perm os.FileMode) (string, error) {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	stem := base[:len(base)-len(ext)]

	for i := 1; i <= maxAttachmentCollisions; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%s (%d)%s", stem, i, ext)
		}
		out := filepath.Join(destDir, candidate)

		f, err := os.OpenFile(out, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
		if err != nil {
			if os.IsExist(err) {
				continue // taken — try the next number
			}
			return "", err
		}
		if _, werr := f.Write(data); werr != nil {
			f.Close()
			return "", werr
		}
		if cerr := f.Close(); cerr != nil {
			return "", cerr
		}
		return out, nil
	}
	return "", fmt.Errorf("could not find a free name for %q in %q after %d attempts",
		base, destDir, maxAttachmentCollisions)
}

// MarkRead sets a message's read state.
func (o *outlookProvider) MarkRead(ctx context.Context, id string, read bool) error {
	uid, err := o.user()
	if err != nil {
		return err
	}
	patch := graphmodels.NewMessage()
	patch.SetIsRead(&read)
	_, err = o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Patch(ctx, patch, nil)
	if err != nil {
		return fmt.Errorf("outlook markread %s/%s: %w", uid, id, err)
	}
	return nil
}

// SetLabels replaces a message's Outlook categories.
func (o *outlookProvider) SetLabels(ctx context.Context, id string, labels []string) error {
	uid, err := o.user()
	if err != nil {
		return err
	}
	patch := graphmodels.NewMessage()
	patch.SetCategories(labels)
	_, err = o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Patch(ctx, patch, nil)
	if err != nil {
		return fmt.Errorf("outlook setlabels %s/%s: %w", uid, id, err)
	}
	return nil
}

// Delete removes a message. Graph DELETE is a soft delete (moves to Deleted
// Items); the v1.0 SDK has no permanent-delete action, so permanent=true is
// reported as unsupported rather than silently soft-deleting.
func (o *outlookProvider) Delete(ctx context.Context, id string, permanent bool) error {
	uid, err := o.user()
	if err != nil {
		return err
	}
	if permanent {
		return fmt.Errorf("outlook delete %s/%s permanent: %w (Graph v1.0 has no permanentDelete; use permanent=false to move to Deleted Items)", uid, id, ErrUnsupported)
	}
	if err := o.client.Users().ByUserId(uid).Messages().ByMessageId(id).Delete(ctx, nil); err != nil {
		return fmt.Errorf("outlook delete %s/%s: %w", uid, id, err)
	}
	return nil
}

// ListFolders returns the mailbox's mail folders.
func (o *outlookProvider) ListFolders(ctx context.Context) ([]Folder, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	cfg := &graphusers.ItemMailFoldersRequestBuilderGetRequestConfiguration{
		QueryParameters: &graphusers.ItemMailFoldersRequestBuilderGetQueryParameters{
			Select: []string{"id", "displayName", "unreadItemCount"},
			Top:    i32ptr(100),
		},
	}
	resp, err := o.client.Users().ByUserId(uid).MailFolders().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook listfolders %s: %w", uid, err)
	}
	var out []Folder
	for _, f := range resp.GetValue() {
		out = append(out, Folder{
			ID:     derefStr(f.GetId()),
			Name:   derefStr(f.GetDisplayName()),
			Unread: int(derefI32(f.GetUnreadItemCount())),
		})
	}
	return out, nil
}

// --- conversion helpers -----------------------------------------------------

// outlookFilter builds a $filter string from the unread/since options.
func outlookFilter(opts ListOptions) *string {
	var parts []string
	if opts.UnreadOnly {
		parts = append(parts, "isRead eq false")
	}
	if !opts.Since.IsZero() {
		parts = append(parts, "receivedDateTime ge "+opts.Since.UTC().Format(time.RFC3339))
	}
	if len(parts) == 0 {
		return nil
	}
	return strptr(strings.Join(parts, " and "))
}

// outlookTop maps a positive Limit to a $top page size.
func outlookTop(opts ListOptions) *int32 {
	if opts.Limit > 0 {
		n := opts.Limit
		if n > 1000 { // Graph caps page size; clamp to avoid int32 overflow
			n = 1000
		}
		return i32ptr(int32(n))
	}
	return nil
}

// outlookSummary converts a Graph message header to a Summary.
func outlookSummary(m graphmodels.Messageable) Summary {
	s := Summary{
		ID:             derefStr(m.GetId()),
		Subject:        derefStr(m.GetSubject()),
		HasAttachments: derefBool(m.GetHasAttachments()),
		Unread:         !derefBool(m.GetIsRead()),
		Labels:         m.GetCategories(),
	}
	if from := m.GetFrom(); from != nil {
		if ea := from.GetEmailAddress(); ea != nil {
			s.From = derefStr(ea.GetAddress())
		}
	}
	if rd := m.GetReceivedDateTime(); rd != nil {
		s.Received = *rd
	}
	return s
}

// outlookRecipientAddrs extracts addresses from a recipient slice.
func outlookRecipientAddrs(rs []graphmodels.Recipientable) []string {
	var out []string
	for _, r := range rs {
		if r == nil {
			continue
		}
		if ea := r.GetEmailAddress(); ea != nil {
			if a := derefStr(ea.GetAddress()); a != "" {
				out = append(out, a)
			}
		}
	}
	return out
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefBool(b *bool) bool {
	return b != nil && *b
}

func derefI32(i *int32) int32 {
	if i == nil {
		return 0
	}
	return *i
}
