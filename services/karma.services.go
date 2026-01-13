package services

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bot/telegram/shared"
	"bot/telegram/structs"

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

	if *karmaValue > 0 {
		senderKarmaGivenIncrement = 1
	} else if *karmaValue < 0 {
		senderKarmaTakenIncrement = 1
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
		CreateErrorRecord(conn, ErrorRecordInput{
			GroupID:    senderGroupID,
			SenderID:   senderID,
			ReceiverID: messageToGiveKarma.ID,
			Error:      fmt.Sprintf("error updating karma_given/taken for sender: %v", err),
		})
		return fmt.Errorf("error updating karma_given/taken for sender: %w", err)
	}

	successMessage := fmt.Sprintf("Karma added to %s. Total karma: %d", messageToGiveKarma.FirstName, totalKarma)
	if err := SendMessageWithReply(chatId, replyToMessageId, successMessage); err != nil {
		CreateErrorRecord(conn, ErrorRecordInput{
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

		chatID := update.Message.Chat.ID
		text := update.Message.Text
		chatType := update.Message.Chat.Type

		// Handle commands that don't require a reply message
		if update.Message.From != nil {
			userID := update.Message.From.ID

			// Handle /start command (for deep links from birthday button)
			if strings.HasPrefix(text, "/start") && IsPrivateChat(chatType) {
				handled, err := HandleStartCommand(conn, chatID, update.Message.From, text)
				if err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				}
				if handled {
					continue
				}
			}

			// Handle private messages for conversational flow (birthday registration)
			if IsPrivateChat(chatType) && !strings.HasPrefix(text, "/") {
				handled, err := HandlePrivateMessage(conn, chatID, update.Message.From, text)
				if err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				}
				if handled {
					continue
				}
			}

			// Handle /add command in group chat (shows birthday button)
			if (strings.HasPrefix(text, "/add") || strings.HasPrefix(text, "/add@")) && IsGroupChat(chatType) {
				if err := HandleAddCommand(chatID); err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{GroupID: chatID, Error: err.Error()})
				}
				continue
			}

			// Handle /birthday command in group chat (alias for /add)
			if (strings.HasPrefix(text, "/birthday") || strings.HasPrefix(text, "/birthday@")) && IsGroupChat(chatType) {
				if err := HandleBirthdayCommand(chatID); err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{GroupID: chatID, Error: err.Error()})
				}
				continue
			}

			// Handle /createevent command in group chat
			if strings.HasPrefix(text, "/createevent") && IsGroupChat(chatType) {
				if err := HandleCreateEventCommand(conn, chatID, userID, text); err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{GroupID: chatID, SenderID: userID, Error: err.Error()})
				}
				continue
			}

			// Handle /events command in group chat
			if (strings.HasPrefix(text, "/events") || strings.HasPrefix(text, "/events@")) && IsGroupChat(chatType) {
				if err := HandleEventsCommand(conn, chatID); err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{GroupID: chatID, Error: err.Error()})
				}
				continue
			}

			// Handle /deleteevent command in group chat
			if strings.HasPrefix(text, "/deleteevent") && IsGroupChat(chatType) {
				if err := HandleDeleteEventCommand(conn, chatID, text); err != nil {
					CreateErrorRecord(conn, ErrorRecordInput{GroupID: chatID, SenderID: userID, Error: err.Error()})
				}
				continue
			}
		}

		// From here, we need a reply message for karma functionality
		if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
			continue
		}

		senderMessageId := update.Message.MessageID

		// Handle /lovedusers command
		if strings.HasPrefix(text, "/lovedusers@WillibertoBot") {
			lovedUsers, err := GetMostLovedUsers(conn)
			if err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				continue
			}
			fmt.Println(lovedUsers)
			continue
		}

		// Handle /hatedusers command
		if strings.HasPrefix(text, "/hatedusers@WillibertoBot") {
			hatedUsers, err := GetMostHatedUsers(conn)
			if err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{Error: err.Error()})
				continue
			}
			fmt.Println(hatedUsers)
			continue
		}

		isPlusMinusOne, karmaValue := shared.ParsePlusMinusOneFromMessage(text)
		if !isPlusMinusOne {
			continue
		}

		// Validations for giving karma
		if err := KarmaValidations(update, conn); err != nil {
			CreateErrorRecord(conn, ErrorRecordInput{
				GroupID:    chatID,
				SenderID:   update.Message.From.ID,
				ReceiverID: update.Message.ReplyToMessage.From.ID,
				Error:      err.Error(),
			})
			continue
		}

		if err := AddKarmaToUser(update, karmaValue, conn); err != nil {
			errorInput := ErrorRecordInput{
				SenderID:   update.Message.ReplyToMessage.From.ID,
				ReceiverID: update.Message.From.ID,
				GroupID:    chatID,
				Error:      err.Error(),
			}
			if err := SendMessageWithReply(chatID, senderMessageId, "Error adding karma"); err != nil {
				CreateErrorRecord(conn, ErrorRecordInput{
					GroupID:    chatID,
					SenderID:   update.Message.From.ID,
					ReceiverID: update.Message.ReplyToMessage.From.ID,
					Error:      err.Error(),
				})
				continue
			}
			CreateErrorRecord(conn, errorInput)
			continue
		}
	}

	return newOffset, nil
}
