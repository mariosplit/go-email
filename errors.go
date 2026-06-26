// errors.go - Sentinel errors for the email package.
package email

import "errors"

var (
	// ErrUnsupported is returned when a configured provider does not implement
	// the requested mailbox operation (i.e. it is not a MailboxProvider).
	ErrUnsupported = errors.New("operation not supported by provider")

	// ErrNotFound is returned when a referenced message, folder, or label does
	// not exist.
	ErrNotFound = errors.New("not found")
)
