package notify

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"google.golang.org/protobuf/proto"

	"github.com/nnc/family-manager/libs/go/database/pgconv"
	"github.com/nnc/family-manager/libs/go/events"
	familyv1 "github.com/nnc/family-manager/sdk/go/family/v1"
	financev1 "github.com/nnc/family-manager/sdk/go/finance/v1"
	recipesv1 "github.com/nnc/family-manager/sdk/go/recipes/v1"
	"github.com/nnc/family-manager/services/notifications/db"
	"github.com/nnc/family-manager/services/notifications/internal/dbtest"
	"github.com/nnc/family-manager/services/notifications/internal/expo"
	"github.com/nnc/family-manager/services/notifications/internal/topics"
)

const (
	alice  = "11111111-1111-4111-8111-111111111111"
	bob    = "22222222-2222-4222-8222-222222222222"
	carol  = "33333333-3333-4333-8333-333333333333"
	family = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
	other  = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
)

var fixedNow = time.Date(2026, 10, 4, 12, 0, 0, 0, time.UTC)

type fixture struct {
	n        *Notifier
	store    *dbtest.Fake
	sender   *fakeSender
	families *fakeFamilies
	now      time.Time
}

func newFixture() *fixture {
	f := &fixture{
		store:    dbtest.New(),
		sender:   newFakeSender(),
		families: &fakeFamilies{of: map[string]string{}},
		now:      fixedNow,
	}
	f.store.Now = func() time.Time { return f.now }
	f.n = New(Options{
		Queries:      f.store,
		Sender:       f.sender,
		Families:     f.families,
		Now:          func() time.Time { return f.now },
		ReceiptDelay: 15 * time.Minute,
	})
	return f
}

func token(user, app, device string) string {
	return fmt.Sprintf("ExponentPushToken[%s-%s-%s]", user[:4], app, device)
}

func (f *fixture) member(user, familyID string, apps ...string) {
	f.families.of[user] = familyID
	for _, app := range apps {
		f.addToken(user, familyID, app, "phone")
	}
}

func (f *fixture) addToken(user, familyID, app, device string) string {
	tok := token(user, app, device)
	var fam pgtype.UUID
	if familyID != "" {
		fam = pgconv.MustUUID(familyID)
	}
	now := f.now
	f.now = now.Add(-time.Minute)
	_, _ = f.store.UpsertPushToken(context.Background(), db.UpsertPushTokenParams{
		Token: tok, UserID: pgconv.MustUUID(user), FamilyID: fam,
		Platform: "android", App: app, DeviceID: device,
	})
	f.now = now
	return tok
}

func (f *fixture) mute(user, topic string) {
	_ = f.store.InsertMute(context.Background(), db.InsertMuteParams{UserID: pgconv.MustUUID(user), Topic: topic})
}

func (f *fixture) handle(t *testing.T, subject events.Subject, msg proto.Message) error {
	t.Helper()
	payload, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return f.n.Handle(context.Background(), subject, payload)
}

func recipients(msgs []expo.Message) map[string]bool {
	out := map[string]bool{}
	for _, m := range msgs {
		out[m.To] = true
	}
	return out
}

func exceeded() *financev1.BudgetExceededEvent {
	return &financev1.BudgetExceededEvent{
		FamilyId: family,
		BudgetId: "budget-1",
		Limit:    &financev1.Money{AmountMinor: 100000, CurrencyCode: "UAH"},
		Spent:    &financev1.Money{AmountMinor: 123450, CurrencyCode: "UAH"},
		Share:    1.2345,
	}
}

