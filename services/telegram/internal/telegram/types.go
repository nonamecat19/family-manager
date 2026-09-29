package telegram

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	EditedMessage *Message       `json:"edited_message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID int64           `json:"message_id"`
	From      *User           `json:"from,omitempty"`
	Chat      Chat            `json:"chat"`
	Date      int64           `json:"date"`
	Text      string          `json:"text,omitempty"`
	Entities  []MessageEntity `json:"entities,omitempty"`
}

type MessageEntity struct {
	Type   string `json:"type"`
	Offset int    `json:"offset"`
	Length int    `json:"length"`
}

type User struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID       int64  `json:"id"`
	Type     string `json:"type"`
	Title    string `json:"title,omitempty"`
	Username string `json:"username,omitempty"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type SendMessageParams struct {
	ChatID             int64                 `json:"chat_id"`
	Text               string                `json:"text"`
	ParseMode          string                `json:"parse_mode,omitempty"`
	DisableLinkPreview bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyToMessageID   int64                 `json:"reply_to_message_id,omitempty"`
	ReplyMarkup        *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type EditMessageTextParams struct {
	ChatID             int64                 `json:"chat_id"`
	MessageID          int64                 `json:"message_id"`
	Text               string                `json:"text"`
	ParseMode          string                `json:"parse_mode,omitempty"`
	DisableLinkPreview bool                  `json:"disable_web_page_preview,omitempty"`
	ReplyMarkup        *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}

type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

type BotCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type setMyCommandsParams struct {
	Commands []BotCommand `json:"commands"`
}

type getUpdatesParams struct {
	Offset         int64    `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}

type setWebhookParams struct {
	URL            string   `json:"url"`
	SecretToken    string   `json:"secret_token,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
	MaxConnections int      `json:"max_connections,omitempty"`
}

type deleteWebhookParams struct {
	DropPendingUpdates bool `json:"drop_pending_updates,omitempty"`
}

type answerCallbackParams struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}
