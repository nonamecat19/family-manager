package telegram

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const secretToken = "8973034124:AAEH0IbhTxgFEM-supersecret"

func TestTransportErrorsHideTheBotToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			t.Error("test server cannot hijack")
			return
		}
		conn, _, err := hijacker.Hijack()
		if err != nil {
			t.Error(err)
			return
		}
		_ = conn.Close()
	}))
	defer server.Close()

	client, err := New(Options{Token: secretToken, APIBase: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.GetMe(context.Background())
	if err == nil {
		t.Fatal("expected a transport error")
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Fatalf("the bot token leaked into %q", err)
	}
	if !strings.Contains(err.Error(), redacted) {
		t.Fatalf("error %q does not mark the redaction", err)
	}
}

func TestDecodeErrorsHideTheBotToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("<html>not json</html>"))
	}))
	defer server.Close()

	client, err := New(Options{Token: secretToken, APIBase: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	if _, err := client.GetMe(context.Background()); err == nil {
		t.Fatal("expected a decode error")
	} else if strings.Contains(err.Error(), secretToken) {
		t.Fatalf("the bot token leaked into %q", err)
	}
}

func TestAPIErrorsCarryRetryAfter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"ok":false,"error_code":429,"description":"Too Many Requests",` +
			`"parameters":{"retry_after":7}}`))
	}))
	defer server.Close()

	client, err := New(Options{Token: secretToken, APIBase: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	_, err = client.GetMe(context.Background())
	wait, ok := RetryAfter(err)
	if !ok || wait != 7*time.Second {
		t.Fatalf("RetryAfter = %v, %v; want 7s, true", wait, ok)
	}
	if strings.Contains(err.Error(), secretToken) {
		t.Fatalf("the bot token leaked into %q", err)
	}
}

func TestSendMessageRoundTrip(t *testing.T) {
	var gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		gotBody = string(buf)
		_, _ = w.Write([]byte(`{"ok":true,"result":{"message_id":12,` +
			`"chat":{"id":99,"type":"private"},"text":"hi"}}`))
	}))
	defer server.Close()

	client, err := New(Options{Token: "abc", APIBase: server.URL})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	msg, err := client.SendMessage(context.Background(), SendMessageParams{ChatID: 99, Text: "hi"})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if msg.MessageID != 12 || msg.Chat.ID != 99 {
		t.Fatalf("message = %+v", msg)
	}
	if gotPath != "/botabc/sendMessage" {
		t.Fatalf("path = %q", gotPath)
	}
	if !strings.Contains(gotBody, `"chat_id":99`) {
		t.Fatalf("body = %q", gotBody)
	}
}
