package expo

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSendMapsTicketsInOrder(t *testing.T) {
	var got []Message
	var auth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/send" {
			t.Errorf("path = %q", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		_ = json.NewDecoder(r.Body).Decode(&got)
		_, _ = w.Write([]byte(`{"data":[
			{"status":"ok","id":"t1"},
			{"status":"error","message":"gone","details":{"error":"DeviceNotRegistered"}}
		]}`))
	}))
	defer srv.Close()

	c := New(srv.URL, "secret", 0)
	tickets, err := c.Send(context.Background(), []Message{
		{To: "ExponentPushToken[a]", Title: "hi"},
		{To: "ExponentPushToken[b]", Title: "hi"},
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if auth != "Bearer secret" {
		t.Errorf("authorization = %q", auth)
	}
	if len(got) != 2 || got[1].To != "ExponentPushToken[b]" {
		t.Fatalf("server got %+v", got)
	}
	if tickets[0].Status != StatusOK || tickets[0].ID != "t1" {
		t.Errorf("ticket 0 = %+v", tickets[0])
	}
	if tickets[1].Status != StatusError || tickets[1].Error != DeviceNotRegistered {
		t.Errorf("ticket 1 = %+v", tickets[1])
	}
}

func TestSendRejectsOversizedBatch(t *testing.T) {
	c := New("http://unused.invalid", "", 0)
	if _, err := c.Send(context.Background(), make([]Message, MaxBatch+1)); err == nil {
		t.Fatal("a batch over the limit was sent")
	}
}

func TestSendSurfacesRequestErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"errors":[{"code":"PUSH_TOO_MANY_EXPERIENCE_IDS","message":"mixed"}]}`))
	}))
	defer srv.Close()

	_, err := New(srv.URL, "", 0).Send(context.Background(), []Message{{To: "x"}})
	if err == nil || !strings.Contains(err.Error(), "PUSH_TOO_MANY_EXPERIENCE_IDS") {
		t.Fatalf("err = %v", err)
	}
}

func TestSendFailsOnServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "overloaded", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	if _, err := New(srv.URL, "", 0).Send(context.Background(), []Message{{To: "x"}}); err == nil {
		t.Fatal("a 503 was treated as success")
	}
}

func TestReceipts(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/getReceipts" {
			t.Errorf("path = %q", r.URL.Path)
		}
		var body struct {
			IDs []string `json:"ids"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if len(body.IDs) != 2 {
			t.Errorf("ids = %v", body.IDs)
		}
		_, _ = w.Write([]byte(`{"data":{
			"t1":{"status":"ok"},
			"t2":{"status":"error","details":{"error":"DeviceNotRegistered"}}
		}}`))
	}))
	defer srv.Close()

	got, err := New(srv.URL, "", 0).Receipts(context.Background(), []string{"t1", "t2"})
	if err != nil {
		t.Fatalf("Receipts: %v", err)
	}
	if got["t1"].Status != StatusOK || got["t2"].Error != DeviceNotRegistered {
		t.Fatalf("receipts = %+v", got)
	}
}
