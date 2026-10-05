//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/nnc/family-manager/services/telegram/internal/i18n"
)

var (
	emuURL     = envOr("TGEMU_URL", "http://localhost:8090")
	authURL    = envOr("AUTH_URL", "http://localhost:8081")
	familyURL  = envOr("FAMILY_URL", "http://localhost:8082")
	financeURL = envOr("FINANCE_URL", "http://localhost:8083")
	bots       = strings.Split(envOr("E2E_BOTS", "finance,recipes,notes,family"), ",")
)

func translations(keys ...i18n.Key) []string {
	var out []string
	for _, loc := range i18n.Supported {
		for _, k := range keys {
			out = append(out, i18n.T(loc, k))
		}
	}
	return out
}

func plain(m *message) string {
	if m == nil {
		return ""
	}
	return html.UnescapeString(m.Text)
}

func mentions(m *message, texts []string) (string, bool) {
	p := plain(m)
	for _, t := range texts {
		if strings.Contains(p, t) {
			return t, true
		}
	}
	return "", false
}

func envOr(k, v string) string {
	if s := os.Getenv(k); s != "" {
		return s
	}
	return v
}

type message struct {
	MessageID   int64  `json:"message_id"`
	Text        string `json:"text"`
	ReplyMarkup *struct {
		InlineKeyboard [][]struct {
			Text         string `json:"text"`
			CallbackData string `json:"callback_data"`
		} `json:"inline_keyboard"`
	} `json:"reply_markup"`
}

type event struct {
	Seq             int64    `json:"seq"`
	Kind            string   `json:"kind"`
	Message         *message `json:"message"`
	CallbackQueryID string   `json:"callback_query_id"`
	Text            string   `json:"text"`
}

type call struct {
	Method string `json:"method"`
	Status int    `json:"status"`
	Error  string `json:"error"`
}

type client struct {
	t    *testing.T
	http *http.Client
}

func (c *client) do(method, u string, body, out any, header map[string]string) int {
	c.t.Helper()
	var r io.Reader
	if body != nil {
		raw, _ := json.Marshal(body)
		r = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, u, r)
	if err != nil {
		c.t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range header {
		req.Header.Set(k, v)
	}
	res, err := c.http.Do(req)
	if err != nil {
		c.t.Fatalf("%s %s: %v", method, u, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if out != nil && len(raw) > 0 {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: decode %q: %v", method, u, raw, err)
		}
	}
	if res.StatusCode >= 400 && out == nil {
		c.t.Logf("%s %s -> %d %s", method, u, res.StatusCode, raw)
	}
	return res.StatusCode
}

func (c *client) rpc(service, method string, body, out any, token string) {
	c.t.Helper()
	c.rpcAt(authURL, service, method, body, out, token)
}

func (c *client) rpcAt(base, service, method string, body, out any, token string) {
	c.t.Helper()
	h := map[string]string{}
	if token != "" {
		h["Authorization"] = "Bearer " + token
	}
	var raw json.RawMessage
	code := c.do("POST", fmt.Sprintf("%s/%s/%s", base, service, method), body, &raw, h)
	if code != 200 {
		c.t.Fatalf("%s/%s: %d %s", service, method, code, raw)
	}
	if err := json.Unmarshal(raw, out); err != nil {
		c.t.Fatal(err)
	}
}

type tgUser struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Lang      string `json:"language_code"`
}

type session struct {
	*client
	bot  string
	user tgUser
	seq  int64
}

func (s *session) path(p string) string {
	return emuURL + "/_emu/bots/" + url.PathEscape(s.bot) + p
}

func (s *session) send(text string) {
	s.t.Helper()
	var out struct {
		Seq int64 `json:"seq"`
	}
	if code := s.do("POST", s.path("/messages"), map[string]any{"from": s.user, "text": text}, &out, nil); code != 200 {
		s.t.Fatalf("send %q: %d", text, code)
	}
	s.seq = out.Seq
}

func (s *session) press(messageID int64, data string) (string, bool) {
	s.t.Helper()
	var out struct {
		Seq int64  `json:"seq"`
		ID  string `json:"callback_query_id"`
	}
	code := s.do("POST", s.path("/callbacks"),
		map[string]any{"from": s.user, "message_id": messageID, "data": data}, &out, nil)
	if code != 200 {
		return "", false
	}
	s.seq = out.Seq
	return out.ID, true
}

func (s *session) await(kinds string, quiet time.Duration) []event {
	s.t.Helper()
	var all []event
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		wait := quiet
		if len(all) == 0 {
			wait = time.Until(deadline)
		}
		var evs []event
		q := fmt.Sprintf("/events?after=%d&chat=%d&kind=%s&timeout=%s", s.seq, s.user.ID, kinds, wait.Round(time.Millisecond))
		s.do("GET", s.path(q), nil, &evs, nil)
		if len(evs) == 0 {
			break
		}
		all = append(all, evs...)
		s.seq = evs[len(evs)-1].Seq
	}
	return all
}

