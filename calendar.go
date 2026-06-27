// calendar.go - provider-agnostic calendar types and the Client wrappers that
// expose them. The Outlook implementation lives in outlook_calendar.go; Gmail
// does not implement CalendarProvider (the build-in gmailProvider returns
// ErrUnsupported via the type assertion in calendar()).
package email

import (
	"context"
	"time"
)

// Event is a calendar event. Times are wall-clock values paired with TimeZone
// (an IANA name, e.g. "Australia/Perth"); for an all-day event the Start/End
// carry the date with a zero time-of-day and AllDay is true.
type Event struct {
	// ID is the provider event identifier. Treat it as opaque. Empty when an
	// Event is passed to Create (the provider assigns it).
	ID string

	// Subject is the event title.
	Subject string

	// Start, End are the event's wall-clock start/end in TimeZone.
	Start time.Time
	End   time.Time

	// TimeZone is the IANA zone the Start/End are expressed in (e.g.
	// "Australia/Perth"). Empty on input means the provider's default zone.
	TimeZone string

	// AllDay reports an all-day event (Start/End are dates).
	AllDay bool

	// Location is the free-text venue, if any.
	Location string

	// BodyText is the plain-text event description.
	BodyText string

	// Organizer is the organiser's email address (output only).
	Organizer string

	// Attendees holds attendee email addresses.
	Attendees []string

	// Categories holds the event's Outlook category tags.
	Categories []string
}

// EventListOptions bounds a calendar list query to a date range.
type EventListOptions struct {
	// Start, End bound the query (inclusive of events overlapping the range).
	// Zero Start means "now"; zero End means "no upper bound" for the
	// upcoming-events list.
	Start time.Time
	End   time.Time

	// Limit caps the number of events returned (0 = provider default).
	Limit int
}

// CalendarProvider is implemented by providers that support calendar
// operations (Outlook 365). All methods take a context and act on the mailbox
// the provider was configured for (OutlookConfig.UserID).
type CalendarProvider interface {
	// ListEvents returns events in the option's date range, soonest first.
	ListEvents(ctx context.Context, opts EventListOptions) ([]Event, error)

	// ReadEvent returns one event by id, including body and attendees.
	ReadEvent(ctx context.Context, id string) (*Event, error)

	// CreateEvent creates an event and returns it with its assigned id.
	CreateEvent(ctx context.Context, e Event) (*Event, error)

	// UpdateEvent applies the non-zero fields of e to the event with the given
	// id and returns the updated event. Categories, when non-nil, REPLACE the
	// event's category list wholesale (Graph PATCH semantics).
	UpdateEvent(ctx context.Context, id string, e Event) (*Event, error)

	// DeleteEvent removes an event by id.
	DeleteEvent(ctx context.Context, id string) error
}

// calendar returns the client's provider as a CalendarProvider, or an error if
// the configured provider does not support calendar operations.
func (c *Client) calendar() (CalendarProvider, error) {
	cp, ok := c.provider.(CalendarProvider)
	if !ok {
		return nil, ErrUnsupported
	}
	return cp, nil
}

// ListEvents lists calendar events in the given range (context.Background).
func (c *Client) ListEvents(opts EventListOptions) ([]Event, error) {
	return c.ListEventsWithContext(context.Background(), opts)
}

// ListEventsWithContext lists calendar events in the given range.
func (c *Client) ListEventsWithContext(ctx context.Context, opts EventListOptions) ([]Event, error) {
	cp, err := c.calendar()
	if err != nil {
		return nil, err
	}
	return cp.ListEvents(ctx, opts)
}

// ReadEvent reads one event by id (context.Background).
func (c *Client) ReadEvent(id string) (*Event, error) {
	return c.ReadEventWithContext(context.Background(), id)
}

// ReadEventWithContext reads one event by id.
func (c *Client) ReadEventWithContext(ctx context.Context, id string) (*Event, error) {
	cp, err := c.calendar()
	if err != nil {
		return nil, err
	}
	return cp.ReadEvent(ctx, id)
}

// CreateEvent creates a calendar event (context.Background).
func (c *Client) CreateEvent(e Event) (*Event, error) {
	return c.CreateEventWithContext(context.Background(), e)
}

// CreateEventWithContext creates a calendar event.
func (c *Client) CreateEventWithContext(ctx context.Context, e Event) (*Event, error) {
	cp, err := c.calendar()
	if err != nil {
		return nil, err
	}
	return cp.CreateEvent(ctx, e)
}

// UpdateEvent updates a calendar event (context.Background).
func (c *Client) UpdateEvent(id string, e Event) (*Event, error) {
	return c.UpdateEventWithContext(context.Background(), id, e)
}

// UpdateEventWithContext updates a calendar event.
func (c *Client) UpdateEventWithContext(ctx context.Context, id string, e Event) (*Event, error) {
	cp, err := c.calendar()
	if err != nil {
		return nil, err
	}
	return cp.UpdateEvent(ctx, id, e)
}

// DeleteEvent deletes a calendar event (context.Background).
func (c *Client) DeleteEvent(id string) error {
	return c.DeleteEventWithContext(context.Background(), id)
}

// DeleteEventWithContext deletes a calendar event.
func (c *Client) DeleteEventWithContext(ctx context.Context, id string) error {
	cp, err := c.calendar()
	if err != nil {
		return err
	}
	return cp.DeleteEvent(ctx, id)
}
