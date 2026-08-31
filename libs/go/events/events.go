// Package events is the JetStream publish/subscribe surface shared by every Go service.
// Cross-service side effects travel as events on these subjects; a service never imports
// another service's Go package. See docs/adr/0003-nats-jetstream.md.
package events

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/logger"
)

// Subject is a `<domain>.<entity>.<verb>` subject, the naming scheme fixed in AGENTS.md.
type Subject string

// Subjects published in this repo. Add the constant here, not a string literal at the call site.
const (
	SubjectFamilyMemberInvited Subject = "family.member.invited"
	SubjectFamilyMemberJoined  Subject = "family.member.joined"
	SubjectFamilyMemberRemoved Subject = "family.member.removed"

	SubjectFinanceTransactionCreated Subject = "finance.transaction.created"
	SubjectFinanceTransactionUpdated Subject = "finance.transaction.updated"
	SubjectFinanceTransactionDeleted Subject = "finance.transaction.deleted"
	SubjectFinanceTransferCreated    Subject = "finance.transfer.created"
	SubjectFinanceAccountCreated     Subject = "finance.account.created"
	SubjectFinanceAccountUpdated     Subject = "finance.account.updated"
	SubjectFinanceBudgetCreated      Subject = "finance.budget.created"
	SubjectFinanceBudgetUpdated      Subject = "finance.budget.updated"
	SubjectFinanceBudgetExceeded     Subject = "finance.budget.exceeded"
	SubjectFinanceBudgetRecovered    Subject = "finance.budget.recovered"
	SubjectFinanceTemplateUsed       Subject = "finance.template.used"
	SubjectFinanceRecurringPosted    Subject = "finance.recurring.posted"

	SubjectRecipesRecipeCreated Subject = "recipes.recipe.created"
	SubjectRecipesRecipeUpdated Subject = "recipes.recipe.updated"
	SubjectRecipesRecipeDeleted Subject = "recipes.recipe.deleted"
	SubjectRecipesMealPlanned   Subject = "recipes.meal.planned"
)

// ErrBadSubject is returned when a subject does not match `<domain>.<entity>.<verb>`.
var ErrBadSubject = errors.New("events: subject must be <domain>.<entity>.<verb>")

// Domain is the first segment, which is also the stream a subject belongs to.
func (s Subject) Domain() string {
	parts := strings.Split(string(s), ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

// Validate enforces the three-segment naming rule.
func (s Subject) Validate() error {
	parts := strings.Split(string(s), ".")
	if len(parts) != 3 {
		return fmt.Errorf("%w: got %q", ErrBadSubject, string(s))
	}
	for _, p := range parts {
		if p == "" {
			return fmt.Errorf("%w: got %q", ErrBadSubject, string(s))
		}
	}
	return nil
}

// Config describes the connection to NATS.
type Config struct {
	// URL is the NATS server, e.g. nats://nats:4222.
	URL string
	// Name identifies the client in NATS monitoring; use the service name.
	Name string
	// ConnectTimeout bounds the initial connect. Zero means five seconds.
	ConnectTimeout time.Duration
}

// Bus is a JetStream connection with the helpers services actually need. Safe for
// concurrent use.
type Bus struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

// Connect dials NATS and opens JetStream.
func Connect(cfg Config) (*Bus, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("events: empty URL")
	}
	timeout := cfg.ConnectTimeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}

	conn, err := nats.Connect(cfg.URL, nats.Name(cfg.Name), nats.Timeout(timeout))
	if err != nil {
		return nil, fmt.Errorf("events: connect: %w", err)
	}

	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("events: jetstream: %w", err)
	}
	return &Bus{conn: conn, js: js}, nil
}

// Close drains and shuts the connection down.
func (b *Bus) Close() {
	if b.conn != nil {
		_ = b.conn.Drain()
	}
}

// EnsureStream creates (or updates) the stream that owns `<domain>.>`. Each service calls
// this at boot for the domains it publishes.
func (b *Bus) EnsureStream(ctx context.Context, domain string) error {
	if domain == "" {
		return fmt.Errorf("events: empty domain")
	}
	_, err := b.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      strings.ToUpper(domain),
		Subjects:  []string{domain + ".>"},
		Retention: jetstream.LimitsPolicy,
		MaxAge:    30 * 24 * time.Hour,
		Storage:   jetstream.FileStorage,
	})
	if err != nil {
		return fmt.Errorf("events: ensure stream %s: %w", domain, err)
	}
	return nil
}

