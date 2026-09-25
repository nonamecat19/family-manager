package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/nnc/family-manager/services/telegram/db"
)

const stateTTL = 10 * time.Minute

type States interface {
	UpsertChatState(ctx context.Context, arg db.UpsertChatStateParams) (db.ChatState, error)
	GetChatState(ctx context.Context, arg db.GetChatStateParams) (db.ChatState, error)
	ClearChatState(ctx context.Context, arg db.ClearChatStateParams) (int64, error)
}

type State struct {
	Kind    string
	Payload map[string]string
}

func (c *Context) SetState(ctx context.Context, kind string, payload map[string]string) error {
	if c.Bot.states == nil {
		return errors.New("bot: no state store configured")
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("bot: encode state: %w", err)
	}
	_, err = c.Bot.states.UpsertChatState(ctx, db.UpsertChatStateParams{
		Bot:            c.Bot.name,
		TelegramUserID: c.From.ID,
		Kind:           kind,
		Payload:        body,
		ExpiresAt:      timestamp(time.Now().Add(stateTTL)),
	})
	return err
}

func (c *Context) State(ctx context.Context) (State, bool) {
	if c.Bot.states == nil {
		return State{}, false
	}
	row, err := c.Bot.states.GetChatState(ctx, db.GetChatStateParams{
		Bot: c.Bot.name, TelegramUserID: c.From.ID,
	})
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			c.Bot.log.WarnContext(ctx, "read chat state", "error", err.Error())
		}
		return State{}, false
	}

	payload := map[string]string{}
	if len(row.Payload) > 0 {
		if err := json.Unmarshal(row.Payload, &payload); err != nil {
			return State{}, false
		}
	}
	return State{Kind: row.Kind, Payload: payload}, true
}

func (c *Context) ClearState(ctx context.Context) {
	if c.Bot.states == nil {
		return
	}
	if _, err := c.Bot.states.ClearChatState(ctx, db.ClearChatStateParams{
		Bot: c.Bot.name, TelegramUserID: c.From.ID,
	}); err != nil {
		c.Bot.log.WarnContext(ctx, "clear chat state", "error", err.Error())
	}
}
