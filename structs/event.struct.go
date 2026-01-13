package structs

import "time"

// EventType constants
const (
	EventTypeEvent    = "event"
	EventTypeBirthday = "birthday"
)

// RecurrenceType constants
const (
	RecurrenceWeekly   = "weekly"
	RecurrenceMonthly  = "monthly"
	RecurrenceAnnually = "annually"
)

// GroupEvent represents an event or birthday in the database
type GroupEvent struct {
	ID             int64      `json:"id"`
	GroupID        int64      `json:"group_id"`
	UserID         int64      `json:"user_id"`
	EventType      string     `json:"event_type"`
	Title          string     `json:"title"`
	Description    string     `json:"description,omitempty"`
	EventDate      *time.Time `json:"event_date,omitempty"`
	IsRecurring    bool       `json:"is_recurring"`
	RecurrenceType string     `json:"recurrence_type,omitempty"`
	RecurrenceDay  string     `json:"recurrence_day,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// EventWithUser includes user info from users_ranking JOIN
type EventWithUser struct {
	GroupEvent
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

// InlineKeyboardButton for Telegram inline keyboards
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	URL          string `json:"url,omitempty"`
	CallbackData string `json:"callback_data,omitempty"`
}

// InlineKeyboardMarkup for Telegram inline keyboards
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// SendMessageRequest for Telegram sendMessage API with reply markup
type SendMessageRequest struct {
	ChatID      int64                 `json:"chat_id"`
	Text        string                `json:"text"`
	ParseMode   string                `json:"parse_mode,omitempty"`
	ReplyMarkup *InlineKeyboardMarkup `json:"reply_markup,omitempty"`
}