// Publish sends a protobuf-encoded event and waits for the JetStream ack, so a publish that
// returns nil is durable.
func (b *Bus) Publish(ctx context.Context, subject Subject, msg proto.Message) error {
	if err := subject.Validate(); err != nil {
		return err
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("events: marshal %s: %w", subject, err)
	}
	m := &nats.Msg{Subject: string(subject), Data: payload}
	// The request that caused the event travels with it. Without this the causal chain stops
	// at the publisher: a consumer's log lines are a separate trace, and "the invite never
	// arrived" cannot be followed from the request that sent it to the handler that dropped
	// it — which docs/adr/0006 names as the reason events are the hard case.
	if id := logger.RequestID(ctx); id != "" {
		m.Header = nats.Header{RequestIDHeader: []string{id}}
	}
	if _, err := b.js.PublishMsg(ctx, m); err != nil {
		return fmt.Errorf("events: publish %s: %w", subject, err)
	}
	return nil
}

// RequestIDHeader carries the publishing request's id on the message. Same name as the HTTP
// header services exchange, so one id spans an RPC, the event it produced and the handler that
// consumed it.
const RequestIDHeader = "X-Request-Id"

// maxDeliver is how many times JetStream will redeliver before giving up on a message.
const maxDeliver = 5

// redeliveryBackoff is the delay before each redelivery, indexed by how many attempts have
// already been made.
//
// A bare Nak asks for immediate redelivery, so a handler failing for a reason that takes time
// to clear — the database is restarting, a sibling service is mid-deploy — burned all five
// attempts inside a few milliseconds and the event was gone. MaxDeliver: 5 was not a retry
// policy; it was five attempts at the same instant.
//
// The last value is used for any attempt beyond the list, though MaxDeliver stops it first.
var redeliveryBackoff = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
}

// ackable is the part of jetstream.Msg dispatch needs, so the delivery logic is testable
// without a NATS server.
type ackable interface {
	Subject() string
	Data() []byte
	Headers() nats.Header
	Ack() error
	NakWithDelay(delay time.Duration) error
	Metadata() (*jetstream.MsgMetadata, error)
}

// dispatch runs the handler for one message and acks or nacks it.
//
// A panic in a handler is treated as a failure of that message, not of the consumer. Without
// the recover it propagates out of the library's goroutine and takes the whole service with
// it — an event a service cannot parse should not be able to kill the service.
func dispatch(ctx context.Context, h Handler, m ackable) {
	// Put the publisher's id back on the context, so the handler's log lines join the trace
	// of the request that caused the event rather than starting a new one.
	if id := m.Headers().Get(RequestIDHeader); id != "" {
		ctx = logger.WithRequestID(ctx, id)
	}

	err := func() (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("events: handler panicked: %v", r)
			}
		}()
		return h(ctx, Subject(m.Subject()), m.Data())
	}()

	if err == nil {
		_ = m.Ack()
		return
	}
	_ = m.NakWithDelay(backoffFor(m))
}

// backoffFor picks the delay from how many times this message has already been delivered.
// Metadata is unavailable for a message that did not come from a stream, in which case the
// first delay is the safe answer.
func backoffFor(m ackable) time.Duration {
	meta, err := m.Metadata()
	if err != nil || meta == nil || meta.NumDelivered == 0 {
		return redeliveryBackoff[0]
	}
	i := int(meta.NumDelivered) - 1
	if i >= len(redeliveryBackoff) {
		i = len(redeliveryBackoff) - 1
	}
	return redeliveryBackoff[i]
}

// Handler processes one delivered event. Returning an error nacks the message so JetStream
// redelivers it; returning nil acks.
type Handler func(ctx context.Context, subject Subject, payload []byte) error

// Subscribe attaches a durable consumer to `<domain>.>` filtered to subject and dispatches
// to h until ctx is cancelled.
func (b *Bus) Subscribe(ctx context.Context, subject Subject, durable string, h Handler) (func(), error) {
	if err := subject.Validate(); err != nil {
		return nil, err
	}
	stream, err := b.js.Stream(ctx, strings.ToUpper(subject.Domain()))
	if err != nil {
		return nil, fmt.Errorf("events: stream %s: %w", subject.Domain(), err)
	}

	cons, err := stream.CreateOrUpdateConsumer(ctx, jetstream.ConsumerConfig{
		Durable:       durable,
		FilterSubject: string(subject),
		AckPolicy:     jetstream.AckExplicitPolicy,
		MaxDeliver:    maxDeliver,
	})
	if err != nil {
		return nil, fmt.Errorf("events: consumer %s: %w", durable, err)
	}

	cc, err := cons.Consume(func(m jetstream.Msg) { dispatch(ctx, h, m) })
	if err != nil {
		return nil, fmt.Errorf("events: consume %s: %w", durable, err)
	}
	return cc.Stop, nil
}
