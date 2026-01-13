package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"bot/telegram/config"
	"bot/telegram/shared"
	"bot/telegram/structs"
)

// SendMessageWithInlineKeyboard sends a message with inline keyboard buttons
func SendMessageWithInlineKeyboard(chatID int64, text string, keyboard *structs.InlineKeyboardMarkup) error {
	baseURL := config.Env.TelegramBaseURL + config.Env.Token + "/sendMessage"

	payload := structs.SendMessageRequest{
		ChatID:      chatID,
		Text:        text,
		ParseMode:   "Markdown",
		ReplyMarkup: keyboard,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := shared.CustomClient.Post(baseURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status %d", resp.StatusCode)
	}

	return nil
}

// GenerateBirthdayDeepLink creates a deep link for birthday registration
// The user will be redirected to bot's private chat with start parameter containing groupID
func GenerateBirthdayDeepLink(botUsername string, groupID int64) string {
	return fmt.Sprintf("https://t.me/%s?start=birthday_%d", botUsername, groupID)
}

// SendBirthdayButton sends a message with the birthday registration button
func SendBirthdayButton(chatID int64, botUsername string) error {
	deepLink := GenerateBirthdayDeepLink(botUsername, chatID)

	keyboard := &structs.InlineKeyboardMarkup{
		InlineKeyboard: [][]structs.InlineKeyboardButton{
			{
				{
					Text: "🎂 Registrar mi cumpleaños",
					URL:  deepLink,
				},
			},
		},
	}

	message := "🎉 *¡Hola!* Presiona el botón para registrar tu cumpleaños y recibir una felicitación especial cuando llegue tu día."

	return SendMessageWithInlineKeyboard(chatID, message, keyboard)
}

// ParseStartParameter parses the start parameter from /start command
// Returns the type and value (e.g., "birthday", "123456789")
func ParseStartParameter(param string) (string, int64, error) {
	if len(param) < 10 || param[:9] != "birthday_" {
		return "", 0, fmt.Errorf("invalid start parameter")
	}

	groupIDStr := param[9:]
	groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
	if err != nil {
		return "", 0, fmt.Errorf("invalid group ID: %w", err)
	}

	return "birthday", groupID, nil
}
