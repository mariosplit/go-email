// gmail_read.go - Gmail implementation of the MailboxProvider read/management
// operations. The send path lives in gmail.go; this file adds list/read/
// search/move/attachments/flags/delete/folders.
//
// Gmail differences baked in here, per the verified API reference:
//   - No folders, only labels. "Folder" == label; Move == relabel.
//   - Messages.List returns only {Id, ThreadId} stubs; each needs a Get.
//   - Bodies and attachments are base64URL WITHOUT padding -> RawURLEncoding.
//   - Modify/Trash/Labels need gmail.modify scope; permanent Delete needs the
//     full mail.google.com scope.
package email

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"time"

	gmail "google.golang.org/api/gmail/v1"
)

// gmailSystemLabels are the well-known system label ids whose id == name.
var gmailSystemLabels = map[string]bool{
	"INBOX": true, "SENT": true, "DRAFT": true, "TRASH": true, "SPAM": true,
	"UNREAD": true, "STARRED": true, "IMPORTANT": true,
	"CATEGORY_PERSONAL": true, "CATEGORY_SOCIAL": true, "CATEGORY_PROMOTIONS": true,
	"CATEGORY_UPDATES": true, "CATEGORY_FORUMS": true,
}

// List returns message headers from a label (default INBOX), newest first.
func (g *gmailProvider) List(ctx context.Context, opts ListOptions) ([]Summary, error) {
	label := opts.Folder
	if label == "" {
		label = "INBOX"
	}
	labelID, err := g.resolveLabelID(ctx, label)
	if err != nil {
		return nil, err
	}

	q := gmailQuery(opts)
	stubs, err := g.listStubs(ctx, q, []string{labelID}, opts.Limit)
	if err != nil {
		return nil, err
	}
	return g.hydrate(ctx, stubs)
}

// Search runs a Gmail search using Gmail's native operator syntax (e.g.
// "from:x subject:y has:attachment newer_than:7d"). It is the List endpoint
// with a query and no label restriction.
func (g *gmailProvider) Search(ctx context.Context, query string, opts ListOptions) ([]Summary, error) {
	// Combine the explicit query with any unread/since options.
	q := strings.TrimSpace(query + " " + gmailQuery(opts))
	stubs, err := g.listStubs(ctx, q, nil, opts.Limit)
	if err != nil {
		return nil, err
	}
	return g.hydrate(ctx, stubs)
}

// listStubs runs the List endpoint, paginating until the limit is reached.
// Each returned message carries only Id/ThreadId.
func (g *gmailProvider) listStubs(ctx context.Context, query string, labelIDs []string, limit int) ([]*gmail.Message, error) {
	var stubs []*gmail.Message
	pageToken := ""
	for {
		call := g.service.Users.Messages.List("me").Context(ctx)
		if query != "" {
			call = call.Q(query)
		}
		if len(labelIDs) > 0 {
			call = call.LabelIds(labelIDs...)
		}
		if limit > 0 {
			call = call.MaxResults(int64(limit))
		}
		if pageToken != "" {
			call = call.PageToken(pageToken)
		}
		resp, err := call.Do()
		if err != nil {
			return nil, fmt.Errorf("gmail list: %w", err)
		}
		stubs = append(stubs, resp.Messages...)
		if limit > 0 && len(stubs) >= limit {
			return stubs[:limit], nil
		}
		if resp.NextPageToken == "" {
			break
		}
		pageToken = resp.NextPageToken
	}
	return stubs, nil
}

// hydrate fetches metadata-format messages for each stub and converts them to
// Summaries.
func (g *gmailProvider) hydrate(ctx context.Context, stubs []*gmail.Message) ([]Summary, error) {
	out := make([]Summary, 0, len(stubs))
	for _, st := range stubs {
		m, err := g.service.Users.Messages.Get("me", st.Id).
			Format("metadata").
			MetadataHeaders("From", "Subject", "Date").
			Context(ctx).Do()
		if err != nil {
			return out, fmt.Errorf("gmail get %s: %w", st.Id, err)
		}
		out = append(out, gmailSummary(m))
	}
	return out, nil
}

