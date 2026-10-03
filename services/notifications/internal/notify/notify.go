package notify

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/services/notifications/db"
	"github.com/nnc/family-manager/services/notifications/internal/expo"
	"github.com/nnc/family-manager/services/notifications/internal/topics"
)

const (
	receiptRetention   = 24 * time.Hour
	processedRetention = 60 * 24 * time.Hour
)

var Subjects = []events.Subject{
	events.SubjectFamilyMemberJoined,
	events.SubjectFamilyMemberRemoved,
	events.SubjectFinanceBudgetExceeded,
	events.SubjectRecipesRecipeCreated,
}

type Bus interface {
	Subscribe(ctx context.Context, subject events.Subject, durable string, h events.Handler) (func(), error)
}

type Sender interface {
	Send(ctx context.Context, msgs []expo.Message) ([]expo.Ticket, error)
	Receipts(ctx context.Context, ids []string) (map[string]expo.Receipt, error)
}

type Families interface {
	FamilyOf(ctx context.Context, userID string) (string, error)
}

type Options struct {
	Queries      db.Querier
	Sender       Sender
	Families     Families
	Log          *slog.Logger
	Now          func() time.Time
	ReceiptDelay time.Duration
}

type Notifier struct {
	q            db.Querier
	sender       Sender
	families     Families
	log          *slog.Logger
	now          func() time.Time
	receiptDelay time.Duration
}

func New(opts Options) *Notifier {
	n := &Notifier{
		q:            opts.Queries,
		sender:       opts.Sender,
		families:     opts.Families,
		log:          opts.Log,
		now:          opts.Now,
		receiptDelay: opts.ReceiptDelay,
	}
	if n.log == nil {
		n.log = slog.Default()
	}
	if n.now == nil {
		n.now = time.Now
	}
	if n.receiptDelay == 0 {
		n.receiptDelay = 15 * time.Minute
	}
	return n
}

func (n *Notifier) Subscribe(ctx context.Context, bus Bus) (func(), error) {
	var stops []func()
	stopAll := func() {
		for _, stop := range stops {
			stop()
		}
	}
	for _, subject := range Subjects {
		durable := "notifications-" + strings.ReplaceAll(string(subject), ".", "-")
		stop, err := bus.Subscribe(ctx, subject, durable, n.Handle)
		if err != nil {
			stopAll()
			return nil, err
		}
		stops = append(stops, stop)
	}
	return stopAll, nil
}

func EventID(subject events.Subject, payload []byte) string {
	sum := sha256.New()
	sum.Write([]byte(subject))
	sum.Write([]byte{0})
	sum.Write(payload)
	return hex.EncodeToString(sum.Sum(nil))
}

type notice struct {
	topic    string
	title    string
	body     string
	data     map[string]string
	familyID string
	userID   string
	except   string
}

func (n *Notifier) Handle(ctx context.Context, subject events.Subject, payload []byte) error {
	id := EventID(subject, payload)
	done, err := n.q.IsEventProcessed(ctx, id)
	if err != nil {
		return fmt.Errorf("is event processed: %w", err)
	}
	if done {
		return nil
	}

	nt, err := n.interpret(ctx, subject, payload)
	if err != nil {
		return err
	}
	if nt != nil {
		if err := n.deliver(ctx, nt); err != nil {
			return err
		}
	}

	if err := n.q.MarkEventProcessed(ctx, db.MarkEventProcessedParams{
		EventID: id, Subject: string(subject),
	}); err != nil {
		return fmt.Errorf("mark event processed: %w", err)
	}
	return nil
}

func (n *Notifier) interpret(ctx context.Context, subject events.Subject, payload []byte) (*notice, error) {
	switch subject {
	case events.SubjectFamilyMemberJoined:
		var ev familyv1.MemberJoinedEvent
		if !n.decode(ctx, subject, payload, &ev) {
			return nil, nil
		}
		return n.memberJoined(ctx, &ev)
	case events.SubjectFamilyMemberRemoved:
		var ev familyv1.MemberRemovedEvent
		if !n.decode(ctx, subject, payload, &ev) {
			return nil, nil
		}
		return n.memberRemoved(ctx, &ev)
	case events.SubjectFinanceBudgetExceeded:
		var ev financev1.BudgetExceededEvent
		if !n.decode(ctx, subject, payload, &ev) {
			return nil, nil
		}
		return budgetExceeded(&ev), nil
	case events.SubjectRecipesRecipeCreated:
		var ev recipesv1.RecipeCreatedEvent
		if !n.decode(ctx, subject, payload, &ev) {
			return nil, nil
		}
		return recipeCreated(&ev), nil
	default:
		return nil, nil
	}
}

