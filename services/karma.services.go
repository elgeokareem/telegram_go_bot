package services

import (
	"bot/telegram/structs"
	"bot/telegram/utils"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

func AddKarmaToUser(update structs.Update) error {
	// TODO: change this logic to outside of this function
	if update.Message.ReplyToMessage == nil || update.Message.ReplyToMessage.From == nil {
		return errors.New("no reply or sender")
	}

	chatId := update.Message.Chat.ID
	tableName := fmt.Sprintf("table_%d", chatId)

	currentMessage := update.Message
	messageToGiveKarma := update.Message.ReplyToMessage.From

	fmt.Println(tableName)
	conn, err := CreateDbConnection(tableName)
	if err != nil {
		conn.Close(context.Background())
		fmt.Printf("Error connecting to the db. %s", err)
		return err
	}

	defer conn.Close(context.Background())

	// Check if User can give karma
	CheckIfUserCanGiveKarma(conn, currentMessage)

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

	_, err = conn.Exec(
		context.Background(),
		sqlToAddKarma,
		messageToGiveKarma.ID,
		messageToGiveKarma.FirstName,
		messageToGiveKarma.LastName,
		messageToGiveKarma.Username,
		nil,
	)

	if err != nil {
		return fmt.Errorf("unable to upsert user ranking: %w", err)
	}

	SendMessage(chatId, "Ayy lmaoshin")

	return nil
}

// Check when was the last time the user gave karma.
func CheckIfUserCanGiveKarma(conn *pgx.Conn, currentMessage *structs.Message) error {
	// TODO: For the future add a flag in the table with true or false something like.
	sqlToAddKarmaGiver := `
		SELECT last_karma_given FROM users_ranking ur WHERE ur.user_id = $1
	`

	test, err := conn.Exec(
		context.Background(),
		sqlToAddKarmaGiver,
		currentMessage.From.ID,
	)
	if err != nil {
		return fmt.Errorf("tremendo lio %w", err)
	}

	fmt.Println("Test", test)

	return nil
}

func UpdateKarmaGivenTimeOfUser(conn *pgx.Conn, currentMessage *structs.Message) {
}

func GetKarmaUpdates(telegramUrl string, token string, offset int) (int, error) {
	completeUrl := fmt.Sprintf("%s%s/getUpdates?offset=%d", telegramUrl, token, offset)

	response, err := http.Get(completeUrl)
	if err != nil {
		log.Fatal(err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result structs.UpdateResponse
	errBody := json.Unmarshal(body, &result)
	if errBody != nil {
		return offset, err
	}

	for _, update := range result.Result {
		updateID := update.UpdateID
		offset = updateID + 1
		if updateID >= offset {
		}

		isPlusOne := utils.ParsePlusOneFromMessage(update.Message.Text)

		if !isPlusOne {
			continue
		}

		AddKarmaToUser(update)
	}

	return offset, nil
}
