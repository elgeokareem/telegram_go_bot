package services

import (
	"bot/telegram/config"
	"bot/telegram/errors"
	"bot/telegram/shared"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

type chatMember struct {
	User   chatMemberUser `json:"user"`
	Status string         `json:"status"`
}

type chatMemberUser struct {
	ID int64 `json:"id"`
}

type getChatAdministratorsResponse struct {
	OK     bool         `json:"ok"`
	Result []chatMember `json:"result"`
}

type adminCache struct {
	timestamp time.Time
	admins    map[int64]bool
}

var (
	adminCacheMu sync.RWMutex
	adminCaches  = make(map[int64]*adminCache)
)

const adminCacheTTL = 5 * time.Minute

func getChatAdministrators(chatId int64) (map[int64]bool, error) {
	adminCacheMu.RLock()
	cached, found := adminCaches[chatId]
	adminCacheMu.RUnlock()

	if found && time.Since(cached.timestamp) < adminCacheTTL {
		return cached.admins, nil
	}

	env := config.Current
	url := fmt.Sprintf("%s%s/getChatAdministrators?chat_id=%d", env.TelegramBaseURL, env.Token, chatId)

	resp, err := shared.CustomClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("getChatAdministrators request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read getChatAdministrators response: %w", err)
	}

	var result getChatAdministratorsResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse getChatAdministrators response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("getChatAdministrators API returned not OK")
	}

	admins := make(map[int64]bool, len(result.Result))
	for _, member := range result.Result {
		if member.Status == "creator" || member.Status == "administrator" {
			admins[member.User.ID] = true
		}
	}

	adminCacheMu.Lock()
	adminCaches[chatId] = &adminCache{timestamp: time.Now(), admins: admins}
	adminCacheMu.Unlock()

	return admins, nil
}

func isUserAdmin(chatId int64, userId int64) (bool, error) {
	admins, err := getChatAdministrators(chatId)
	if err != nil {
		return false, err
	}
	return admins[userId], nil
}

type eventRow struct {
	ID      int64
	Title   string
	Type    string
	EventAt *string
}

func ShowEvents(conn *pgx.Conn, chatId int64) {
	ctx := context.Background()
	rows, err := conn.Query(ctx, `
		SELECT id, title, type,
			CASE
				WHEN is_all_day THEN to_char(event_date, 'YYYY-MM-DD')
				ELSE to_char(event_at, 'YYYY-MM-DD HH24:MI')
			END AS event_at
		FROM events
		WHERE chat_id = $1 AND is_active = TRUE
		ORDER BY created_at DESC
	`, chatId)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID: chatId,
			Error:   fmt.Sprintf("query events: %v", err),
		})
		_ = SendMessage(chatId, "Failed to retrieve events.")
		return
	}
	defer rows.Close()

	var events []eventRow
	for rows.Next() {
		var e eventRow
		if err := rows.Scan(&e.ID, &e.Title, &e.Type, &e.EventAt); err != nil {
			_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
				GroupID: chatId,
				Error:   fmt.Sprintf("scan event row: %v", err),
			})
			_ = SendMessage(chatId, "Failed to read events.")
			return
		}
		events = append(events, e)
	}

	if len(events) == 0 {
		_ = SendMessage(chatId, "No active events in this group.")
		return
	}

	var b strings.Builder
	b.WriteString("Active events:\n\n")
	for _, e := range events {
		eventAt := ""
		if e.EventAt != nil {
			eventAt = *e.EventAt
		}
		b.WriteString(fmt.Sprintf("#%d | %s | %s", e.ID, e.Type, e.Title))
		if eventAt != "" {
			b.WriteString(fmt.Sprintf(" | %s", eventAt))
		}
		b.WriteString("\n")
	}

	_ = SendMessage(chatId, b.String())
}

func DeleteEvent(conn *pgx.Conn, chatId int64, userId int64, eventIdStr string) {
	isAdmin, err := isUserAdmin(chatId, userId)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:  chatId,
			SenderID: userId,
			Error:    fmt.Sprintf("check admin: %v", err),
		})
		_ = SendMessage(chatId, "Failed to verify admin permissions.")
		return
	}

	if !isAdmin {
		_ = SendMessage(chatId, "Only group admins can delete events.")
		return
	}

	eventId, err := strconv.ParseInt(strings.TrimSpace(eventIdStr), 10, 64)
	if err != nil {
		_ = SendMessage(chatId, "Invalid event ID. Usage: /delete_event <id>")
		return
	}

	ctx := context.Background()
	tag, err := conn.Exec(ctx, `
		DELETE FROM events
		WHERE id = $1 AND chat_id = $2
	`, eventId, chatId)
	if err != nil {
		_ = errors.CreateErrorRecord(conn, errors.ErrorRecordInput{
			GroupID:  chatId,
			SenderID: userId,
			Error:    fmt.Sprintf("delete event %d: %v", eventId, err),
		})
		_ = SendMessage(chatId, "Failed to delete event.")
		return
	}

	if tag.RowsAffected() == 0 {
		_ = SendMessage(chatId, fmt.Sprintf("Event #%d not found in this group.", eventId))
		return
	}

	_ = SendMessage(chatId, fmt.Sprintf("Event #%d deleted.", eventId))
}