// Read returns one message including its body.
func (g *gmailProvider) Read(ctx context.Context, id string) (*FullMessage, error) {
	m, err := g.service.Users.Messages.Get("me", id).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail read %s: %w", id, err)
	}
	full := &FullMessage{Summary: gmailSummary(m)}
	full.To = splitAddrs(gmailHeader(m, "To"))
	full.Cc = splitAddrs(gmailHeader(m, "Cc"))
	if m.Payload != nil {
		full.BodyText = gmailBodyByType(m.Payload, "text/plain")
		full.BodyHTML = gmailBodyByType(m.Payload, "text/html")
	}
	return full, nil
}

// Move adds the destination label and removes INBOX (archive-style move).
func (g *gmailProvider) Move(ctx context.Context, id, dest string) error {
	destID, err := g.resolveLabelIDCreating(ctx, dest)
	if err != nil {
		return err
	}
	req := &gmail.ModifyMessageRequest{
		AddLabelIds:    []string{destID},
		RemoveLabelIds: []string{"INBOX"},
	}
	if _, err := g.service.Users.Messages.Modify("me", id, req).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail move %s -> %s: %w", id, dest, err)
	}
	return nil
}

// ListAttachments returns metadata for a message's attachments.
func (g *gmailProvider) ListAttachments(ctx context.Context, id string) ([]AttachmentMeta, error) {
	m, err := g.service.Users.Messages.Get("me", id).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail attachments %s: %w", id, err)
	}
	var out []AttachmentMeta
	walkParts(m.Payload, func(p *gmail.MessagePart) {
		if p.Filename == "" || p.Body == nil {
			return
		}
		out = append(out, AttachmentMeta{
			ID:       p.Body.AttachmentId,
			Filename: p.Filename,
			MimeType: p.MimeType,
			Size:     p.Body.Size,
		})
	})
	return out, nil
}

// SaveAttachments writes a message's attachments into destDir.
func (g *gmailProvider) SaveAttachments(ctx context.Context, id, destDir string) ([]string, error) {
	m, err := g.service.Users.Messages.Get("me", id).Format("full").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail attachments %s: %w", id, err)
	}
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return nil, fmt.Errorf("gmail save attachments: mkdir %q: %w", destDir, err)
	}
	var saved []string
	var walkErr error
	walkParts(m.Payload, func(p *gmail.MessagePart) {
		if walkErr != nil || p.Filename == "" || p.Body == nil {
			return
		}
		data, err := g.attachmentData(ctx, id, p)
		if err != nil {
			walkErr = err
			return
		}
		out := filepath.Join(destDir, filepath.Base(p.Filename))
		if err := os.WriteFile(out, data, 0o644); err != nil {
			walkErr = fmt.Errorf("gmail save attachment %q: %w", out, err)
			return
		}
		saved = append(saved, out)
	})
	return saved, walkErr
}

// attachmentData returns a part's bytes, fetching by attachment id when the
// data is not inlined (large attachments) and decoding inline data otherwise.
func (g *gmailProvider) attachmentData(ctx context.Context, msgID string, p *gmail.MessagePart) ([]byte, error) {
	if p.Body.AttachmentId != "" {
		att, err := g.service.Users.Messages.Attachments.
			Get("me", msgID, p.Body.AttachmentId).Context(ctx).Do()
		if err != nil {
			return nil, fmt.Errorf("gmail get attachment %s: %w", p.Filename, err)
		}
		return base64.RawURLEncoding.DecodeString(att.Data)
	}
	if p.Body.Data != "" {
		return base64.RawURLEncoding.DecodeString(p.Body.Data)
	}
	return nil, nil
}

// MarkRead toggles the UNREAD label.
func (g *gmailProvider) MarkRead(ctx context.Context, id string, read bool) error {
	req := &gmail.ModifyMessageRequest{}
	if read {
		req.RemoveLabelIds = []string{"UNREAD"}
	} else {
		req.AddLabelIds = []string{"UNREAD"}
	}
	if _, err := g.service.Users.Messages.Modify("me", id, req).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail markread %s: %w", id, err)
	}
	return nil
}