func TestBudgetExceededReachesTheFinanceAppOfEveryMember(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance", "recipes")
	f.member(bob, family, "finance")
	f.member(carol, other, "finance")

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got := recipients(f.sender.sent())
	want := map[string]bool{token(alice, "finance", "phone"): true, token(bob, "finance", "phone"): true}
	if len(got) != len(want) {
		t.Fatalf("sent to %v, want %v", got, want)
	}
	for tok := range want {
		if !got[tok] {
			t.Errorf("missing %s", tok)
		}
	}

	msg := f.sender.sent()[0]
	if msg.Title != "Budget exceeded" || msg.Body != "Spent 1234.50 UAH of 1000.00 UAH (123%)." {
		t.Errorf("message = %q / %q", msg.Title, msg.Body)
	}
	if msg.Data["topic"] != topics.FinanceBudgetExceeded || msg.Data["budget_id"] != "budget-1" {
		t.Errorf("data = %v", msg.Data)
	}
	if len(f.store.Tickets) != 2 {
		t.Errorf("stored %d tickets, want 2", len(f.store.Tickets))
	}
}

func TestTheSameEventIsDeliveredOnce(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")

	ev := exceeded()
	for range 3 {
		if err := f.handle(t, events.SubjectFinanceBudgetExceeded, ev); err != nil {
			t.Fatalf("Handle: %v", err)
		}
	}
	if n := len(f.sender.sent()); n != 1 {
		t.Fatalf("sent %d pushes for one event, want 1", n)
	}
}

func TestMutedDomainsAndTopicsAreSkipped(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance", "recipes")
	f.member(bob, family, "finance", "recipes")
	f.member(carol, family, "recipes")
	f.mute(alice, "finance")
	f.mute(bob, topics.RecipesRecipeCreated)

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if err := f.handle(t, events.SubjectRecipesRecipeCreated, &recipesv1.RecipeCreatedEvent{
		FamilyId: family, RecipeId: "r1", AuthorUserId: carol,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got := recipients(f.sender.sent())
	want := map[string]bool{
		token(bob, "finance", "phone"):   true,
		token(alice, "recipes", "phone"): true,
	}
	if len(got) != len(want) {
		t.Fatalf("sent to %v, want %v", got, want)
	}
	for tok := range want {
		if !got[tok] {
			t.Errorf("missing %s", tok)
		}
	}
}

func TestRecipeCreatedSkipsTheAuthor(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "recipes")
	f.member(bob, family, "recipes")

	if err := f.handle(t, events.SubjectRecipesRecipeCreated, &recipesv1.RecipeCreatedEvent{
		FamilyId: family, RecipeId: "r1", AuthorUserId: alice,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	got := recipients(f.sender.sent())
	if len(got) != 1 || !got[token(bob, "recipes", "phone")] {
		t.Fatalf("sent to %v, want only bob", got)
	}
}

func TestAFormerMemberIsNotNotifiedAndTheirTokensAreCorrected(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.member(bob, family, "finance")
	f.families.of[bob] = ""

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	got := recipients(f.sender.sent())
	if len(got) != 1 || !got[token(alice, "finance", "phone")] {
		t.Fatalf("sent to %v, want only alice", got)
	}
	if f.store.Tokens[token(bob, "finance", "phone")].FamilyID.Valid {
		t.Error("bob's token still points at the family he left")
	}
}

func TestFamilyServiceOutageIsRetried(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.families.fail = errors.New("family unavailable")

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err == nil {
		t.Fatal("an unresolvable audience was acked")
	}
	if len(f.store.Events) != 0 {
		t.Fatal("a failed event kept its claim")
	}

	f.families.fail = nil
	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(f.sender.sent()) != 1 {
		t.Fatalf("sent %d after retry, want 1", len(f.sender.sent()))
	}
}

func TestSendFailureIsRetriedNotAcked(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.sender.fail = errors.New("expo 503")

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err == nil {
		t.Fatal("a failed send was acked")
	}
	if len(f.store.Events) != 0 {
		t.Fatal("a failed send kept its claim")
	}

	f.sender.fail = nil
	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("retry: %v", err)
	}
	if len(f.sender.sent()) != 1 {
		t.Fatalf("sent %d after retry, want 1", len(f.sender.sent()))
	}
}

func TestAnEventInFlightElsewhereIsNotSentTwice(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	payload, _ := proto.Marshal(exceeded())
	id := EventID(events.SubjectFinanceBudgetExceeded, payload)
	_, _ = f.store.ClaimEvent(context.Background(), db.ClaimEventParams{
		EventID: id, Subject: string(events.SubjectFinanceBudgetExceeded),
	})

	err := f.n.Handle(context.Background(), events.SubjectFinanceBudgetExceeded, payload)
	if !errors.Is(err, errInFlight) {
		t.Fatalf("err = %v, want errInFlight so the bus redelivers later", err)
	}
	if len(f.sender.sent()) != 0 {
		t.Fatal("sent while another delivery held the claim")
	}

	f.now = f.now.Add(claimTimeout + time.Second)
	if err := f.n.Handle(context.Background(), events.SubjectFinanceBudgetExceeded, payload); err != nil {
		t.Fatalf("Handle after the claim went stale: %v", err)
	}
	if len(f.sender.sent()) != 1 {
		t.Fatalf("sent %d after taking over a stale claim, want 1", len(f.sender.sent()))
	}
	if !f.store.Processed(id) {
		t.Error("the event was not marked done")
	}
}

func TestAPartlyAcceptedDeliveryIsNotRetried(t *testing.T) {
	f := newFixture()
	f.member(alice, family)
	for i := range 150 {
		f.addToken(alice, family, "finance", fmt.Sprintf("device-%03d", i))
	}
	f.sender.failFromBatch = 2

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v, want an ack once expo accepted a batch", err)
	}
	if n := len(f.sender.sent()); n != expo.MaxBatch {
		t.Fatalf("sent %d, want the first batch of %d", n, expo.MaxBatch)
	}

	f.sender.failFromBatch = 0
	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("redelivery: %v", err)
	}
	if n := len(f.sender.sent()); n != expo.MaxBatch {
		t.Fatalf("redelivery re-sent: %d total, want %d", n, expo.MaxBatch)
	}
}

