package services

import (
	"bot/telegram/structs"
	"bot/telegram/utils"
	"context"
	"fmt"
)

func AddKarmaToUser(update structs.Update) error {
	chatId := utils.Abs(update.Message.Chat.ID)
	tableName := fmt.Sprintf("table_%d", chatId)
	message := update.Message.ReplyToMessage.From

	fmt.Println(tableName)
	conn, err := CreateDbConnection(tableName)
	if err != nil {
		conn.Close(context.Background())
		fmt.Printf("Error connecting to the db. %s", err)
		return err
	}

	defer conn.Close(context.Background())

	sql := `
		INSERT INTO users_ranking (user_id, first_name, last_name, username, karma)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (user_id)
		DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			username = EXCLUDED.username,
			karma = users_ranking.karma + 1
		RETURNING id
	`

	err = conn.QueryRow(
		context.Background(),
		sql,
		message.ID,
		message.FirstName,
		message.LastName,
		message.Username,
	).Scan()

	if err != nil {
		return fmt.Errorf("unable to upsert user ranking: %w", err)
	}

	return nil
}