// SetLabels replaces a message's user labels with the given set, preserving
// system labels (INBOX/UNREAD/etc). Missing user labels are created on demand.
func (g *gmailProvider) SetLabels(ctx context.Context, id string, labels []string) error {
	// Resolve/create target label ids.
	want := make(map[string]bool, len(labels))
	var add []string
	for _, name := range labels {
		lid, err := g.resolveLabelIDCreating(ctx, name)
		if err != nil {
			return err
		}
		if !want[lid] {
			want[lid] = true
			add = append(add, lid)
		}
	}

	// Fetch the message's current user labels so we can remove the ones not in
	// the target set (system labels are left untouched).
	m, err := g.service.Users.Messages.Get("me", id).Format("minimal").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail setlabels get %s: %w", id, err)
	}
	var remove []string
	for _, lid := range m.LabelIds {
		if gmailSystemLabels[lid] || want[lid] {
			continue
		}
		remove = append(remove, lid)
	}

	req := &gmail.ModifyMessageRequest{AddLabelIds: add, RemoveLabelIds: remove}
	if _, err := g.service.Users.Messages.Modify("me", id, req).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail setlabels %s: %w", id, err)
	}
	return nil
}

// Delete trashes a message (recoverable) or permanently deletes it. Permanent
// deletion requires the full gmail.MailGoogleComScope; with only modify scope
// Gmail returns 403.
func (g *gmailProvider) Delete(ctx context.Context, id string, permanent bool) error {
	if permanent {
		if err := g.service.Users.Messages.Delete("me", id).Context(ctx).Do(); err != nil {
			return fmt.Errorf("gmail delete %s (permanent): %w", id, err)
		}
		return nil
	}
	if _, err := g.service.Users.Messages.Trash("me", id).Context(ctx).Do(); err != nil {
		return fmt.Errorf("gmail trash %s: %w", id, err)
	}
	return nil
}

// ListFolders returns the mailbox's labels as Folders.
func (g *gmailProvider) ListFolders(ctx context.Context) ([]Folder, error) {
	resp, err := g.service.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return nil, fmt.Errorf("gmail listfolders: %w", err)
	}
	out := make([]Folder, 0, len(resp.Labels))
	for _, l := range resp.Labels {
		out = append(out, Folder{ID: l.Id, Name: l.Name})
	}
	return out, nil
}

// --- label resolution -------------------------------------------------------

// loadLabels populates the name->id cache from the Labels.List endpoint.
func (g *gmailProvider) loadLabels(ctx context.Context) error {
	resp, err := g.service.Users.Labels.List("me").Context(ctx).Do()
	if err != nil {
		return fmt.Errorf("gmail labels: %w", err)
	}
	g.labelCache = make(map[string]string, len(resp.Labels))
	for _, l := range resp.Labels {
		g.labelCache[l.Name] = l.Id
	}
	return nil
}

// resolveLabelID maps a label name (or system id) to its id without creating.
func (g *gmailProvider) resolveLabelID(ctx context.Context, name string) (string, error) {
	if gmailSystemLabels[strings.ToUpper(name)] {
		return strings.ToUpper(name), nil
	}
	if g.labelCache == nil {
		if err := g.loadLabels(ctx); err != nil {
			return "", err
		}
	}
	if id, ok := g.labelCache[name]; ok {
		return id, nil
	}
	return "", fmt.Errorf("gmail: label %q: %w", name, ErrNotFound)
}

// resolveLabelIDCreating maps a label name to its id, creating a user label if
// it does not yet exist. System labels are returned as-is.
func (g *gmailProvider) resolveLabelIDCreating(ctx context.Context, name string) (string, error) {
	if gmailSystemLabels[strings.ToUpper(name)] {
		return strings.ToUpper(name), nil
	}
	if g.labelCache == nil {
		if err := g.loadLabels(ctx); err != nil {
			return "", err
		}
	}
	if id, ok := g.labelCache[name]; ok {
		return id, nil
	}
	created, err := g.service.Users.Labels.Create("me", &gmail.Label{Name: name}).Context(ctx).Do()
	if err != nil {
		// A concurrent create or pre-existing label surfaces as 409; refresh
		// the cache and retry the lookup once.
		if reloadErr := g.loadLabels(ctx); reloadErr == nil {
			if id, ok := g.labelCache[name]; ok {
				return id, nil
			}
		}
		return "", fmt.Errorf("gmail create label %q: %w", name, err)
	}
	g.labelCache[created.Name] = created.Id
	return created.Id, nil
}

