package events

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/nnc/family-manager/libs/go/logger"
)

func TestSubjectValidate(t *testing.T) {
	valid := []Subject{
		SubjectFamilyMemberInvited,
		SubjectFinanceTransactionCreated,
		SubjectFinanceRecurringPosted,
		SubjectRecipesRecipeCreated,
		SubjectRecipesMealPlanned,
	}
	for _, s := range valid {
		if err := s.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", s, err)
		}
	}

	invalid := []Subject{"", "recipes", "recipes.recipe", "recipes..created", "a.b.c.d"}
	for _, s := range invalid {
		if err := s.Validate(); !errors.Is(err, ErrBadSubject) {
			t.Errorf("Validate(%q) = %v, want ErrBadSubject", s, err)
		}
	}
}

func TestSubjectDomain(t *testing.T) {
	if got := SubjectRecipesRecipeCreated.Domain(); got != "recipes" {
		t.Errorf("Domain() = %q, want recipes", got)
	}
	if got := SubjectFamilyMemberJoined.Domain(); got != "family" {
		t.Errorf("Domain() = %q, want family", got)
	}
	if got := SubjectFinanceBudgetExceeded.Domain(); got != "finance" {
		t.Errorf("Domain() = %q, want finance", got)
	}
}

// fakeMsg is the minimum of jetstream.Msg that dispatch touches.
type fakeMsg struct {
	subject   string
	data      []byte
	headers   nats.Header
	delivered uint64
	noMeta    bool

	acked     bool
	nakDelays []time.Duration
}

func (m *fakeMsg) Subject() string { return m.subject }
func (m *fakeMsg) Data() []byte    { return m.data }

func (m *fakeMsg) Headers() nats.Header {
	if m.headers == nil {
		return nats.Header{}
	}
	return m.headers
}
func (m *fakeMsg) Ack() error { m.acked = true; return nil }

func (m *fakeMsg) NakWithDelay(d time.Duration) error {
	m.nakDelays = append(m.nakDelays, d)
	return nil
}

func (m *fakeMsg) Metadata() (*jetstream.MsgMetadata, error) {
	if m.noMeta {
		return nil, errors.New("not a jetstream message")
	}
	return &jetstream.MsgMetadata{NumDelivered: m.delivered}, nil
}

func TestDispatchAcksOnSuccess(t *testing.T) {
	m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), delivered: 1}
	dispatch(context.Background(), func(context.Context, Subject, []byte) error { return nil }, m)

	if !m.acked {
		t.Fatal("a handled message was not acked")
	}
	if len(m.nakDelays) != 0 {
		t.Fatalf("nacked a message the handler accepted: %v", m.nakDelays)
	}
}

// A bare Nak asks for immediate redelivery, so five attempts happened inside a few
// milliseconds and MaxDeliver was spent before the cause could possibly have cleared.
func TestDispatchBacksOffBeforeRedelivery(t *testing.T) {
	for attempt, want := range redeliveryBackoff {
		m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), delivered: uint64(attempt + 1)}
		dispatch(context.Background(), func(context.Context, Subject, []byte) error {
			return errors.New("database restarting")
		}, m)

		if m.acked {
			t.Fatal("acked a message the handler rejected")
		}
		if len(m.nakDelays) != 1 || m.nakDelays[0] != want {
			t.Fatalf("delivery %d nacked with %v, want %v", attempt+1, m.nakDelays, want)
		}
	}
}

func TestDispatchClampsTheBackoff(t *testing.T) {
	m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), delivered: 99}
	dispatch(context.Background(), func(context.Context, Subject, []byte) error {
		return errors.New("still broken")
	}, m)

	if want := redeliveryBackoff[len(redeliveryBackoff)-1]; m.nakDelays[0] != want {
		t.Fatalf("delay = %v, want the last step %v", m.nakDelays[0], want)
	}
}

func TestDispatchWithoutMetadataUsesTheFirstDelay(t *testing.T) {
	m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), noMeta: true}
	dispatch(context.Background(), func(context.Context, Subject, []byte) error {
		return errors.New("boom")
	}, m)

	if m.nakDelays[0] != redeliveryBackoff[0] {
		t.Fatalf("delay = %v, want %v", m.nakDelays[0], redeliveryBackoff[0])
	}
}

// An event a handler cannot parse must not be able to take the service down with it.
func TestDispatchSurvivesAPanickingHandler(t *testing.T) {
	m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), delivered: 1}
	dispatch(context.Background(), func(context.Context, Subject, []byte) error {
		panic("nil map read")
	}, m)

	if m.acked {
		t.Fatal("acked a message whose handler panicked")
	}
	if len(m.nakDelays) != 1 {
		t.Fatalf("panicking handler produced %d nak(s), want 1", len(m.nakDelays))
	}
}

// The whole point of putting the id on the message: the consumer's work belongs to the trace
// of the request that caused it.
func TestDispatchRestoresThePublishersRequestID(t *testing.T) {
	m := &fakeMsg{
		subject:   string(SubjectFamilyMemberJoined),
		delivered: 1,
		headers:   nats.Header{RequestIDHeader: []string{"abc123"}},
	}

	var seen string
	dispatch(context.Background(), func(ctx context.Context, _ Subject, _ []byte) error {
		seen = logger.RequestID(ctx)
		return nil
	}, m)

	if seen != "abc123" {
		t.Fatalf("request id in handler context = %q, want %q", seen, "abc123")
	}
}

// An event published outside a request carries no id, and the handler must not be handed an
// empty one that looks like a trace.
func TestDispatchWithoutARequestID(t *testing.T) {
	m := &fakeMsg{subject: string(SubjectFamilyMemberJoined), delivered: 1}

	var seen string
	dispatch(context.Background(), func(ctx context.Context, _ Subject, _ []byte) error {
		seen = logger.RequestID(ctx)
		return nil
	}, m)

	if seen != "" {
		t.Fatalf("request id = %q, want empty", seen)
	}
}