func TestLargeAudiencesAreBatchedAtTheExpoLimit(t *testing.T) {
	f := newFixture()
	f.member(alice, family)
	for i := range 250 {
		f.addToken(alice, family, "finance", fmt.Sprintf("device-%03d", i))
	}

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(f.sender.batches) != 3 {
		t.Fatalf("batches = %d, want 3", len(f.sender.batches))
	}
	for i, b := range f.sender.batches {
		if len(b) > expo.MaxBatch {
			t.Errorf("batch %d has %d messages", i, len(b))
		}
	}
	if len(f.sender.sent()) != 250 {
		t.Errorf("sent %d, want 250", len(f.sender.sent()))
	}
}

func TestDeviceNotRegisteredTicketDeletesTheToken(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.member(bob, family, "finance")
	dead := token(bob, "finance", "phone")
	f.sender.dead[dead] = true

	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if _, ok := f.store.Tokens[dead]; ok {
		t.Fatal("an unregistered device kept its token")
	}
	if _, ok := f.store.Tokens[token(alice, "finance", "phone")]; !ok {
		t.Fatal("a live token was deleted")
	}
	if len(f.store.Tickets) != 1 {
		t.Errorf("tickets = %d, want 1", len(f.store.Tickets))
	}
}

func TestReceiptsDropUnregisteredDevices(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.member(bob, family, "finance")
	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	var bobTicket, aliceTicket string
	for id, tk := range f.store.Tickets {
		if tk.Token == token(bob, "finance", "phone") {
			bobTicket = id
		} else {
			aliceTicket = id
		}
	}

	if err := f.n.CheckReceipts(context.Background()); err != nil {
		t.Fatalf("CheckReceipts: %v", err)
	}
	if len(f.sender.asked) != 0 {
		t.Fatal("receipts were requested before the delay")
	}

	f.now = f.now.Add(20 * time.Minute)
	f.sender.receipts[bobTicket] = expo.Receipt{Status: expo.StatusError, Error: expo.DeviceNotRegistered}
	if err := f.n.CheckReceipts(context.Background()); err != nil {
		t.Fatalf("CheckReceipts: %v", err)
	}
	if _, ok := f.store.Tokens[token(bob, "finance", "phone")]; ok {
		t.Fatal("a DeviceNotRegistered receipt left the token in place")
	}
	if _, ok := f.store.Tickets[bobTicket]; ok {
		t.Error("a checked ticket was kept")
	}
	if _, ok := f.store.Tickets[aliceTicket]; !ok {
		t.Error("a ticket without a receipt yet was dropped early")
	}

	f.now = f.now.Add(25 * time.Hour)
	if err := f.n.CheckReceipts(context.Background()); err != nil {
		t.Fatalf("CheckReceipts: %v", err)
	}
	if len(f.store.Tickets) != 0 {
		t.Errorf("a ticket past Expo's receipt retention was kept: %v", f.store.Tickets)
	}
	if _, ok := f.store.Tokens[token(alice, "finance", "phone")]; !ok {
		t.Error("alice's token was dropped without a receipt saying so")
	}
}

