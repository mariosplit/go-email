// outlook_calendar.go - Outlook 365 (Microsoft Graph) implementation of the
// CalendarProvider interface. Mirrors the SDK idiom of outlook_read.go: the
// configured UserID is the mailbox, builder/config type names are verified
// against msgraph-sdk-go v1.59.0, and @odata.nextLink is followed for list.
//
// Graph permission required: Calendars.ReadWrite (application) on the same
// Azure app dl/go-email already use. With only Mail.* the event calls 403.
package email

import (
	"context"
	"fmt"
	"time"

	abstractions "github.com/microsoft/kiota-abstractions-go"
	graphmodels "github.com/microsoftgraph/msgraph-sdk-go/models"
	graphusers "github.com/microsoftgraph/msgraph-sdk-go/users"
)

// preferTZ is the zone events are read back in. Without the Prefer header Graph
// returns event times in UTC; we ask for AWST so the wall-clock matches what
// callers set. Read methods that omit it would surface UTC — keep it on all
// event GETs.
const preferTZ = "Australia/Perth"

// tzHeaders builds the Prefer: outlook.timezone header set for a read.
func tzHeaders() *abstractions.RequestHeaders {
	h := abstractions.NewRequestHeaders()
	h.TryAdd("Prefer", fmt.Sprintf("outlook.timezone=\"%s\"", preferTZ))
	return h
}

// graphDateLayout is Graph's dateTime form for event start/end (no zone suffix;
// the zone travels in the sibling timeZone field).
const graphDateLayout = "2006-01-02T15:04:05"

// eventSelect is the field set fetched for event listings/reads.
var eventSelect = []string{
	"id", "subject", "start", "end", "location", "organizer",
	"isAllDay", "bodyPreview", "categories",
}

// outlookProvider implements CalendarProvider (compile-time check).
var _ CalendarProvider = (*outlookProvider)(nil)

// ListEvents returns events overlapping the range, soonest first, via the
// calendarView endpoint (which expands recurrences within the window).
func (o *outlookProvider) ListEvents(ctx context.Context, opts EventListOptions) ([]Event, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	start := opts.Start
	if start.IsZero() {
		start = time.Now()
	}
	end := opts.End
	if end.IsZero() {
		end = start.AddDate(1, 0, 0) // default 1-year upper bound
	}
	cfg := &graphusers.ItemCalendarViewRequestBuilderGetRequestConfiguration{
		Headers: tzHeaders(),
		QueryParameters: &graphusers.ItemCalendarViewRequestBuilderGetQueryParameters{
			StartDateTime: strptr(start.UTC().Format("2006-01-02T15:04:05Z")),
			EndDateTime:   strptr(end.UTC().Format("2006-01-02T15:04:05Z")),
			Select:        eventSelect,
			Orderby:       []string{"start/dateTime"},
			Top:           outlookEventTop(opts),
		},
	}
	resp, err := o.client.Users().ByUserId(uid).CalendarView().Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook list events %s: %w", uid, err)
	}
	var out []Event
	for _, e := range resp.GetValue() {
		out = append(out, outlookEvent(e))
		if opts.Limit > 0 && len(out) >= opts.Limit {
			return out, nil
		}
	}
	next := resp.GetOdataNextLink()
	for next != nil && *next != "" {
		page, err := o.client.Users().ByUserId(uid).CalendarView().WithUrl(*next).Get(ctx, cfg)
		if err != nil {
			return out, fmt.Errorf("outlook list events %s (page): %w", uid, err)
		}
		for _, e := range page.GetValue() {
			out = append(out, outlookEvent(e))
			if opts.Limit > 0 && len(out) >= opts.Limit {
				return out, nil
			}
		}
		next = page.GetOdataNextLink()
	}
	return out, nil
}

// ReadEvent returns one event by id, including body and attendees.
func (o *outlookProvider) ReadEvent(ctx context.Context, id string) (*Event, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	cfg := &graphusers.ItemEventsEventItemRequestBuilderGetRequestConfiguration{
		Headers: tzHeaders(),
		QueryParameters: &graphusers.ItemEventsEventItemRequestBuilderGetQueryParameters{
			Select: append(eventSelect, "body", "attendees"),
		},
	}
	m, err := o.client.Users().ByUserId(uid).Events().ByEventId(id).Get(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("outlook read event %s/%s: %w", uid, id, err)
	}
	ev := outlookEvent(m)
	if b := m.GetBody(); b != nil {
		if c := b.GetContent(); c != nil {
			ev.BodyText = *c
		}
	}
	for _, a := range m.GetAttendees() {
		if ea := a.GetEmailAddress(); ea != nil {
			if addr := ea.GetAddress(); addr != nil {
				ev.Attendees = append(ev.Attendees, *addr)
			}
		}
	}
	return &ev, nil
}

// CreateEvent creates an event and returns it with its assigned id.
func (o *outlookProvider) CreateEvent(ctx context.Context, e Event) (*Event, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	body := o.buildEvent(e, false)
	created, err := o.client.Users().ByUserId(uid).Events().Post(ctx, body, nil)
	if err != nil {
		return nil, fmt.Errorf("outlook create event %s: %w", uid, err)
	}
	ev := outlookEvent(created)
	return &ev, nil
}

