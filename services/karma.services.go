package services

import (
	"bot/telegram/errors"
	"bot/telegram/shared"
	"bot/telegram/structs"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update, karmaValue *int, conn *pgx.Conn) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID
	messageToGiveKarma := update.Message.ReplyToMessage.From

	totalKarma, err := UpsertUserKarma(
		conn,
		messageToGiveKarma.ID,
		chatId,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		*karmaValue, // Dereference karmaValue here
		0,           // karmaGivenIncrement for receiver
		0,           // karmaTakenIncrement for receiver
	)
	if err != nil {
		return err
	}

	// Update karma_given or karma_taken for the sender
	senderID := update.Message.From.ID
	senderGroupID := update.Message.Chat.ID
	senderFirstName := update.Message.From.FirstName
	senderLastName := update.Message.From.LastName
	senderUsername := update.Message.From.Username

	senderKarmaGivenIncrement := 0
	senderKarmaTakenIncrement := 0
	karmaMessage := ""

	if *karmaValue > 0 {
		senderKarmaGivenIncrement = 1
		karmaMessage = "given to"
	} else if *karmaValue < 0 {
		senderKarmaTakenIncrement = 1
		karmaMessage = "taken from"
	}

	_, err = UpsertUserKarma(
		conn,
		senderID,
		senderGroupID,
		senderFirstName,
		senderLastName,
		senderUsername,
		0, // karmaValue for sender (not changing sender's main karma score)
		senderKarmaGivenIncrement,
		senderKarmaTakenIncrement,
	)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    senderGroupID,
			SenderID:   senderID,
			ReceiverID: messageToGiveKarma.ID,
			Error:      fmt.Sprintf("error updating karma_given/taken for sender: %v", err),
		})
		return fmt.Errorf("error updating karma_given/taken for sender: %w", err)
	}

	successMessage := fmt.Sprintf("Karma %s %s. Total karma: %d", karmaMessage, messageToGiveKarma.FirstName, totalKarma)
	if err := SendMessageWithReply(chatId, replyToMessageId, successMessage); err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   update.Message.From.ID,
			ReceiverID: messageToGiveKarma.ID,
			Error:      err.Error(),
		})
	}

	// TODO: Add karma restrictrions per group

	return nil
}

// makeTelegramAPIRequest performs HTTP request to Telegram API with retry logic
func makeTelegramAPIRequest(url string) (*http.Response, error) {
	var lastErr error

	// Retry up to 3 times with exponential backoff
	for attempt := 0; attempt < 3; attempt++ {
		response, err := shared.CustomClient.Get(url)
		if err != nil {
			lastErr = err
			if attempt < 2 { // Don't sleep on the last attempt
				waitTime := time.Duration(attempt+1) * time.Second
				fmt.Printf("Telegram API request failed (attempt %d/3): %s. Retrying in %v...\n", attempt+1, err, waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		// Check if the response status is OK
		if response.StatusCode != http.StatusOK {
			if response.StatusCode == http.StatusConflict {
				response.Body.Close()
				return nil, fmt.Errorf("telegram getUpdates conflict (409): another bot instance is running")
			}
			response.Body.Close()
			lastErr = fmt.Errorf("telegram API returned status %d", response.StatusCode)
			if attempt < 2 {
				waitTime := time.Duration(attempt+1) * time.Second
				fmt.Printf("Telegram API returned error status %d (attempt %d/3). Retrying in %v...\n", response.StatusCode, attempt+1, waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		return response, nil
	}

	return nil, fmt.Errorf("telegram API request failed after 3 attempts: %w", lastErr)
}

func ProcessTelegramMessages(telegramUrl string, token string, offset int, conn *pgx.Conn) (int, error) {
	longPollTimeout := 25
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d&timeout=%d", telegramUrl, token, offset, longPollTimeout)

	response, err := makeTelegramAPIRequest(completeUrl)
	if err != nil {
		return offset, fmt.Errorf("failed to get updates from Telegram API: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return offset, fmt.Errorf("failed to read response body: %w", err)
	}

	var result structs.UpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return offset, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	newOffset := offset
	for _, update := range result.Result {
		newOffset = update.UpdateID + 1

		if update.Message == nil {
			continue
		}

		chatId := update.Message.Chat.ID

		// Handle /lovedusers command
		if strings.Contains(update.Message.Text, "/lovedusers") {
			MostLovedUsers(conn, chatId)
			continue
		}

		// Handle /hatedusers command
		if strings.Contains(update.Message.Text, "/hatedusers") {
			MostHatedUsers(conn, chatId)
			continue // Move to next update
		}

		// Handle +1/-1 karma updates
		isPlusMinusOne, karmaValue := shared.ParsePlusMinusOneFromMessage(update.Message.Text)
		if isPlusMinusOne {
			UpdateKarma(conn, update, karmaValue)
		}
	}

	return newOffset, nil
}

func MostLovedUsers(conn *pgx.Conn, chatId int64) {
	lovedUsers, err := GetMostLovedUsers(conn, chatId)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{Error: err.Error()})
		return
	}

	if len(lovedUsers) == 0 {
		_ = SendMessage(chatId, "No users found for this group yet.")
		return
	}

	var b strings.Builder
	b.WriteString("Most loved users (top 10):\n\n")
	for i, u := range lovedUsers {
		name := strings.TrimSpace(u.Name)
		if name == "" {
			name = "Unknown"
		}
		b.WriteString(fmt.Sprintf("%d) %s — %d\n", i+1, name, u.Karma))
	}
	_ = SendMessage(chatId, b.String())
}

func MostHatedUsers(conn *pgx.Conn, chatId int64) {
	hatedUsers, err := GetMostHatedUsers(conn, chatId)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{Error: err.Error()})
		return
	}

	if len(hatedUsers) == 0 {
		_ = SendMessage(chatId, "No users found for this group yet.")
		return
	}

	var b strings.Builder
	b.WriteString("Most hated folks here (top 10):\n\n")
	for i, u := range hatedUsers {
		name := strings.TrimSpace(u.Name)
		if name == "" {
			name = "Unknown"
		}
		b.WriteString(fmt.Sprintf("%d) %s — %d\n", i+1, name, u.Karma))
	}
	_ = SendMessage(chatId, b.String())
}

func UpdateKarma(conn *pgx.Conn, update structs.Update, karmaValue *int) {
	message := update.Message
	if update.Message == nil || message.From == nil || message.ReplyToMessage == nil || message.ReplyToMessage.From == nil {
		return
	}

	chatId := message.Chat.ID
	senderMessageId := message.MessageID

	// Handle adding/removing karma
	if err := KarmaValidations(update, conn); err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:    chatId,
			SenderID:   update.Message.From.ID,
			ReceiverID: update.Message.ReplyToMessage.From.ID,
			Error:      err.Error(),
		})
		return
	}

	if err := AddKarmaToUser(update, karmaValue, conn); err != nil {
		errorInput := errors.ErrorRecordInput{
			SenderID:   update.Message.ReplyToMessage.From.ID,
			ReceiverID: update.Message.From.ID,
			GroupID:    chatId,
			Error:      err.Error(),
		}
		if err := SendMessageWithReply(chatId, senderMessageId, "Error adding karma"); err != nil {
			_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
				GroupID:    chatId,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
			return
		}
		_ = errors.CreateErrorRecord(conn, errorInput)
	}
}