func TestAReceiptDoesNotDeleteATokenRegisteredAgainSinceTheSend(t *testing.T) {
	f := newFixture()
	f.member(bob, family, "finance")
	if err := f.handle(t, events.SubjectFinanceBudgetExceeded, exceeded()); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	var ticket string
	for id := range f.store.Tickets {
		ticket = id
	}

	f.now = f.now.Add(5 * time.Minute)
	tok := token(bob, "finance", "phone")
	_, _ = f.store.UpsertPushToken(context.Background(), db.UpsertPushTokenParams{
		Token: tok, UserID: pgconv.MustUUID(bob), FamilyID: pgconv.MustUUID(family),
		Platform: "android", App: "finance", DeviceID: "phone",
	})

	f.now = f.now.Add(20 * time.Minute)
	f.sender.receipts[ticket] = expo.Receipt{Status: expo.StatusError, Error: expo.DeviceNotRegistered}
	if err := f.n.CheckReceipts(context.Background()); err != nil {
		t.Fatalf("CheckReceipts: %v", err)
	}
	if _, ok := f.store.Tokens[tok]; !ok {
		t.Fatal("a token registered after the send was deleted by its stale receipt")
	}
	if _, ok := f.store.Tickets[ticket]; ok {
		t.Error("the checked ticket was kept")
	}
}

