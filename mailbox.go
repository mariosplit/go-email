// mailbox.go - Read/manage operations (list, read, search, move, labels,
// attachments, delete, folders) layered additively on top of the send-only
// core. Existing send-only callers are unaffected: they simply never call
// these methods. Both the Outlook 365 and Gmail providers implement them.
//
// Provider-portability note: Outlook has real folders; Gmail has only labels.
// The interface speaks a common vocabulary of folder/label NAMES (e.g. "Inbox",
// "Archive"); each provider maps names to its own identifiers. Gmail "Move" is
// implemented as a label change (add destination, remove INBOX). These quirks
// are documented per method rather than leaked into the type system.
package email

import (
	"context"
	"time"
)

// Summary is a lightweight message header returned by List and Search. It
// deliberately omits the body so listing a folder is cheap.
type Summary struct {
	// ID is the provider-specific message identifier. For Outlook it is the
	// Graph message id; for Gmail the Gmail message id. Treat it as opaque.
	ID string

	// ThreadID groups a conversation. Populated for Gmail; empty for Outlook
	// (Outlook exposes conversationId, which is not currently surfaced).
	ThreadID string

	// From is the sender's email address.
	From string

	// Subject is the message subject (UTF-8, may be non-ASCII).
	Subject string

	// Received is the time the message was received.
	Received time.Time

	// HasAttachments reports whether the message carries file attachments.
	HasAttachments bool

	// Unread reports whether the message is unread.
	Unread bool

	// Labels holds the message's labels (Gmail) or categories (Outlook).
	Labels []string
}

// FullMessage is a message with its body, returned by Read.
type FullMessage struct {
	Summary

	// To, Cc are the recipient addresses.
	To []string
	Cc []string

	// BodyText is the plain-text body, if available.
	BodyText string

	// BodyHTML is the HTML body, if the message was HTML.
	BodyHTML string
}

// ListOptions filters and bounds a List or Search call. The zero value lists
// the inbox with provider defaults.
type ListOptions struct {
	// Folder is the folder/label to list. Empty means "inbox". For Outlook
	// this is a well-known folder name or folder id (e.g. "inbox",
	// "sentitems", "archive"); for Gmail it is a label name or system label
	// id (e.g. "INBOX", "SENT"). Ignored by Search.
	Folder string

	// UnreadOnly restricts results to unread messages.
	UnreadOnly bool

	// Since restricts results to messages received at or after this time.
	// The zero value means no lower bound.
	Since time.Time

	// Limit caps the number of messages returned (0 means provider default,
	// no explicit cap). Acts as a ceiling across pages.
	Limit int
}

// Folder is a mail folder (Outlook) or label (Gmail).
type Folder struct {
	// ID is the provider identifier, suitable for ListOptions.Folder and Move.
	ID string

	// Name is the human-readable display name.
	Name string

	// Unread is the count of unread items, where the provider reports it
	// (Outlook); 0 if unknown.
	Unread int
}

// AttachmentMeta describes an attachment without downloading its content.
type AttachmentMeta struct {
	// ID is the provider attachment identifier (used to fetch content).
	ID string

	// Filename is the attachment's file name.
	Filename string

	// MimeType is the attachment's content type.
	MimeType string

	// Size is the attachment size in bytes (0 if unknown).
	Size int64
}