// --- conversion helpers -----------------------------------------------------

// gmailQuery builds Gmail search operators from the unread/since options.
func gmailQuery(opts ListOptions) string {
	var parts []string
	if opts.UnreadOnly {
		parts = append(parts, "is:unread")
	}
	if !opts.Since.IsZero() {
		// Gmail's after: takes epoch seconds or YYYY/MM/DD; epoch is exact.
		parts = append(parts, fmt.Sprintf("after:%d", opts.Since.Unix()))
	}
	return strings.Join(parts, " ")
}

// gmailSummary converts a Gmail message (metadata or full) to a Summary.
func gmailSummary(m *gmail.Message) Summary {
	s := Summary{
		ID:       m.Id,
		ThreadID: m.ThreadId,
		From:     parseAddr(decodeMIMEHeader(gmailHeader(m, "From"))),
		Subject:  decodeMIMEHeader(gmailHeader(m, "Subject")),
	}
	// Received: prefer internalDate (epoch ms); fall back to the Date header.
	if m.InternalDate != 0 {
		s.Received = time.UnixMilli(m.InternalDate)
	} else if d := gmailHeader(m, "Date"); d != "" {
		if t, err := mail.ParseDate(d); err == nil {
			s.Received = t
		}
	}
	for _, lid := range m.LabelIds {
		switch lid {
		case "UNREAD":
			s.Unread = true
		default:
			s.Labels = append(s.Labels, lid)
		}
	}
	// hasAttachments: inferred from a filename-bearing part when payload present.
	if m.Payload != nil {
		walkParts(m.Payload, func(p *gmail.MessagePart) {
			if p.Filename != "" {
				s.HasAttachments = true
			}
		})
	}
	return s
}

// gmailHeader returns a payload header value (case-insensitive), or "".
func gmailHeader(m *gmail.Message, name string) string {
	if m.Payload == nil {
		return ""
	}
	for _, h := range m.Payload.Headers {
		if strings.EqualFold(h.Name, name) {
			return h.Value
		}
	}
	return ""
}

// gmailBodyByType walks the MIME tree and returns the decoded body of the first
// part whose MIME type matches want (e.g. "text/plain").
func gmailBodyByType(part *gmail.MessagePart, want string) string {
	if part == nil {
		return ""
	}
	if strings.HasPrefix(strings.ToLower(part.MimeType), want) && part.Body != nil && part.Body.Data != "" {
		if data, err := base64.RawURLEncoding.DecodeString(part.Body.Data); err == nil {
			return string(data)
		}
	}
	for _, p := range part.Parts {
		if body := gmailBodyByType(p, want); body != "" {
			return body
		}
	}
	return ""
}

// walkParts visits every part in a MIME tree depth-first.
func walkParts(part *gmail.MessagePart, fn func(*gmail.MessagePart)) {
	if part == nil {
		return
	}
	fn(part)
	for _, p := range part.Parts {
		walkParts(p, fn)
	}
}

// decodeMIMEHeader decodes RFC 2047 encoded-words (e.g. UTF-8 Croatian
// subjects) to plain UTF-8, returning the input unchanged on failure.
func decodeMIMEHeader(s string) string {
	if s == "" || !strings.Contains(s, "=?") {
		return s
	}
	dec := new(mime.WordDecoder)
	if out, err := dec.DecodeHeader(s); err == nil {
		return out
	}
	return s
}

// parseAddr extracts the bare address from a possibly display-named header
// value (e.g. `Marios <m@x.com>` -> `m@x.com`).
func parseAddr(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.LastIndex(s, "<"); i >= 0 {
		if j := strings.Index(s[i:], ">"); j >= 0 {
			return strings.TrimSpace(s[i+1 : i+j])
		}
	}
	return s
}

// splitAddrs splits a comma-separated recipient header into bare addresses.
func splitAddrs(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(s, ",") {
		if a := parseAddr(decodeMIMEHeader(part)); a != "" {
			out = append(out, a)
		}
	}
	return out
}
