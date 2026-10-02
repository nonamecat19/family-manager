package expo

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	MaxBatch            = 100
	MaxReceiptBatch     = 1000
	StatusOK            = "ok"
	StatusError         = "error"
	DeviceNotRegistered = "DeviceNotRegistered"
)

type Message struct {
	To    string            `json:"to"`
	Title string            `json:"title,omitempty"`
	Body  string            `json:"body,omitempty"`
	Data  map[string]string `json:"data,omitempty"`
	Sound string            `json:"sound,omitempty"`
}

type Ticket struct {
	ID      string
	Status  string
	Message string
	Error   string
}

type Receipt struct {
	Status  string
	Message string
	Error   string
}

type Client struct {
	baseURL     string
	accessToken string
	http        *http.Client
}

func New(baseURL, accessToken string, timeout time.Duration) *Client {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &Client{
		baseURL:     strings.TrimRight(baseURL, "/"),
		accessToken: accessToken,
		http:        &http.Client{Timeout: timeout},
	}
}

type details struct {
	Error string `json:"error"`
}

type wireResult struct {
	ID      string  `json:"id"`
	Status  string  `json:"status"`
	Message string  `json:"message"`
	Details details `json:"details"`
}

type wireError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (c *Client) Send(ctx context.Context, msgs []Message) ([]Ticket, error) {
	if len(msgs) == 0 {
		return nil, nil
	}
	if len(msgs) > MaxBatch {
		return nil, fmt.Errorf("expo: %d messages exceed the batch limit of %d", len(msgs), MaxBatch)
	}
	var out struct {
		Data   []wireResult `json:"data"`
		Errors []wireError  `json:"errors"`
	}
	if err := c.post(ctx, "/send", msgs, &out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("expo: send: %s: %s", out.Errors[0].Code, out.Errors[0].Message)
	}
	if len(out.Data) != len(msgs) {
		return nil, fmt.Errorf("expo: send: %d tickets for %d messages", len(out.Data), len(msgs))
	}
	tickets := make([]Ticket, len(out.Data))
	for i, r := range out.Data {
		tickets[i] = Ticket{ID: r.ID, Status: r.Status, Message: r.Message, Error: r.Details.Error}
	}
	return tickets, nil
}

func (c *Client) Receipts(ctx context.Context, ids []string) (map[string]Receipt, error) {
	if len(ids) == 0 {
		return map[string]Receipt{}, nil
	}
	if len(ids) > MaxReceiptBatch {
		return nil, fmt.Errorf("expo: %d receipt ids exceed the limit of %d", len(ids), MaxReceiptBatch)
	}
	var out struct {
		Data   map[string]wireResult `json:"data"`
		Errors []wireError           `json:"errors"`
	}
	if err := c.post(ctx, "/getReceipts", map[string][]string{"ids": ids}, &out); err != nil {
		return nil, err
	}
	if len(out.Errors) > 0 {
		return nil, fmt.Errorf("expo: receipts: %s: %s", out.Errors[0].Code, out.Errors[0].Message)
	}
	receipts := make(map[string]Receipt, len(out.Data))
	for id, r := range out.Data {
		receipts[id] = Receipt{Status: r.Status, Message: r.Message, Error: r.Details.Error}
	}
	return receipts, nil
}

func (c *Client) post(ctx context.Context, path string, body, out any) error {
	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("expo: marshal: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("expo: request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}

	res, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("expo: %s: %w", path, err)
	}
	defer func() { _ = res.Body.Close() }()

	raw, err := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("expo: read %s: %w", path, err)
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("expo: %s: status %d: %s", path, res.StatusCode, truncate(string(raw), 512))
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return fmt.Errorf("expo: decode %s: %w", path, err)
	}
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