// MailboxProvider extends Provider with read and management operations.
// Both built-in providers (Outlook 365, Gmail) implement it. Code that only
// sends can continue to use Provider; code that needs the wider surface can
// type-assert a provider to MailboxProvider or use the Client methods below,
// which require it.
//
// All methods take a context for timeout/cancellation. The mailbox operated on
// is the one the provider was configured for (Outlook: the address passed as
// the message From / the configured user; Gmail: the authenticated "me").
type MailboxProvider interface {
	Provider

	// List returns message headers from a folder per opts (default inbox),
	// newest first.
	List(ctx context.Context, opts ListOptions) ([]Summary, error)

	// Read returns one message including its body.
	Read(ctx context.Context, id string) (*FullMessage, error)

	// Search runs a provider-native full-text search. The query uses the
	// provider's own syntax (Graph $search KQL / Gmail search operators).
	// opts bounds the results (Folder is ignored).
	Search(ctx context.Context, query string, opts ListOptions) ([]Summary, error)

	// Move relocates a message to the destination folder/label. For Outlook
	// the message is moved; for Gmail the destination label is added and
	// INBOX removed (archive-style move). dest is a folder/label name or id.
	Move(ctx context.Context, id, dest string) error

	// ListAttachments returns metadata for a message's file attachments.
	ListAttachments(ctx context.Context, id string) ([]AttachmentMeta, error)

	// SaveAttachments writes a message's file attachments into destDir and
	// returns the paths written. destDir is created if it does not exist.
	SaveAttachments(ctx context.Context, id, destDir string) ([]string, error)

	// SaveMessageRaw writes the message's raw RFC822 MIME (.eml) into destDir
	// under a collision-free name derived from baseName (".eml" is appended if
	// absent; reserved/control chars in baseName are sanitized), and returns the
	// path written. destDir is created if it does not exist. The raw MIME is the
	// provider's verbatim wire form (Graph $value / Gmail raw), suitable for an
	// .eml->PDF converter. Providers that cannot export raw MIME return
	// ErrUnsupported.
	SaveMessageRaw(ctx context.Context, id, destDir, baseName string) (string, error)

	// MarkRead sets a message's read state.
	MarkRead(ctx context.Context, id string, read bool) error

	// SetLabels replaces a message's labels (Gmail) or categories (Outlook)
	// with the given set. Names are used; for Gmail, missing user labels are
	// created on demand.
	SetLabels(ctx context.Context, id string, labels []string) error

	// Delete removes a message. If permanent is false the message is moved to
	// the trash/deleted-items and is recoverable; if true it is permanently
	// deleted where the provider supports it (Gmail requires the full-access
	// scope for permanent deletion).
	Delete(ctx context.Context, id string, permanent bool) error

	// ListFolders returns the mailbox's folders (Outlook) or labels (Gmail).
	ListFolders(ctx context.Context) ([]Folder, error)
}

// Compile-time guarantees that both built-in providers implement the full
// mailbox surface, not just Send.
var (
	_ MailboxProvider = (*outlookProvider)(nil)
	_ MailboxProvider = (*gmailProvider)(nil)
)

// defaultTimeout matches the send path's default per-call timeout.
const defaultTimeout = 30 * time.Second

// mailbox returns the client's provider as a MailboxProvider, or an error if
// the configured provider does not support mailbox operations. Both built-in
// providers do; this guard exists for custom providers.
func (c *Client) mailbox() (MailboxProvider, error) {
	mp, ok := c.provider.(MailboxProvider)
	if !ok {
		return nil, ErrUnsupported
	}
	return mp, nil
}

// List returns message headers from a folder (default inbox), newest first,
// with a default timeout.
func (c *Client) List(opts ListOptions) ([]Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.ListWithContext(ctx, opts)
}

// ListWithContext is List with a caller-supplied context.
func (c *Client) ListWithContext(ctx context.Context, opts ListOptions) ([]Summary, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.List(ctx, opts)
}

// Read returns one message including its body, with a default timeout.
func (c *Client) Read(id string) (*FullMessage, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.ReadWithContext(ctx, id)
}

// ReadWithContext is Read with a caller-supplied context.
func (c *Client) ReadWithContext(ctx context.Context, id string) (*FullMessage, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.Read(ctx, id)
}

// Search runs a provider-native full-text search, with a default timeout.
func (c *Client) Search(query string, opts ListOptions) ([]Summary, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.SearchWithContext(ctx, query, opts)
}

// SearchWithContext is Search with a caller-supplied context.
func (c *Client) SearchWithContext(ctx context.Context, query string, opts ListOptions) ([]Summary, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.Search(ctx, query, opts)
}