func (n *Notifier) decode(ctx context.Context, subject events.Subject, payload []byte, msg proto.Message) bool {
	if err := proto.Unmarshal(payload, msg); err != nil {
		n.log.ErrorContext(ctx, "bad event payload",
			slog.String("subject", string(subject)), slog.String("error", err.Error()))
		return false
	}
	return true
}

func (n *Notifier) memberJoined(ctx context.Context, ev *familyv1.MemberJoinedEvent) (*notice, error) {
	familyID, userID, ok := uuids(ev.GetFamilyId(), ev.GetUserId())
	if !ok {
		n.log.ErrorContext(ctx, "member joined: ids are not uuids")
		return nil, nil
	}
	if err := n.q.SetUserFamily(ctx, db.SetUserFamilyParams{UserID: userID, FamilyID: familyID}); err != nil {
		return nil, fmt.Errorf("set user family: %w", err)
	}
	return &notice{
		topic:    topics.FamilyMemberJoined,
		title:    "New family member",
		body:     "Someone new joined your family.",
		data:     map[string]string{"family_id": ev.GetFamilyId(), "user_id": ev.GetUserId()},
		familyID: ev.GetFamilyId(),
		except:   ev.GetUserId(),
	}, nil
}

func (n *Notifier) memberRemoved(ctx context.Context, ev *familyv1.MemberRemovedEvent) (*notice, error) {
	familyID, userID, ok := uuids(ev.GetFamilyId(), ev.GetUserId())
	if !ok {
		n.log.ErrorContext(ctx, "member removed: ids are not uuids")
		return nil, nil
	}
	if err := n.q.ClearUserFamily(ctx, db.ClearUserFamilyParams{UserID: userID, FamilyID: familyID}); err != nil {
		return nil, fmt.Errorf("clear user family: %w", err)
	}
	by := ev.GetRemovedByUserId()
	if by == "" || by == ev.GetUserId() {
		return nil, nil
	}
	return &notice{
		topic:  topics.FamilyMemberRemoved,
		title:  "Removed from family",
		body:   "A family admin removed you from the family.",
		data:   map[string]string{"family_id": ev.GetFamilyId()},
		userID: ev.GetUserId(),
	}, nil
}

func budgetExceeded(ev *financev1.BudgetExceededEvent) *notice {
	body := fmt.Sprintf("Spent %s of %s.", formatMoney(ev.GetSpent()), formatMoney(ev.GetLimit()))
	if share := ev.GetShare(); share > 0 {
		body = fmt.Sprintf("Spent %s of %s (%.0f%%).",
			formatMoney(ev.GetSpent()), formatMoney(ev.GetLimit()), share*100)
	}
	return &notice{
		topic:    topics.FinanceBudgetExceeded,
		title:    "Budget exceeded",
		body:     body,
		data:     map[string]string{"family_id": ev.GetFamilyId(), "budget_id": ev.GetBudgetId()},
		familyID: ev.GetFamilyId(),
	}
}

func recipeCreated(ev *recipesv1.RecipeCreatedEvent) *notice {
	return &notice{
		topic:    topics.RecipesRecipeCreated,
		title:    "New recipe",
		body:     "A new recipe was added to your family cookbook.",
		data:     map[string]string{"family_id": ev.GetFamilyId(), "recipe_id": ev.GetRecipeId()},
		familyID: ev.GetFamilyId(),
		except:   ev.GetAuthorUserId(),
	}
}

func formatMoney(m *financev1.Money) string {
	minor := m.GetAmountMinor()
	sign := ""
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	return strings.TrimSpace(fmt.Sprintf("%s%d.%02d %s", sign, minor/100, minor%100, m.GetCurrencyCode()))
}

