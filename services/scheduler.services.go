package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	schedulerOnce     sync.Once
	lastBirthdayCheck time.Time
)

// StartScheduler starts the background scheduler for birthday announcements
func StartScheduler(dbName string) {
	schedulerOnce.Do(func() {
		go runScheduler(dbName)
	})
}

func runScheduler(dbName string) {
	// Wait a bit before starting to let the main app initialize
	time.Sleep(10 * time.Second)

	for {
		now := time.Now()

		// Check birthdays once per day at 9:00 AM
		if now.Hour() == 9 && now.Day() != lastBirthdayCheck.Day() {
			checkAndAnnounceBirthdays(dbName)
			lastBirthdayCheck = now
		}

		// Sleep for 1 minute before next check
		time.Sleep(1 * time.Minute)
	}
}

func checkAndAnnounceBirthdays(dbName string) {
	conn, err := GlobalPoolManager.GetConnectionFromPool(dbName)
	if err != nil {
		fmt.Printf("Scheduler: Failed to get DB connection: %s\n", err)
		return
	}
	defer conn.Release()

	birthdays, err := GetTodaysBirthdays(conn.Conn())
	if err != nil {
		fmt.Printf("Scheduler: Failed to get today's birthdays: %s\n", err)
		return
	}

	for _, birthday := range birthdays {
		message := GetBirthdayAnnouncementMessage(birthday.FirstName, birthday.LastName)
		if err := SendMessage(birthday.GroupID, message); err != nil {
			fmt.Printf("Scheduler: Failed to send birthday message to group %d: %s\n", birthday.GroupID, err)
		} else {
			fmt.Printf("Scheduler: Sent birthday announcement for user %d in group %d\n", birthday.UserID, birthday.GroupID)
		}
	}
}

// CheckAndAnnounceBirthdaysNow is a manual trigger for testing
func CheckAndAnnounceBirthdaysNow(conn *pgx.Conn) error {
	birthdays, err := GetTodaysBirthdays(conn)
	if err != nil {
		return fmt.Errorf("failed to get birthdays: %w", err)
	}

	for _, birthday := range birthdays {
		message := GetBirthdayAnnouncementMessage(birthday.FirstName, birthday.LastName)
		if err := SendMessage(birthday.GroupID, message); err != nil {
			return fmt.Errorf("failed to send message: %w", err)
		}
	}

	return nil
}

// GetBotUsername returns the bot's username from config or uses default
func GetBotUsername() string {
	// For now, using hardcoded value matching the existing bot commands
	// This could be fetched via getMe API call
	return "WillibertoBot"
}

// EnsureUserInRanking ensures the user exists in users_ranking table
func EnsureUserInRanking(conn *pgx.Conn, userID, groupID int64, firstName, lastName, username string) error {
	sql := `
		INSERT INTO users_ranking (user_id, group_id, first_name, last_name, username, karma)
		VALUES ($1, $2, $3, $4, $5, 0)
		ON CONFLICT (user_id, group_id) DO UPDATE SET
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			username = EXCLUDED.username
	`

	_, err := conn.Exec(context.Background(), sql, userID, groupID, firstName, lastName, username)
	return err
}
