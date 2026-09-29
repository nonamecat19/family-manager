package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultAPIBase = "https://api.telegram.org"
	redacted       = "<bot-token>"
)

var AllowedUpdates = []string{"message", "edited_message", "callback_query"}

type Client struct {
	token   string
	base    string
	http    *http.Client
	timeout time.Duration
}

type Options struct {
	Token      string
	APIBase    string
	HTTPClient *http.Client
	Timeout    time.Duration
}

func New(opts Options) (*Client, error) {
	if strings.TrimSpace(opts.Token) == "" {
		return nil, errors.New("telegram: empty bot token")
	}
	base := opts.APIBase
	if base == "" {
		base = defaultAPIBase
	}
	timeout := opts.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}
	client := opts.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: timeout}
	}
	return &Client{token: opts.Token, base: strings.TrimSuffix(base, "/"), http: client, timeout: timeout}, nil
}

type apiError struct {
	Code        int
	Description string
	RetryAfter  time.Duration
}

func (e *apiError) Error() string {
	return fmt.Sprintf("telegram: api error %d: %s", e.Code, e.Description)
}

func RetryAfter(err error) (time.Duration, bool) {
	var apiErr *apiError
	if errors.As(err, &apiErr) && apiErr.RetryAfter > 0 {
		return apiErr.RetryAfter, true
	}
	return 0, false
}

type envelope struct {
	OK          bool            `json:"ok"`
	Result      json.RawMessage `json:"result"`
	ErrorCode   int             `json:"error_code"`
	Description string          `json:"description"`
	Parameters  *struct {
		RetryAfter int `json:"retry_after"`
	} `json:"parameters,omitempty"`
}

func (c *Client) call(ctx context.Context, method string, params, out any) error {
	var body io.Reader
	if params != nil {
		encoded, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("telegram: encode %s: %w", method, err)
		}
		body = bytes.NewReader(encoded)
	}

	endpoint := c.base + "/bot" + url.PathEscape(c.token) + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return c.sanitize(fmt.Errorf("telegram: new request %s: %w", method, err))
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	res, err := c.http.Do(req)
	if err != nil {
		return c.sanitize(fmt.Errorf("telegram: %s: %w", method, err))
	}
	defer res.Body.Close()

	var env envelope
	if err := json.NewDecoder(res.Body).Decode(&env); err != nil {
		return c.sanitize(fmt.Errorf("telegram: decode %s: %w", method, err))
	}
	if !env.OK {
		apiErr := &apiError{Code: env.ErrorCode, Description: env.Description}
		if env.Parameters != nil && env.Parameters.RetryAfter > 0 {
			apiErr.RetryAfter = time.Duration(env.Parameters.RetryAfter) * time.Second
		}
		return apiErr
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(env.Result, out); err != nil {
		return fmt.Errorf("telegram: decode %s result: %w", method, err)
	}
	return nil
}

func (c *Client) sanitize(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	cleaned := strings.ReplaceAll(msg, c.token, redacted)
	if cleaned == msg {
		return err
	}
	return errors.New(cleaned)
}

func (c *Client) GetMe(ctx context.Context) (User, error) {
	var me User
	err := c.call(ctx, "getMe", nil, &me)
	return me, err
}

func (c *Client) GetUpdates(ctx context.Context, offset int64, timeout time.Duration) ([]Update, error) {
	var updates []Update
	err := c.call(ctx, "getUpdates", getUpdatesParams{
		Offset:         offset,
		Timeout:        int(timeout.Seconds()),
		AllowedUpdates: AllowedUpdates,
	}, &updates)
	return updates, err
}

func (c *Client) SendMessage(ctx context.Context, p SendMessageParams) (Message, error) {
	var sent Message
	err := c.call(ctx, "sendMessage", p, &sent)
	return sent, err
}

func (c *Client) EditMessageText(ctx context.Context, p EditMessageTextParams) error {
	err := c.call(ctx, "editMessageText", p, nil)
	if IsNotModified(err) {
		return nil
	}
	return err
}

func (c *Client) SetMyCommands(ctx context.Context, commands []BotCommand) error {
	return c.call(ctx, "setMyCommands", setMyCommandsParams{Commands: commands}, nil)
}

func IsNotModified(err error) bool {
	var apiErr *apiError
	return errors.As(err, &apiErr) && strings.Contains(apiErr.Description, "message is not modified")
}

func (c *Client) SetWebhook(ctx context.Context, hookURL, secret string) error {
	return c.call(ctx, "setWebhook", setWebhookParams{
		URL:            hookURL,
		SecretToken:    secret,
		AllowedUpdates: AllowedUpdates,
	}, nil)
}

func (c *Client) DeleteWebhook(ctx context.Context) error {
	return c.call(ctx, "deleteWebhook", deleteWebhookParams{}, nil)
}

func (c *Client) AnswerCallbackQuery(ctx context.Context, id, text string, alert bool) error {
	return c.call(ctx, "answerCallbackQuery", answerCallbackParams{
		CallbackQueryID: id, Text: text, ShowAlert: alert,
	}, nil)
}
