package services

import (
	"context"

	"github.com/jackc/pgx/v5"
)

type ErrorRecordInput struct {
	GroupID *int64
	UserID  *int64
	Error   string
}

func CreateErrorRecord(conn *pgx.Conn, input ErrorRecordInput) error {
	sqlInsertErrorRecord := `
		INSERT INTO bot_errors (user_id, group_id, error)
		VALUES ($1, $2, $3)
	`

	_, err := conn.Exec(context.Background(), sqlInsertErrorRecord)
	return err
}
