package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/services/telegram/db"
	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

const (
	errorBackoff   = 2 * time.Second
	maxConcurrency = 16
)

type Dispatcher interface {
	Name() string
	Handle(ctx context.Context, update telegram.Update)
	API() *telegram.Client
}

type Offsets interface {
	GetBotOffset(ctx context.Context, bot string) (int64, error)
	SetBotOffset(ctx context.Context, arg db.SetBotOffsetParams) error
}

func Poll(
	ctx context.Context, bot Dispatcher, offsets Offsets, timeout time.Duration, log *slog.Logger,
) error {
	log = log.With(slog.String("bot", bot.Name()))

	if err := bot.API().DeleteWebhook(ctx); err != nil {
		log.WarnContext(ctx, "delete webhook before polling", slog.String("error", err.Error()))
	}

	offset, err := offsets.GetBotOffset(ctx, bot.Name())
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	for {
		if ctx.Err() != nil {
			return nil
		}

		updates, err := bot.API().GetUpdates(ctx, offset, timeout)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			wait := errorBackoff
			if after, ok := telegram.RetryAfter(err); ok {
				wait = after
			}
			log.WarnContext(ctx, "getUpdates", slog.String("error", err.Error()))
			if !sleep(ctx, wait) {
				return nil
			}
			continue
		}

		for _, update := range updates {
			bot.Handle(ctx, update)
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
		}

		if len(updates) == 0 {
			continue
		}
		if err := offsets.SetBotOffset(ctx, db.SetBotOffsetParams{
			Bot: bot.Name(), OffsetID: offset,
		}); err != nil {
			log.WarnContext(ctx, "persist offset", slog.String("error", err.Error()))
		}
	}
}

func sleep(ctx context.Context, d time.Duration) bool {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

type Webhook struct {
	bots    map[string]Dispatcher
	secret  string
	log     *slog.Logger
	root    context.Context
	slots   chan struct{}
	pending sync.WaitGroup
}

func NewWebhook(root context.Context, bots map[string]Dispatcher, secret string, log *slog.Logger) *Webhook {
	return &Webhook{
		bots:   bots,
		secret: secret,
		log:    log,
		root:   root,
		slots:  make(chan struct{}, maxConcurrency),
	}
}

func WebhookPath(bot string) string { return "/tg/" + bot }

func (w *Webhook) Register(mux *http.ServeMux) {
	mux.Handle("POST /tg/{bot}", w)
}

func (w *Webhook) Wait() { w.pending.Wait() }

func (w *Webhook) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	bot, ok := w.bots[r.PathValue("bot")]
	if !ok {
		http.NotFound(rw, r)
		return
	}
	if r.Header.Get("X-Telegram-Bot-Api-Secret-Token") != w.secret {
		http.Error(rw, "forbidden", http.StatusForbidden)
		return
	}

	var update telegram.Update
	if err := json.NewDecoder(http.MaxBytesReader(rw, r.Body, 1<<20)).Decode(&update); err != nil {
		http.Error(rw, "bad request", http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)

	select {
	case w.slots <- struct{}{}:
	default:
		w.log.WarnContext(r.Context(), "webhook queue is full; dropping update",
			slog.String("bot", bot.Name()), slog.Int64("update_id", update.UpdateID))
		return
	}

	w.pending.Add(1)
	go func() {
		defer w.pending.Done()
		defer func() { <-w.slots }()
		bot.Handle(w.root, update)
	}()
}
