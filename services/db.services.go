package services

import (
	"context"
	"fmt"
	"time"

	"bot/telegram/config"
	"bot/telegram/shared"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Global pool manager instance
var GlobalPoolManager = &PoolManager{
	pools: make(map[string]*pgxpool.Pool),
}

type PoolManager struct {
	pools map[string]*pgxpool.Pool
}

func (pm *PoolManager) GetPool(dbName string) (*pgxpool.Pool, error) {
	if pool, exists := pm.pools[dbName]; exists {
		// Check if pool is still healthy
		if pool.Ping(context.Background()) == nil {
			return pool, nil
		}
		// If ping fails, close the old pool and create a new one
		pool.Close()
		delete(pm.pools, dbName)
	}

	conn, err := CreateDbConnection(dbName)
	if err != nil {
		return nil, err
	}

	// Convert the single connection to a pool
	poolConfig, err := pgxpool.ParseConfig(conn.Config().ConnString())
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = 10
	poolConfig.MinConns = 2
	poolConfig.MaxConnLifetime = time.Hour
	poolConfig.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	pm.pools[dbName] = pool
	return pool, nil
}

// GetConnectionFromPool gets a connection from the pool
func (pm *PoolManager) GetConnectionFromPool(dbName string) (*pgxpool.Conn, error) {
	pool, err := pm.GetPool(dbName)
	if err != nil {
		return nil, err
	}

	conn, err := pool.Acquire(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to acquire connection from pool: %w", err)
	}

	return conn, nil
}

func CreateDbConnection(tableName string) (*pgx.Conn, error) {
	dbUrl := shared.CreateDbString(config.Env.DBSchema, config.Env.DBUser, config.Env.DBPassword, config.Env.DBHost, config.Env.DBPort, config.Env.DBName)

	conn, err := pgx.Connect(context.Background(), dbUrl)
	if err != nil {
		return nil, fmt.Errorf("couldn't connecto to default %s DB. %w", config.Env.DBSchema, err)
	}

	// if all good, check if the name of the DB exists, and if not create it.
	exists, err := databaseExists(conn, config.Env.DBName)
	if err != nil {
		return nil, fmt.Errorf("couldn't check the existence of the database. %w", err)
	}

	if !exists {
		err := createDatabase(conn, config.Env.DBName)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to target database: %w", err)
		}
	}

	// Close the existing connection
	conn.Close(context.Background())

	// Create a new connection string with the updated database name
	newDbUrl := shared.CreateDbString(config.Env.DBSchema, config.Env.DBUser, config.Env.DBPassword, config.Env.DBHost, config.Env.DBPort, config.Env.DBName)

	// Connect to the newly created database
	newConn, err := pgx.Connect(context.Background(), newDbUrl)
	if err != nil {
		return nil, fmt.Errorf("unable to connect to target database: %w", err)
	}

	err = createUsersRankingTable(newConn)
	if err != nil {
		newConn.Close(context.Background())
		return nil, fmt.Errorf("unable to create users_ranking table: %w", err)
	}

	err = createErrorsTable(newConn)
	if err != nil {
		newConn.Close(context.Background())
		return nil, fmt.Errorf("unable to create bot_errors table: %w", err)
	}

	err = createEventsTable(newConn)
	if err != nil {
		newConn.Close(context.Background())
		return nil, fmt.Errorf("unable to create group_events table: %w", err)
	}

	return newConn, err
}

func databaseExists(conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(context.Background(), "SELECT EXISTS (SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", dbName).Scan(&exists)
	return exists, err
}