// UpdateEvent PATCHes the non-zero fields of e onto the event and returns it.
func (o *outlookProvider) UpdateEvent(ctx context.Context, id string, e Event) (*Event, error) {
	uid, err := o.user()
	if err != nil {
		return nil, err
	}
	body := o.buildEvent(e, true)
	updated, err := o.client.Users().ByUserId(uid).Events().ByEventId(id).Patch(ctx, body, nil)
	if err != nil {
		return nil, fmt.Errorf("outlook update event %s/%s: %w", uid, id, err)
	}
	ev := outlookEvent(updated)
	return &ev, nil
}

// DeleteEvent removes an event by id.
func (o *outlookProvider) DeleteEvent(ctx context.Context, id string) error {
	uid, err := o.user()
	if err != nil {
		return err
	}
	if err := o.client.Users().ByUserId(uid).Events().ByEventId(id).Delete(ctx, nil); err != nil {
		return fmt.Errorf("outlook delete event %s/%s: %w", uid, id, err)
	}
	return nil
}

// buildEvent maps an Event to a Graph Eventable. When patch is true only the
// set (non-zero) fields are written, so an UpdateEvent leaves omitted fields
// untouched; on create the core fields are always written. Categories, when
// non-nil, are written (replacing the list under PATCH semantics).
func (o *outlookProvider) buildEvent(e Event, patch bool) graphmodels.Eventable {
	m := graphmodels.NewEvent()
	tz := e.TimeZone
	if tz == "" {
		tz = "UTC"
	}
	if e.Subject != "" || !patch {
		m.SetSubject(strptr(e.Subject))
	}
	if !e.Start.IsZero() {
		m.SetStart(graphDateTimeTZ(e.Start, tz))
	}
	if !e.End.IsZero() {
		m.SetEnd(graphDateTimeTZ(e.End, tz))
	}
	if e.AllDay || !patch {
		b := e.AllDay
		m.SetIsAllDay(&b)
	}
	if e.Location != "" {
		loc := graphmodels.NewLocation()
		loc.SetDisplayName(strptr(e.Location))
		m.SetLocation(loc)
	}
	if e.BodyText != "" {
		body := graphmodels.NewItemBody()
		ct := graphmodels.TEXT_BODYTYPE
		body.SetContentType(&ct)
		body.SetContent(strptr(e.BodyText))
		m.SetBody(body)
	}
	if len(e.Attendees) > 0 {
		var as []graphmodels.Attendeeable
		for _, addr := range e.Attendees {
			a := graphmodels.NewAttendee()
			ea := graphmodels.NewEmailAddress()
			ea.SetAddress(strptr(addr))
			a.SetEmailAddress(ea)
			t := graphmodels.REQUIRED_ATTENDEETYPE
			a.SetTypeEscaped(&t)
			as = append(as, a)
		}
		m.SetAttendees(as)
	}
	if e.Categories != nil {
		m.SetCategories(e.Categories)
	}
	return m
}

// graphDateTimeTZ builds a Graph DateTimeTimeZone from a wall-clock time and an
// IANA zone name. The time's clock fields are sent verbatim (formatted without
// a zone suffix); Graph interprets them in the named zone.
func graphDateTimeTZ(t time.Time, tz string) graphmodels.DateTimeTimeZoneable {
	d := graphmodels.NewDateTimeTimeZone()
	d.SetDateTime(strptr(t.Format(graphDateLayout)))
	d.SetTimeZone(strptr(tz))
	return d
}

// outlookEvent maps a Graph Eventable to our Event.
func outlookEvent(m graphmodels.Eventable) Event {
	e := Event{}
	if m.GetId() != nil {
		e.ID = *m.GetId()
	}
	if m.GetSubject() != nil {
		e.Subject = *m.GetSubject()
	}
	e.Start, e.TimeZone = parseGraphDT(m.GetStart())
	e.End, _ = parseGraphDT(m.GetEnd())
	if m.GetIsAllDay() != nil {
		e.AllDay = *m.GetIsAllDay()
	}
	if loc := m.GetLocation(); loc != nil && loc.GetDisplayName() != nil {
		e.Location = *loc.GetDisplayName()
	}
	if org := m.GetOrganizer(); org != nil {
		if ea := org.GetEmailAddress(); ea != nil && ea.GetAddress() != nil {
			e.Organizer = *ea.GetAddress()
		}
	}
	if bp := m.GetBodyPreview(); bp != nil && e.BodyText == "" {
		e.BodyText = *bp
	}
	e.Categories = m.GetCategories()
	return e
}

// parseGraphDT parses a Graph DateTimeTimeZone into a wall-clock time and its
// zone name. Returns the zero time if absent/unparseable.
func parseGraphDT(d graphmodels.DateTimeTimeZoneable) (time.Time, string) {
	if d == nil || d.GetDateTime() == nil {
		return time.Time{}, ""
	}
	tz := ""
	if d.GetTimeZone() != nil {
		tz = *d.GetTimeZone()
	}
	// Graph returns dateTime like "2026-07-27T00:00:00.0000000".
	for _, layout := range []string{
		"2006-01-02T15:04:05.0000000",
		"2006-01-02T15:04:05.000000",
		graphDateLayout,
	} {
		if t, err := time.Parse(layout, *d.GetDateTime()); err == nil {
			return t, tz
		}
	}
	return time.Time{}, tz
}

// outlookEventTop maps the option Limit to the Graph $top page size.
func outlookEventTop(opts EventListOptions) *int32 {
	if opts.Limit > 0 && opts.Limit <= 1000 {
		return i32ptr(int32(opts.Limit))
	}
	return i32ptr(100)
}
