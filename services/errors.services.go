package services

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type ErrorRecordInput struct {
	GroupID  int64
	SenderID int64
	Error    string
}

func CreateErrorRecord(conn *pgx.Conn, input ErrorRecordInput) error {
	sqlInsertErrorRecord := `
		INSERT INTO bot_errors (sender_id, group_id, error)
		VALUES ($1, $2, $3)
	`

	_, err := conn.Exec(context.Background(), sqlInsertErrorRecord, input.SenderID, input.GroupID, input.Error)
	return err
}