func (n *Notifier) deliver(ctx context.Context, nt *notice) error {
	tokens, err := n.recipients(ctx, nt)
	if err != nil {
		return err
	}
	tokens, err = n.unmuted(ctx, nt.topic, tokens)
	if err != nil {
		return err
	}
	tokens = route(nt.topic, tokens)
	if len(tokens) == 0 {
		return nil
	}

	data := map[string]string{"topic": nt.topic}
	for k, v := range nt.data {
		data[k] = v
	}

	var dead []string
	for start := 0; start < len(tokens); start += expo.MaxBatch {
		chunk := tokens[start:min(start+expo.MaxBatch, len(tokens))]
		msgs := make([]expo.Message, len(chunk))
		for i, t := range chunk {
			msgs[i] = expo.Message{To: t.Token, Title: nt.title, Body: nt.body, Data: data, Sound: "default"}
		}
		tickets, err := n.sender.Send(ctx, msgs)
		if err != nil {
			return fmt.Errorf("send push: %w", err)
		}
		for i, ticket := range tickets {
			if i >= len(chunk) {
				break
			}
			switch {
			case ticket.Status == expo.StatusOK && ticket.ID != "":
				if err := n.q.InsertPushTicket(ctx, db.InsertPushTicketParams{
					ID: ticket.ID, Token: chunk[i].Token,
				}); err != nil {
					return fmt.Errorf("insert push ticket: %w", err)
				}
			case ticket.Error == expo.DeviceNotRegistered:
				dead = append(dead, chunk[i].Token)
			case ticket.Status == expo.StatusError:
				n.log.WarnContext(ctx, "push rejected",
					slog.String("topic", nt.topic), slog.String("error", ticket.Error),
					slog.String("message", ticket.Message))
			}
		}
	}

	if len(dead) > 0 {
		if _, err := n.q.DeletePushTokens(ctx, dead); err != nil {
			return fmt.Errorf("delete dead tokens: %w", err)
		}
		n.log.InfoContext(ctx, "dropped unregistered push tokens", slog.Int("count", len(dead)))
	}
	return nil
}

func (n *Notifier) recipients(ctx context.Context, nt *notice) ([]db.PushToken, error) {
	if nt.userID != "" {
		userID, err := pgconv.UUID(nt.userID)
		if err != nil {
			return nil, nil
		}
		tokens, err := n.q.ListUserPushTokens(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("list user push tokens: %w", err)
		}
		return tokens, nil
	}

	familyID, err := pgconv.UUID(nt.familyID)
	if err != nil {
		return nil, nil
	}
	rows, err := n.q.ListFamilyPushTokens(ctx, familyID)
	if err != nil {
		return nil, fmt.Errorf("list family push tokens: %w", err)
	}

	var out []db.PushToken
	verified := map[string]bool{}
	for _, t := range rows {
		user := pgconv.UUIDString(t.UserID)
		if user == nt.except {
			continue
		}
		member, seen := verified[user]
		if !seen {
			member, err = n.stillMember(ctx, t.UserID, nt.familyID)
			if err != nil {
				return nil, err
			}
			verified[user] = member
		}
		if member {
			out = append(out, t)
		}
	}
	return out, nil
}

func (n *Notifier) stillMember(ctx context.Context, userID pgtype.UUID, familyID string) (bool, error) {
	actual, err := n.families.FamilyOf(ctx, pgconv.UUIDString(userID))
	if err != nil {
		return false, fmt.Errorf("family of user: %w", err)
	}
	if actual == familyID {
		return true, nil
	}
	if actual == "" {
		family, _ := pgconv.UUID(familyID)
		err = n.q.ClearUserFamily(ctx, db.ClearUserFamilyParams{UserID: userID, FamilyID: family})
	} else if family, perr := pgconv.UUID(actual); perr == nil {
		err = n.q.SetUserFamily(ctx, db.SetUserFamilyParams{UserID: userID, FamilyID: family})
	}
	if err != nil {
		return false, fmt.Errorf("correct user family: %w", err)
	}
	return false, nil
}

