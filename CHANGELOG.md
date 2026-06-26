# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2026-06-26

### Added
- **Mailbox read/management operations** via the new `MailboxProvider`
  interface, implemented for **both** Outlook 365 and Gmail. Existing
  send-only code is unaffected (the `Send` API is unchanged); the new methods
  are purely additive.
  - `List` — message headers from a folder/label (filters: unread-only, since,
    limit), newest first, with pagination.
  - `Read` — full message including text/HTML body and To/Cc.
  - `Search` — provider-native full-text search (Graph `$search` KQL / Gmail
    search operators).
  - `Move` — relocate to a folder (Outlook) or relabel+archive (Gmail).
  - `ListAttachments` / `SaveAttachments` — enumerate and download file
    attachments.
  - `MarkRead`, `SetLabels` (Outlook categories / Gmail labels), `Delete`
    (trash or permanent), `ListFolders`.
- New shared types: `Summary`, `FullMessage`, `ListOptions`, `Folder`,
  `AttachmentMeta`.
- `Client` convenience wrappers for every new operation (default 30s timeout)
  plus `…WithContext` variants, mirroring `Send`/`SendWithContext`.
- `OutlookConfig.UserID` — the mailbox that read/management operations target
  (not required for sending).
- `GmailConfig.Scopes` and `GmailAuthHelper.Scopes` — override OAuth2 scopes.
- Sentinel errors `ErrUnsupported` and `ErrNotFound`.

### Changed
- **Gmail default scopes are now `gmail.send` + `gmail.modify`** (was
  `gmail.send` only) so the new read/move/label/trash operations work out of
  the box. **A token previously minted for `gmail.send` alone must be
  re-consented** (re-run the auth flow); otherwise read/modify calls return
  403. Permanent `Delete` additionally requires `gmail.MailGoogleComScope` —
  add it to `GmailConfig.Scopes`. Outlook is unaffected, but the Azure app
  needs `Mail.ReadWrite` (application) granted for the read/write operations.
- `GmailAuthHelper` now forces the consent screen (`ApprovalForce`) so a
  refresh token is re-issued when scopes widen.

### Notes
- Outlook `Delete(permanent=true)` returns `ErrUnsupported` (Graph v1.0 has no
  permanent-delete action); use `permanent=false` to move to Deleted Items.
- Gmail "folders" are labels; `Move` adds the destination label and removes
  `INBOX`. Bodies/attachments are decoded with unpadded base64url
  (`RawURLEncoding`). UTF-8 (e.g. Croatian) subjects are RFC 2047-decoded.

## [1.0.0] - 2024-01-13

### Added
- Initial release of go-email package
- Support for Outlook 365 via Microsoft Graph API
- Support for Gmail via Gmail API
- OAuth2 authentication for both providers
- HTML email support
- File attachments with automatic MIME type detection
- CC and BCC recipients
- Environment variable configuration
- Context support for timeouts and cancellation
- Comprehensive error handling
- QuickSend convenience function
- Version information API

### Security
- Secure OAuth2 authentication flow
- No hardcoded credentials
- Support for service account authentication (Gmail)

### Documentation
- Comprehensive README with examples
- Integration guide for production use
- Provider-specific setup guides
- API documentation
- Multiple working examples

### Testing
- Unit tests for core functionality
- Integration test framework
- Mock implementations for testing

## [Unreleased]

### Planned
- SendGrid provider support
- AWS SES provider support
- Email template engine
- Webhook support for email events
- Batch sending optimization
- Email validation utilities
- Retry mechanism with exponential backoff
- Connection pooling for high-volume sending