func firstButton(m *message) (string, string, bool) {
	if m == nil || m.ReplyMarkup == nil {
		return "", "", false
	}
	for _, row := range m.ReplyMarkup.InlineKeyboard {
		for _, b := range row {
			if b.CallbackData != "" {
				return b.Text, b.CallbackData, true
			}
		}
	}
	return "", "", false
}

var failures = translations(i18n.Failed, i18n.Internal, i18n.NoFamily, i18n.NotSetUp, i18n.Unknown)

func checkReplies(t *testing.T, step string, evs []event) {
	t.Helper()
	for _, e := range evs {
		if hit, bad := mentions(e.Message, failures); bad {
			t.Errorf("%s: bot replied with a failure (%q): %q", step, hit, plain(e.Message))
		}
	}
}

func hasReply(evs []event, texts []string) bool {
	for _, e := range evs {
		if _, ok := mentions(e.Message, texts); ok {
			return true
		}
	}
	return false
}

func TestBotsEndToEnd(t *testing.T) {
	c := &client{t: t, http: &http.Client{Timeout: 30 * time.Second}}
	if code := c.do("POST", emuURL+"/_emu/reset", nil, nil, nil); code != 200 {
		t.Fatalf("reset emulator at %s: %d", emuURL, code)
	}

	for _, name := range bots {
		var info struct {
			Commands []struct {
				Command string `json:"command"`
			} `json:"commands"`
		}
		deadline := time.Now().Add(30 * time.Second)
		for {
			c.do("GET", emuURL+"/_emu/bots/"+name, nil, &info, nil)
			if len(info.Commands) > 0 || time.Now().After(deadline) {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		if len(info.Commands) == 0 {
			t.Fatalf("bot %s never registered its command menu — is the telegram service pointed at the emulator?", name)
		}
	}

	tg := tgUser{ID: 700000000 + rand.Int64N(99999999), FirstName: "E2E", Lang: "en"}

	t.Run("unlinked", func(t *testing.T) {
		for _, name := range bots {
			s := &session{client: &client{t: t, http: c.http}, bot: name, user: tg}
			s.send("/start")
			evs := s.await("bot_message", 500*time.Millisecond)
			if len(evs) == 0 {
				t.Errorf("%s: no reply to /start for an unlinked user", name)
			}
			checkReplies(t, name+" /start", evs)
		}
	})

	email := fmt.Sprintf("e2e-%d@example.test", tg.ID)
	password := "e2e-password-123"
	var reg struct {
		UserID string `json:"userId"`
	}
	c.rpc("auth.v1.AuthService", "Register", map[string]any{"email": email, "password": password, "name": "E2E"}, &reg, "")
	var login struct {
		AccessToken string `json:"accessToken"`
	}
	c.rpc("auth.v1.AuthService", "Login", map[string]any{"email": email, "password": password}, &login, "")
	var link struct {
		Token string `json:"token"`
	}
	c.rpc("auth.v1.AuthService", "CreateLinkToken", map[string]any{"provider": "telegram"}, &link, login.AccessToken)

	t.Run("link-without-family", func(t *testing.T) {
		s := &session{client: &client{t: t, http: c.http}, bot: bots[0], user: tg}
		s.send("/start " + link.Token)
		evs := s.await("bot_message,bot_edit", 800*time.Millisecond)
		if !hasReply(evs, translations(i18n.LinkedTitle)) {
			t.Fatalf("link was not confirmed: %+v", evs)
		}
		if !hasReply(evs, translations(i18n.NoFamily)) {
			t.Errorf("a user without a family was not told to create one: %+v", evs)
		}
		var ids struct {
			Identities []struct {
				Provider string `json:"provider"`
			} `json:"identities"`
		}
		c.rpc("auth.v1.AuthService", "ListIdentities", map[string]any{}, &ids, login.AccessToken)
		found := false
		for _, id := range ids.Identities {
			found = found || id.Provider == "telegram"
		}
		if !found {
			t.Fatalf("auth has no telegram identity after linking: %+v", ids)
		}

		if bots[0] == "finance" {
			s.send("/balance")
			if evs := s.await("bot_message", 500*time.Millisecond); !hasReply(evs, translations(i18n.NoFamily)) {
				t.Errorf("/balance without a family did not explain the problem: %+v", evs)
			}
		}
	})

	var fam struct {
		Family struct {
			ID string `json:"id"`
		} `json:"family"`
	}
	c.rpcAt(familyURL, "family.v1.FamilyService", "CreateFamily", map[string]any{"name": "E2E family"}, &fam, login.AccessToken)
	if fam.Family.ID == "" {
		t.Fatal("CreateFamily returned no id")
	}

	if bots[0] == "finance" {
		t.Run("finance-not-set-up", func(t *testing.T) {
			s := &session{client: &client{t: t, http: c.http}, bot: "finance", user: tg}
			s.send("/balance")
			if evs := s.await("bot_message", 500*time.Millisecond); !hasReply(evs, translations(i18n.NotSetUp)) {
				t.Errorf("/balance before finance setup did not explain it: %+v", evs)
			}
		})
	}

	c.rpc("auth.v1.AuthService", "Login", map[string]any{"email": email, "password": password}, &login, "")
	var boot json.RawMessage
	c.rpcAt(financeURL, "finance.v1.FinanceService", "BootstrapHousehold", map[string]any{
		"baseCurrencyCode": "UAH", "timezone": "Europe/Kyiv", "seedDefaultTaxonomy": true,
	}, &boot, login.AccessToken)

	for _, name := range bots {
		t.Run("crawl/"+name, func(t *testing.T) {
			s := &session{client: &client{t: t, http: c.http}, bot: name, user: tg}
			var info struct {
				Commands []struct {
					Command string `json:"command"`
				} `json:"commands"`
			}
			s.do("GET", s.path(""), nil, &info, nil)
			for _, cmd := range info.Commands {
				step := name + " /" + cmd.Command
				s.send("/" + cmd.Command)
				evs := s.await("bot_message,bot_edit", 700*time.Millisecond)
				if len(evs) == 0 {
					t.Errorf("%s: no reply", step)
					continue
				}
				checkReplies(t, step, evs)

				var last *message
				for _, e := range evs {
					if e.Message != nil {
						last = e.Message
					}
				}
				label, data, ok := firstButton(last)
				if !ok {
					continue
				}
				id, ok := s.press(last.MessageID, data)
				if !ok {
					t.Errorf("%s: could not press %q", step, label)
					continue
				}
				evs = s.await("callback_answer,bot_message,bot_edit", 700*time.Millisecond)
				answered := false
				for _, e := range evs {
					answered = answered || (e.Kind == "callback_answer" && e.CallbackQueryID == id)
				}
				if !answered {
					t.Errorf("%s → [%s]: callback query never answered (client spinner hangs)", step, label)
				}
				checkReplies(t, step+" → ["+label+"]", evs)
			}
		})
	}

	t.Run("bot-api-errors", func(t *testing.T) {
		for _, name := range bots {
			var calls []call
			c.do("GET", emuURL+"/_emu/bots/"+name+"/calls", nil, &calls, nil)
			for _, cl := range calls {
				if cl.Status == 200 || strings.Contains(cl.Error, "message is not modified") {
					continue
				}
				if strings.EqualFold(cl.Method, "getUpdates") && cl.Status == http.StatusConflict {
					continue
				}
				t.Errorf("%s: %s -> %d %s", name, cl.Method, cl.Status, cl.Error)
			}
		}
	})
}
