package bot

import (
	"html"
	"strings"

	"github.com/nnc/family-manager/services/telegram/internal/telegram"
)

type Button struct {
	Text string
	Data string
	URL  string
}

type Keyboard [][]Button

func Row(buttons ...Button) []Button { return buttons }

func Data(text, data string) Button { return Button{Text: text, Data: data} }

func Link(text, url string) Button { return Button{Text: text, URL: url} }

func (k Keyboard) markup() *telegram.InlineKeyboardMarkup {
	if len(k) == 0 {
		return nil
	}
	rows := make([][]telegram.InlineKeyboardButton, 0, len(k))
	for _, row := range k {
		if len(row) == 0 {
			continue
		}
		out := make([]telegram.InlineKeyboardButton, 0, len(row))
		for _, b := range row {
			out = append(out, telegram.InlineKeyboardButton{
				Text: b.Text, CallbackData: b.Data, URL: b.URL,
			})
		}
		rows = append(rows, out)
	}
	if len(rows) == 0 {
		return nil
	}
	return &telegram.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func (k Keyboard) Grid(buttons []Button, perRow int) Keyboard {
	out := k
	for i := 0; i < len(buttons); i += perRow {
		end := min(i+perRow, len(buttons))
		out = append(out, buttons[i:end])
	}
	return out
}

func Esc(s string) string { return html.EscapeString(s) }

func Bold(s string) string { return "<b>" + Esc(s) + "</b>" }

func Italic(s string) string { return "<i>" + Esc(s) + "</i>" }

func Code(s string) string { return "<code>" + Esc(s) + "</code>" }

func Lines(lines ...string) string { return strings.Join(lines, "\n") }