func createDatabase(conn *pgx.Conn, dbName string) error {
	// Safely quote the identifier
	// dbName = pgx.Identifier{dbName}.Sanitize()

	_, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

func createUsersRankingTable(conn *pgx.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS users_ranking (
			id SERIAL PRIMARY KEY,
      group_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			username VARCHAR(255),
			karma INT DEFAULT 0,
			last_karma_given TIMESTAMP,
      allowed_to_give_karma BOOLEAN DEFAULT TRUE,
      allowed_to_receive_karma BOOLEAN DEFAULT TRUE,
      karma_given INT DEFAULT 0,
      UNIQUE(user_id, group_id)
		)
	`

	_, err := conn.Exec(context.Background(), sql)
	return err
}

func UpsertUserKarma(conn *pgx.Conn, userID int64, groupID int64, firstName, lastName, username string, karmaValue int, karmaGivenIncrement int, karmaTakenIncrement int) (int, error) {
	sql := `
    INSERT INTO users_ranking (user_id, group_id, first_name, last_name, username, karma, last_karma_given, karma_given, karma_taken)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    ON CONFLICT (user_id, group_id)
    DO UPDATE SET
      first_name = EXCLUDED.first_name,
      last_name = EXCLUDED.last_name,
      username = EXCLUDED.username,
      karma = users_ranking.karma + $6,
      last_karma_given = EXCLUDED.last_karma_given,
      karma_given = users_ranking.karma_given + $8,
      karma_taken = users_ranking.karma_taken + $9
    RETURNING karma
  `

	var totalKarma int
	err := conn.QueryRow(
		context.Background(),
		sql,
		userID,
		groupID,
		firstName,
		lastName,
		username,
		karmaValue,
		time.Now().UTC(),
		karmaGivenIncrement,
		karmaTakenIncrement,
	).Scan(&totalKarma)
	if err != nil {
		return 0, err
	}
	return totalKarma, nil
}

type UsersLovedHatedStruct struct {
	FullName string
	Karma    int
}

func GetMostLovedUsers(conn *pgx.Conn) ([]UsersLovedHatedStruct, error) {
	sql := `
		SELECT CONCAT(first_name, ' ', last_name) as fullname, karma FROM users_ranking
		WHERE karma > 0
		ORDER BY karma DESC, fullname ASC
	`

	rows, err := conn.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UsersLovedHatedStruct
	for rows.Next() {
		var user UsersLovedHatedStruct
		if err := rows.Scan(&user.FullName, &user.Karma); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func GetMostHatedUsers(conn *pgx.Conn) ([]UsersLovedHatedStruct, error) {
	sql := `
		SELECT CONCAT(first_name, ' ', last_name) as fullname, karma FROM users_ranking
		WHERE karma < 0
		ORDER BY karma ASC, fullname ASC
	`

	rows, err := conn.Query(context.Background(), sql)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []UsersLovedHatedStruct
	for rows.Next() {
		var user UsersLovedHatedStruct
		if err := rows.Scan(&user.FullName, &user.Karma); err != nil {
			return nil, err
		}
		users = append(users, user)
	}

	// Check for any errors encountered during iteration
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func createErrorsTable(conn *pgx.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS bot_errors (
 			id SERIAL PRIMARY KEY,
       group_id BIGINT,
 			sender_id BIGINT,
       receiver_id BIGINT,
       error TEXT,
       created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
       updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
 		);
 	`

	_, err := conn.Exec(context.Background(), sql)
	return err
}

// CheckDbConnection verifies if the database connection is still alive
func CheckDbConnection(conn *pgx.Conn) error {
	if conn == nil {
		return fmt.Errorf("database connection is nil")
	}

	// Ping the database to check if it's still connected
	if err := conn.Ping(context.Background()); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	return nil
}

func createEventsTable(conn *pgx.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS group_events (
			id SERIAL PRIMARY KEY,
			group_id BIGINT NOT NULL,
			user_id BIGINT NOT NULL,
			event_type VARCHAR(20) NOT NULL DEFAULT 'event',
			title VARCHAR(255) NOT NULL,
			description TEXT,
			event_date TIMESTAMPTZ,
			is_recurring BOOLEAN DEFAULT FALSE,
			recurrence_type VARCHAR(20),
			recurrence_day VARCHAR(20),
			created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, group_id, event_type)
		);
		CREATE INDEX IF NOT EXISTS idx_events_group_id ON group_events(group_id);
		CREATE INDEX IF NOT EXISTS idx_events_type ON group_events(event_type);
	`

	_, err := conn.Exec(context.Background(), sql)
	return err
}
