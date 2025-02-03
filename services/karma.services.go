package services

import (
	"bot/telegram/shared"
	"bot/telegram/structs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update, conn *pgx.Conn) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID

	// currentMessage := update.Message
	messageToGiveKarma := update.Message.ReplyToMessage.From

	// TODO: Probably put it in another place. All validations should be together.
	// Check if User can give karma given time constriction
	// CheckLastTimeUserGaveKarma(conn, currentMessage)

	sqlToAddKarma := `
		INSERT INTO users_ranking (user_id, group_id, first_name, last_name, username, karma, last_karma_given)
		VALUES ($1, $2, $3, $4, $5, 1, $6)
		ON CONFLICT (user_id, group_id)
		DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			username = EXCLUDED.username,
			karma = users_ranking.karma + 1,
			last_karma_given = EXCLUDED.last_karma_given
    RETURNING karma
	`

	// currentMessageJSON, _ := json.MarshalIndent(currentMessage, "", "  ")
	// fmt.Printf("CURRENT MESSAGE: %s\n", string(currentMessageJSON))
	//
	// messageToGiveKarmaJSON, _ := json.MarshalIndent(messageToGiveKarma, "", "  ")
	// fmt.Printf("MESSAGE TO GIVE KARMA: %s\n", string(messageToGiveKarmaJSON))

	var totalKarma int
	err := conn.QueryRow(
		context.Background(),
		sqlToAddKarma,
		messageToGiveKarma.ID,
		chatId,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		time.Now(),
	).Scan(&totalKarma)
	if err != nil {
		SendMessageWithReply(chatId, replyToMessageId, "Error adding karma")
		return fmt.Errorf("unable to upsert user ranking: %w", err)
	}

	successMessage := fmt.Sprintf("Karma added to %s. Total karma: %d", messageToGiveKarma.FirstName, totalKarma)
	SendMessageWithReply(chatId, replyToMessageId, successMessage)

	return nil
}

// Check when was the last time the user gave karma.
func CheckLastTimeUserGaveKarma(conn *pgx.Conn, currentMessage *structs.Message) error {
	// TODO: For the future add a flag in the table with true or false something like.
	sqlToAddKarmaGiver := `
		SELECT last_karma_given FROM users_ranking ur WHERE ur.user_id = $1
	`

	_, err := conn.Exec(
		context.Background(),
		sqlToAddKarmaGiver,
		currentMessage.From.ID,
	)
	if err != nil {
		return fmt.Errorf("error => CheckIfUserCanGiveKarma:  %w", err)
	}

	return nil
}

func UpdateKarmaGivenTimeOfUser(conn *pgx.Conn, currentMessage *structs.Message) {
	// TODO: this lol
}

func KarmaValidations(update structs.Update) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID

	if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
		return errors.New("no reply or sender")
	}

	// If user try to give karma to itself
	if update.Message.ReplyToMessage.From.ID == update.Message.From.ID {
		SendMessageWithReply(chatId, replyToMessageId, "You can't give karma to yourself dummy")
		return errors.New("can't give karma to yourself")
	}

	// TODO: Add time validation to add karma
	// UpdateKarmaGivenTimeOfUser()

	return nil
}

func ProcessTelegramMessages(telegramUrl string, token string, offset int, conn *pgx.Conn) (int, error) {
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d", telegramUrl, token, offset)

	response, err := shared.CustomClient.Get(completeUrl)
	if err != nil {
		fmt.Println("ERROR http.Get(completeUrl)", err)
		return offset + 1, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("ERROR ReadAll(response.Body)", err)
		return offset + 1, err
	}

	var result structs.UpdateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return offset + 1, err
	}

	newOffset := offset
	for _, update := range result.Result {
		newOffset = update.UpdateID + 1

		if update.Message == nil {
			continue // Skip this update instead of returning
		}

		// Handle /lovedusers command
		if strings.HasPrefix(update.Message.Text, "/lovedusers@WillibertoBot") {
			chatId := update.Message.Chat.ID
			tableName := fmt.Sprintf("table_%d", chatId)
			conn, err := CreateDbConnection(tableName)
			if err != nil {
				conn.Close(context.Background())
				fmt.Printf("Error connecting to the db. %s", err)
				continue // Skip this update instead of returning
			}

			lovedUsers, err := GetMostLovedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
				continue // Skip this update instead of returning
			}
			fmt.Println(lovedUsers)
			continue // Move to next update
		}

		// Handle /hatedusers command
		if strings.HasPrefix(update.Message.Text, "/hatedusers@WillibertoBot") {
			chatId := update.Message.Chat.ID
			tableName := fmt.Sprintf("table_%d", chatId)
			conn, err := CreateDbConnection(tableName)
			if err != nil {
				conn.Close(context.Background())
				fmt.Printf("Error connecting to the db. %s", err)
				continue // Skip this update instead of returning
			}

			hatedUsers, err := GetMostHatedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
				continue // Skip this update instead of returning
			}
			fmt.Println(hatedUsers)
			continue // Move to next update
		}

		isPlusMinusOne := shared.ParsePlusMinusOneFromMessage(update.Message.Text)
		if !isPlusMinusOne {
			continue
		}

		// Validations for giving karma
		if err := KarmaValidations(update); err != nil {
			continue // Skip this update instead of returning
		}

		if err := AddKarmaToUser(update, conn); err != nil {
			continue // Skip this update instead of returning
		}
	}

	return newOffset, nil
}