// Move relocates a message to the destination folder/label, with a default
// timeout. See MailboxProvider.Move for Gmail's archive-style semantics.
func (c *Client) Move(id, dest string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.MoveWithContext(ctx, id, dest)
}

// MoveWithContext is Move with a caller-supplied context.
func (c *Client) MoveWithContext(ctx context.Context, id, dest string) error {
	mp, err := c.mailbox()
	if err != nil {
		return err
	}
	return mp.Move(ctx, id, dest)
}

// ListAttachments returns metadata for a message's file attachments, with a
// default timeout.
func (c *Client) ListAttachments(id string) ([]AttachmentMeta, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.ListAttachmentsWithContext(ctx, id)
}

// ListAttachmentsWithContext is ListAttachments with a caller-supplied context.
func (c *Client) ListAttachmentsWithContext(ctx context.Context, id string) ([]AttachmentMeta, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.ListAttachments(ctx, id)
}

// SaveAttachments writes a message's file attachments into destDir, with a
// default timeout, and returns the paths written.
func (c *Client) SaveAttachments(id, destDir string) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.SaveAttachmentsWithContext(ctx, id, destDir)
}

// SaveAttachmentsWithContext is SaveAttachments with a caller-supplied context.
func (c *Client) SaveAttachmentsWithContext(ctx context.Context, id, destDir string) ([]string, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.SaveAttachments(ctx, id, destDir)
}

// SaveMessageRaw writes a message's raw RFC822 MIME (.eml) into destDir under a
// collision-free name derived from baseName, with a default timeout, and returns
// the path written. See MailboxProvider.SaveMessageRaw.
func (c *Client) SaveMessageRaw(id, destDir, baseName string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.SaveMessageRawWithContext(ctx, id, destDir, baseName)
}

// SaveMessageRawWithContext is SaveMessageRaw with a caller-supplied context.
func (c *Client) SaveMessageRawWithContext(ctx context.Context, id, destDir, baseName string) (string, error) {
	mp, err := c.mailbox()
	if err != nil {
		return "", err
	}
	return mp.SaveMessageRaw(ctx, id, destDir, baseName)
}

// MarkRead sets a message's read state, with a default timeout.
func (c *Client) MarkRead(id string, read bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.MarkReadWithContext(ctx, id, read)
}

// MarkReadWithContext is MarkRead with a caller-supplied context.
func (c *Client) MarkReadWithContext(ctx context.Context, id string, read bool) error {
	mp, err := c.mailbox()
	if err != nil {
		return err
	}
	return mp.MarkRead(ctx, id, read)
}

// SetLabels replaces a message's labels/categories, with a default timeout.
func (c *Client) SetLabels(id string, labels []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.SetLabelsWithContext(ctx, id, labels)
}

// SetLabelsWithContext is SetLabels with a caller-supplied context.
func (c *Client) SetLabelsWithContext(ctx context.Context, id string, labels []string) error {
	mp, err := c.mailbox()
	if err != nil {
		return err
	}
	return mp.SetLabels(ctx, id, labels)
}

// Delete removes a message (trash if permanent is false), with a default
// timeout.
func (c *Client) Delete(id string, permanent bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.DeleteWithContext(ctx, id, permanent)
}

// DeleteWithContext is Delete with a caller-supplied context.
func (c *Client) DeleteWithContext(ctx context.Context, id string, permanent bool) error {
	mp, err := c.mailbox()
	if err != nil {
		return err
	}
	return mp.Delete(ctx, id, permanent)
}

// ListFolders returns the mailbox's folders (Outlook) or labels (Gmail), with
// a default timeout.
func (c *Client) ListFolders() ([]Folder, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	return c.ListFoldersWithContext(ctx)
}

// ListFoldersWithContext is ListFolders with a caller-supplied context.
func (c *Client) ListFoldersWithContext(ctx context.Context) ([]Folder, error) {
	mp, err := c.mailbox()
	if err != nil {
		return nil, err
	}
	return mp.ListFolders(ctx)
}
