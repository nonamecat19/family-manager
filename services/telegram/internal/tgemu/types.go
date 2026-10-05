package tgemu

type User struct {
	ID           int64  `json:"id"`
	IsBot        bool   `json:"is_bot"`
	FirstName    string `json:"first_name"`
	LastName     string `json:"last_name,omitempty"`
	Username     string `json:"username,omitempty"`
	LanguageCode string `json:"language_code,omitempty"`
}

type Chat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title,omitempty"`
	Username  string `json:"username,omitempty"`
	FirstName string `json:"first_name,omitempty"`
}

type Entity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type Button struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type Markup struct {
	InlineKeyboard [][]Button `json:"inline_keyboard"`
}

type Message struct {
	MessageID      int64    `json:"message_id"`
	From           *User    `json:"from,omitempty"`
	Chat           Chat     `json:"chat"`
	Date           int64    `json:"date"`
	EditDate       int64    `json:"edit_date,omitempty"`
	Text           string   `json:"text,omitempty"`
	Entities       []Entity `json:"entities,omitempty"`
	ReplyMarkup    *Markup  `json:"reply_markup,omitempty"`
	ReplyToMessage *Message `json:"reply_to_message,omitempty"`
}

type CallbackQuery struct {
	ID           string   `json:"id"`
	From         User     `json:"from"`
	Message      *Message `json:"message,omitempty"`
	ChatInstance string   `json:"chat_instance"`
	Data         string   `json:"data,omitempty"`
}

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

func (u Update) kind() string {
	switch {
	case u.Message != nil:
		return "message"
	case u.EditedMessage != nil:
		return "edited_message"
	case u.CallbackQuery != nil:
		return "callback_query"
	}
	return ""
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type Record struct {
	Message   Message  `json:"message"`
	ParseMode string   `json:"parse_mode,omitempty"`
	Outgoing  bool     `json:"outgoing"`
	Deleted   bool     `json:"deleted,omitempty"`
	History   []string `json:"history,omitempty"`
}

type Event struct {
	Seq             int64    `json:"seq"`
	Kind            string   `json:"kind"`
	ChatID          int64    `json:"chat_id,omitempty"`
	Message         *Message `json:"message,omitempty"`
	ParseMode       string   `json:"parse_mode,omitempty"`
	CallbackQueryID string   `json:"callback_query_id,omitempty"`
	Data            string   `json:"data,omitempty"`
	Text            string   `json:"text,omitempty"`
	ShowAlert       bool     `json:"show_alert,omitempty"`
}

type Answer struct {
	Answered  bool   `json:"answered"`
	Text      string `json:"text,omitempty"`
	ShowAlert bool   `json:"show_alert,omitempty"`
	URL       string `json:"url,omitempty"`
}

type Call struct {
	At     int64          `json:"at"`
	Method string         `json:"method"`
	Params map[string]any `json:"params,omitempty"`
	Status int            `json:"status"`
	Error  string         `json:"error,omitempty"`
}

type Fault struct {
	Bot         string `json:"bot"`
	Method      string `json:"method"`
	ErrorCode   int    `json:"error_code"`
	Description string `json:"description"`
	RetryAfter  int    `json:"retry_after,omitempty"`
	Count       int    `json:"count"`
}
