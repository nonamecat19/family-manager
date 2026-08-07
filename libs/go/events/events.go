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
	SubjectFinanceBudgetExceeded     Subject = "finance.budget.exceeded"
	SubjectFinanceRecurringDue       Subject = "finance.recurring.due"
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
	if _, err := b.js.Publish(ctx, string(subject), payload); err != nil {
		return fmt.Errorf("events: publish %s: %w", subject, err)
	}
	return nil
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
		MaxDeliver:    5,
	})
	if err != nil {
		return nil, fmt.Errorf("events: consumer %s: %w", durable, err)
	}

	cc, err := cons.Consume(func(m jetstream.Msg) {
		if err := h(ctx, Subject(m.Subject()), m.Data()); err != nil {
			_ = m.Nak()
			return
		}
		_ = m.Ack()
	})
	if err != nil {
		return nil, fmt.Errorf("events: consume %s: %w", durable, err)
	}
	return cc.Stop, nil
}
