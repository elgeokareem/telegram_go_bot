package services

import (
	"bot/telegram/shared"
	"bot/telegram/structs"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update) error {
	chatId := update.Message.Chat.ID
	replyToMessageId := update.Message.ReplyToMessage.MessageID
	tableName := fmt.Sprintf("table_%d", chatId)

	currentMessage := update.Message
	messageToGiveKarma := update.Message.ReplyToMessage.From

	// Aqui se crea la conexion a la tabla
	fmt.Println("TABLE NAME", tableName)
	conn, err := CreateDbConnection(tableName)
	if err != nil {
		conn.Close(context.Background())
		fmt.Printf("Error connecting to the db. %s", err)
		return err
	}

	defer conn.Close(context.Background())

	// TODO: Probably put it in another place. All validations should be together.
	// Check if User can give karma given time constriction
	// CheckLastTimeUserGaveKarma(conn, currentMessage)

	sqlToAddKarma := `
		INSERT INTO users_ranking (user_id, first_name, last_name, username, karma, last_karma_given)
		VALUES ($1, $2, $3, $4, 1, $5)
		ON CONFLICT (user_id)
		DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			username = EXCLUDED.username,
			karma = users_ranking.karma + 1,
			last_karma_given = EXCLUDED.last_karma_given
	`

	currentMessageJSON, _ := json.MarshalIndent(currentMessage, "", "  ")
	fmt.Printf("CURRENT MESSAGE: %s\n", string(currentMessageJSON))

	messageToGiveKarmaJSON, _ := json.MarshalIndent(messageToGiveKarma, "", "  ")
	fmt.Printf("MESSAGE TO GIVE KARMA: %s\n", string(messageToGiveKarmaJSON))

	_, err = conn.Exec(
		context.Background(),
		sqlToAddKarma,
		messageToGiveKarma.ID,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		nil,
	)

	fmt.Println("ERROR", err)

	if err != nil {
		return fmt.Errorf("unable to upsert user ranking: %w", err)
	}

	SendMessageWithReply(chatId, replyToMessageId, "testt")

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

func ProcessTelegramMessages(telegramUrl string, token string, offset int) (int, error) {
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d", telegramUrl, token, offset)

	response, err := http.Get(completeUrl)
	if err != nil {
		fmt.Println("ERROR http.Get(completeUrl)", err)
		offset++
		return offset, err
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		fmt.Println("ERROR ReadAll(response.Body)", err)
	}

	var result structs.UpdateResponse
	errBody := json.Unmarshal(body, &result)
	if errBody != nil {
		offset++
		return offset, err
	}

	// * This is for loggin
	// jsonData, err := json.MarshalIndent(result, "", "  ")
	// fmt.Println(string(jsonData))

	for _, update := range result.Result {
		fmt.Println("UPDATE", update)
		updateID := update.UpdateID
		offset = updateID + 1

		if update.Message == nil {
			offset++
			return offset, err
		}

		// TODO: Abstract later. Make this one and the one below one
		if strings.HasPrefix(update.Message.Text, "/lovedusers@WillibertoBot") {
			chatId := update.Message.Chat.ID
			tableName := fmt.Sprintf("table_%d", chatId)
			conn, err := CreateDbConnection(tableName)
			if err != nil {
				conn.Close(context.Background())
				fmt.Printf("Error connecting to the db. %s", err)
				return offset, err
			}

			lovedUsers, err := GetMostLovedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
				return offset, err
			}
			fmt.Println(lovedUsers)

			offset++
			return offset, nil
		}

		// TODO: Abstract later. Make this one and the one above one
		if strings.HasPrefix(update.Message.Text, "/hatedusers@WillibertoBot") {
			chatId := update.Message.Chat.ID
			tableName := fmt.Sprintf("table_%d", chatId)
			conn, err := CreateDbConnection(tableName)
			if err != nil {
				conn.Close(context.Background())
				fmt.Printf("Error connecting to the db. %s", err)
				return offset, err
			}

			hatedUsers, err := GetMostHatedUsers(conn)
			if err != nil {
				fmt.Println("ERROR loved users")
				return offset, err
			}
			fmt.Println(hatedUsers)

			offset++
			return offset, nil
		}

		isPlusMinusOne := shared.ParsePlusMinusOneFromMessage(update.Message.Text)

		if !isPlusMinusOne {
			continue
		}

		// Validations for giving karma
		err := KarmaValidations(update)
		if err != nil {
			offset++
			return offset, err
		}

		errKarma := AddKarmaToUser(update)
		if err != nil {
			offset++
			return offset, errKarma
		}
	}

	return offset, nil
}
