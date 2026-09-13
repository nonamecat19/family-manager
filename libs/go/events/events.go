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

type Subject string

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

	SubjectNotesNoteCreated  Subject = "notes.note.created"
	SubjectNotesNoteUpdated  Subject = "notes.note.updated"
	SubjectNotesShareGranted Subject = "notes.share.granted"
)

var ErrBadSubject = errors.New("events: subject must be <domain>.<entity>.<verb>")

func (s Subject) Domain() string {
	parts := strings.Split(string(s), ".")
	if len(parts) == 0 {
		return ""
	}
	return parts[0]
}

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

type Config struct {
	URL            string
	Name           string
	ConnectTimeout time.Duration
}

type Bus struct {
	conn *nats.Conn
	js   jetstream.JetStream
}

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

func (b *Bus) Close() {
	if b.conn != nil {
		_ = b.conn.Drain()
	}
}

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

func (b *Bus) Publish(ctx context.Context, subject Subject, msg proto.Message) error {
	if err := subject.Validate(); err != nil {
		return err
	}
	payload, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("events: marshal %s: %w", subject, err)
	}
	m := &nats.Msg{Subject: string(subject), Data: payload}
	if id := logger.RequestID(ctx); id != "" {
		m.Header = nats.Header{RequestIDHeader: []string{id}}
	}
	if _, err := b.js.PublishMsg(ctx, m); err != nil {
		return fmt.Errorf("events: publish %s: %w", subject, err)
	}
	return nil
}

const RequestIDHeader = "X-Request-Id"

const maxDeliver = 5

var redeliveryBackoff = []time.Duration{
	1 * time.Second,
	5 * time.Second,
	30 * time.Second,
	2 * time.Minute,
}

type ackable interface {
	Subject() string
	Data() []byte
	Headers() nats.Header
	Ack() error
	NakWithDelay(delay time.Duration) error
	Metadata() (*jetstream.MsgMetadata, error)
}

func dispatch(ctx context.Context, h Handler, m ackable) {
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

type Handler func(ctx context.Context, subject Subject, payload []byte) error

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
