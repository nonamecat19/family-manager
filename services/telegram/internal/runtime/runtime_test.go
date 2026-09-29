package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type recorder struct {
	name string

	mu   sync.Mutex
	seen []int64
	done chan struct{}
}

func newRecorder(name string) *recorder {
	return &recorder{name: name, done: make(chan struct{}, 8)}
}

func (r *recorder) Name() string { return r.name }

func (r *recorder) API() *telegram.Client { return nil }

func (r *recorder) Handle(_ context.Context, update telegram.Update) {
	r.mu.Lock()
	r.seen = append(r.seen, update.UpdateID)
	r.mu.Unlock()
	r.done <- struct{}{}
}

func (r *recorder) updates() []int64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]int64{}, r.seen...)
}

type offsetStore struct {
	mu      sync.Mutex
	offsets map[string]int64
}

func (o *offsetStore) GetBotOffset(_ context.Context, bot string) (int64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()
	offset, ok := o.offsets[bot]
	if !ok {
		return 0, pgx.ErrNoRows
	}
	return offset, nil
}

func (o *offsetStore) SetBotOffset(_ context.Context, arg db.SetBotOffsetParams) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.offsets[arg.Bot] = arg.OffsetID
	return nil
}

func post(t *testing.T, handler http.Handler, path, secret string, update telegram.Update) *httptest.ResponseRecorder {
	t.Helper()
	body, err := json.Marshal(update)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	if secret != "" {
		req.Header.Set("X-Telegram-Bot-Api-Secret-Token", secret)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return rec
}

func newWebhookMux(t *testing.T, bots map[string]Dispatcher) (*Webhook, http.Handler) {
	t.Helper()
	hook := NewWebhook(context.Background(), bots, "s3cret", slog.Default())
	mux := http.NewServeMux()
	hook.Register(mux)
	return hook, mux
}

func TestWebhookDeliversToTheRightBot(t *testing.T) {
	finance, notes := newRecorder("finance"), newRecorder("notes")
	hook, mux := newWebhookMux(t, map[string]Dispatcher{"finance": finance, "notes": notes})

	rec := post(t, mux, WebhookPath("finance"), "s3cret", telegram.Update{UpdateID: 7})
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	select {
	case <-finance.done:
	case <-time.After(2 * time.Second):
		t.Fatal("the finance bot never saw the update")
	}
	hook.Wait()

	if got := finance.updates(); len(got) != 1 || got[0] != 7 {
		t.Fatalf("finance saw %v", got)
	}
	if got := notes.updates(); len(got) != 0 {
		t.Fatalf("notes saw %v, want nothing", got)
	}
}

func TestWebhookRejectsAWrongSecret(t *testing.T) {
	finance := newRecorder("finance")
	hook, mux := newWebhookMux(t, map[string]Dispatcher{"finance": finance})

	rec := post(t, mux, WebhookPath("finance"), "guess", telegram.Update{UpdateID: 7})
	hook.Wait()

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
	if got := finance.updates(); len(got) != 0 {
		t.Fatalf("the update was handled anyway: %v", got)
	}
}

func TestWebhookRejectsAMissingSecret(t *testing.T) {
	finance := newRecorder("finance")
	hook, mux := newWebhookMux(t, map[string]Dispatcher{"finance": finance})

	rec := post(t, mux, WebhookPath("finance"), "", telegram.Update{UpdateID: 7})
	hook.Wait()

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rec.Code)
	}
}

func TestWebhookIsNotFoundForAnUnknownBot(t *testing.T) {
	hook, mux := newWebhookMux(t, map[string]Dispatcher{"finance": newRecorder("finance")})

	rec := post(t, mux, WebhookPath("shopping"), "s3cret", telegram.Update{UpdateID: 7})
	hook.Wait()

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}
}

func TestPollAdvancesAndPersistsTheOffset(t *testing.T) {
	var mu sync.Mutex
	var offsets []int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case hasSuffix(r.URL.Path, "deleteWebhook"):
			writeOK(w, `true`)
		case hasSuffix(r.URL.Path, "getUpdates"):
			var params struct {
				Offset int64 `json:"offset"`
			}
			_ = json.NewDecoder(r.Body).Decode(&params)

			mu.Lock()
			offsets = append(offsets, params.Offset)
			round := len(offsets)
			mu.Unlock()

			if round == 1 {
				writeOK(w, `[{"update_id":10},{"update_id":11}]`)
				return
			}
			writeOK(w, `[]`)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	api, err := telegram.New(telegram.Options{Token: "test-token", APIBase: server.URL})
	if err != nil {
		t.Fatalf("telegram.New: %v", err)
	}

	bot := &pollBot{recorder: newRecorder("finance"), api: api}
	store := &offsetStore{offsets: map[string]int64{}}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Poll(ctx, bot, store, time.Millisecond, slog.Default()) }()

	waitFor(t, func() bool {
		store.mu.Lock()
		defer store.mu.Unlock()
		return store.offsets["finance"] == 12
	}, "the offset was never persisted as 12")

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Poll: %v", err)
	}

	if got := bot.recorder.updates(); len(got) != 2 || got[0] != 10 || got[1] != 11 {
		t.Fatalf("handled %v, want both updates once", got)
	}

	mu.Lock()
	defer mu.Unlock()
	if offsets[0] != 0 {
		t.Fatalf("first poll used offset %d, want 0", offsets[0])
	}
	if len(offsets) > 1 && offsets[1] != 12 {
		t.Fatalf("second poll used offset %d, want 12", offsets[1])
	}
}

func TestPollStartsFromTheStoredOffset(t *testing.T) {
	seen := make(chan int64, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hasSuffix(r.URL.Path, "getUpdates") {
			var params struct {
				Offset int64 `json:"offset"`
			}
			_ = json.NewDecoder(r.Body).Decode(&params)
			seen <- params.Offset
			writeOK(w, `[]`)
			return
		}
		writeOK(w, `true`)
	}))
	defer server.Close()

	api, err := telegram.New(telegram.Options{Token: "test-token", APIBase: server.URL})
	if err != nil {
		t.Fatalf("telegram.New: %v", err)
	}

	store := &offsetStore{offsets: map[string]int64{"finance": 500}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		_ = Poll(ctx, &pollBot{recorder: newRecorder("finance"), api: api}, store, time.Millisecond, slog.Default())
	}()

	select {
	case offset := <-seen:
		if offset != 500 {
			t.Fatalf("offset = %d, want the stored 500", offset)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the poller never called getUpdates")
	}
}

type pollBot struct {
	*recorder
	api *telegram.Client
}

func (p *pollBot) API() *telegram.Client { return p.api }

func writeOK(w http.ResponseWriter, result string) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"result":` + result + `}`))
}

func hasSuffix(path, method string) bool {
	return len(path) >= len(method) && path[len(path)-len(method):] == method
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal(msg)
}