func (n *Notifier) unmuted(ctx context.Context, topic string, tokens []db.PushToken) ([]db.PushToken, error) {
	if len(tokens) == 0 {
		return nil, nil
	}
	seen := map[string]bool{}
	var users []pgtype.UUID
	for _, t := range tokens {
		if key := pgconv.UUIDString(t.UserID); !seen[key] {
			seen[key] = true
			users = append(users, t.UserID)
		}
	}
	rows, err := n.q.ListMutesForUsers(ctx, users)
	if err != nil {
		return nil, fmt.Errorf("list mutes: %w", err)
	}
	mutes := map[string]map[string]bool{}
	for _, r := range rows {
		key := pgconv.UUIDString(r.UserID)
		if mutes[key] == nil {
			mutes[key] = map[string]bool{}
		}
		mutes[key][r.Topic] = true
	}

	out := tokens[:0:0]
	for _, t := range tokens {
		if !topics.Muted(mutes[pgconv.UUIDString(t.UserID)], topic) {
			out = append(out, t)
		}
	}
	return out, nil
}

var appRank = map[string]int{"finance": 0, "recipes": 1, "notes": 2}

func route(topic string, tokens []db.PushToken) []db.PushToken {
	domain := topics.Domain(topic)
	if _, isApp := appRank[domain]; isApp {
		out := tokens[:0:0]
		for _, t := range tokens {
			if t.App == domain {
				out = append(out, t)
			}
		}
		return out
	}

	best := map[string]int{}
	var out []db.PushToken
	for _, t := range tokens {
		device := t.DeviceID
		if device == "" {
			device = t.Token
		}
		key := pgconv.UUIDString(t.UserID) + "|" + device
		i, ok := best[key]
		if !ok {
			best[key] = len(out)
			out = append(out, t)
			continue
		}
		if appRank[t.App] < appRank[out[i].App] {
			out[i] = t
		}
	}
	return out
}

func (n *Notifier) CheckReceipts(ctx context.Context) error {
	now := n.now()
	due, err := n.q.ListDuePushTickets(ctx, db.ListDuePushTicketsParams{
		CreatedAt: pgconv.TimestampFrom(now.Add(-n.receiptDelay)),
		Limit:     expo.MaxReceiptBatch,
	})
	if err != nil {
		return fmt.Errorf("list due tickets: %w", err)
	}
	if len(due) == 0 {
		return nil
	}

	ids := make([]string, len(due))
	for i, t := range due {
		ids[i] = t.ID
	}
	receipts, err := n.sender.Receipts(ctx, ids)
	if err != nil {
		return fmt.Errorf("fetch receipts: %w", err)
	}

	var done, dead []string
	for _, t := range due {
		r, ok := receipts[t.ID]
		if !ok {
			if t.CreatedAt.Valid && now.Sub(t.CreatedAt.Time) > receiptRetention {
				done = append(done, t.ID)
			}
			continue
		}
		done = append(done, t.ID)
		if r.Status != expo.StatusError {
			continue
		}
		if r.Error == expo.DeviceNotRegistered {
			dead = append(dead, t.Token)
			continue
		}
		n.log.WarnContext(ctx, "push receipt error",
			slog.String("error", r.Error), slog.String("message", r.Message))
	}

	if len(dead) > 0 {
		if _, err := n.q.DeletePushTokens(ctx, dead); err != nil {
			return fmt.Errorf("delete dead tokens: %w", err)
		}
		n.log.InfoContext(ctx, "dropped unregistered push tokens", slog.Int("count", len(dead)))
	}
	if len(done) > 0 {
		if err := n.q.DeletePushTickets(ctx, done); err != nil {
			return fmt.Errorf("delete tickets: %w", err)
		}
	}
	return nil
}

func (n *Notifier) Prune(ctx context.Context) error {
	if _, err := n.q.PruneProcessedEvents(ctx,
		pgconv.TimestampFrom(n.now().Add(-processedRetention))); err != nil {
		return fmt.Errorf("prune processed events: %w", err)
	}
	return nil
}

func (n *Notifier) RunReceipts(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := n.CheckReceipts(ctx); err != nil && !errors.Is(err, context.Canceled) {
				n.log.WarnContext(ctx, "receipt check failed", slog.String("error", err.Error()))
			}
			if err := n.Prune(ctx); err != nil && !errors.Is(err, context.Canceled) {
				n.log.WarnContext(ctx, "prune failed", slog.String("error", err.Error()))
			}
		}
	}
}

func uuids(familyID, userID string) (pgtype.UUID, pgtype.UUID, bool) {
	f, err := pgconv.UUID(familyID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, false
	}
	u, err := pgconv.UUID(userID)
	if err != nil {
		return pgtype.UUID{}, pgtype.UUID{}, false
	}
	return f, u, true
}