func TestMemberJoinedTellsTheOthersOncePerDevice(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "recipes", "finance", "notes")
	f.addToken(alice, family, "notes", "laptop")
	f.member(bob, "")
	bobTok := f.addToken(bob, "", "finance", "phone")
	f.families.of[bob] = family

	if err := f.handle(t, events.SubjectFamilyMemberJoined, &familyv1.MemberJoinedEvent{
		FamilyId: family, UserId: bob, Role: familyv1.Role_ROLE_MEMBER,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	got := recipients(f.sender.sent())
	want := map[string]bool{token(alice, "finance", "phone"): true, token(alice, "notes", "laptop"): true}
	if len(got) != len(want) {
		t.Fatalf("sent to %v, want %v", got, want)
	}
	for tok := range want {
		if !got[tok] {
			t.Errorf("missing %s", tok)
		}
	}
	if pgconv.UUIDString(f.store.Tokens[bobTok].FamilyID) != family {
		t.Error("the joining member's tokens were not attached to the family")
	}
}

func TestMemberRemovedByAnAdminTellsTheRemovedMember(t *testing.T) {
	f := newFixture()
	f.member(alice, family, "finance")
	f.member(bob, family, "notes")

	if err := f.handle(t, events.SubjectFamilyMemberRemoved, &familyv1.MemberRemovedEvent{
		FamilyId: family, UserId: bob, RemovedByUserId: alice,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	got := recipients(f.sender.sent())
	if len(got) != 1 || !got[token(bob, "notes", "phone")] {
		t.Fatalf("sent to %v, want only bob", got)
	}
	if f.store.Tokens[token(bob, "notes", "phone")].FamilyID.Valid {
		t.Error("a removed member's token still points at the family")
	}
}

func TestLeavingTheFamilySendsNothing(t *testing.T) {
	f := newFixture()
	f.member(bob, family, "notes")

	if err := f.handle(t, events.SubjectFamilyMemberRemoved, &familyv1.MemberRemovedEvent{
		FamilyId: family, UserId: bob, RemovedByUserId: bob,
	}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if n := len(f.sender.sent()); n != 0 {
		t.Fatalf("sent %d pushes for a voluntary leave", n)
	}
}

func TestBadPayloadIsAckedAndSkipped(t *testing.T) {
	f := newFixture()
	if err := f.n.Handle(context.Background(), events.SubjectFinanceBudgetExceeded, []byte{0xff, 0xff}); err != nil {
		t.Fatalf("Handle: %v", err)
	}
	if len(f.store.Events) != 1 {
		t.Error("a poison message was not recorded as processed")
	}
}

func TestPruneDropsOldProcessedEvents(t *testing.T) {
	f := newFixture()
	done := func(id string) {
		_, _ = f.store.ClaimEvent(context.Background(), db.ClaimEventParams{EventID: id, Subject: "x.y.z"})
		_ = f.store.CompleteEvent(context.Background(), id)
	}
	done("old")
	f.now = f.now.Add(61 * 24 * time.Hour)
	done("new")

	if err := f.n.Prune(context.Background()); err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if _, ok := f.store.Events["old"]; ok {
		t.Error("an event past retention was kept")
	}
	if !f.store.Processed("new") {
		t.Error("a recent event was pruned")
	}
}

type fakeBus struct {
	durables  map[events.Subject]string
	heartbeat map[events.Subject]time.Duration
	failOn    events.Subject
	stopped   int
}

func (b *fakeBus) SubscribeWith(
	_ context.Context, subject events.Subject, durable string, _ events.Handler, opts events.SubscribeOptions,
) (func(), error) {
	if subject == b.failOn {
		return nil, errors.New("no stream")
	}
	b.durables[subject] = durable
	if b.heartbeat != nil {
		b.heartbeat[subject] = opts.Heartbeat
	}
	return func() { b.stopped++ }, nil
}

func TestSubscribeBindsOneDurablePerSubject(t *testing.T) {
	f := newFixture()
	bus := &fakeBus{durables: map[events.Subject]string{}, heartbeat: map[events.Subject]time.Duration{}}
	stop, err := f.n.Subscribe(context.Background(), bus)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	if len(bus.durables) != len(Subjects) {
		t.Fatalf("durables = %v", bus.durables)
	}
	if got := bus.durables[events.SubjectFinanceBudgetExceeded]; got != "notifications-finance-budget-exceeded" {
		t.Errorf("durable = %q", got)
	}
	for subject, every := range bus.heartbeat {
		if every <= 0 || every >= handleTimeout {
			t.Errorf("%s heartbeat = %v, want a positive interval under the handler timeout", subject, every)
		}
	}
	stop()
	if bus.stopped != len(Subjects) {
		t.Errorf("stopped %d of %d", bus.stopped, len(Subjects))
	}
}

func TestSubscribeUnwindsOnFailure(t *testing.T) {
	f := newFixture()
	bus := &fakeBus{durables: map[events.Subject]string{}, failOn: events.SubjectRecipesRecipeCreated}
	if _, err := f.n.Subscribe(context.Background(), bus); err == nil {
		t.Fatal("Subscribe succeeded with a failing subject")
	}
	if bus.stopped != len(Subjects)-1 {
		t.Errorf("stopped %d consumers, want %d", bus.stopped, len(Subjects)-1)
	}
}
