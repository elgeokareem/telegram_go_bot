package services

import (
	"bot/telegram/shared"
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PoolManager struct {
	pools map[string]*pgxpool.Pool
}

func (pm *PoolManager) GetPool(dbName string) (*pgxpool.Pool, error) {
	if pool, exists := pm.pools[dbName]; exists {
		return pool, nil
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

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, err
	}

	pm.pools[dbName] = pool
	return pool, nil
}

func CreateDbConnection(tableName string) (*pgx.Conn, error) {
	dbSchema := os.Getenv("DB_SCHEMA")
	dbName := os.Getenv("DB_NAME")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbDefaultName := os.Getenv("DB_DEFAULT_NAME")

	dbUrl := shared.CreateDbString(dbSchema, dbUser, dbPassword, dbHost, dbPort, dbDefaultName)

	conn, err := pgx.Connect(context.Background(), dbUrl)
	if err != nil {
		return nil, fmt.Errorf("couldn't connecto to default %s DB. %w", dbSchema, err)
	}

	// if all good, check if the name of the DB exists, and if not create it.
	exists, err := databaseExists(conn, dbName)
	if err != nil {
		return nil, fmt.Errorf("couldn't check the existence of the database. %w", err)
	}

	if !exists {
		err := createDatabase(conn, dbName)
		if err != nil {
			return nil, fmt.Errorf("unable to connect to target database: %w", err)
		}
	}

	// Close the existing connection
	conn.Close(context.Background())

	// Create a new connection string with the updated database name
	newDbUrl := shared.CreateDbString(dbSchema, dbUser, dbPassword, dbHost, dbPort, dbName)

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
			user_id BIGINT NOT NULL UNIQUE,
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			username VARCHAR(255),
			karma INT,
			last_karma_given TIMESTAMP,
      allowed_to_give_karma BOOLEAN DEFAULT TRUE,
      allowed_to_receive_karma BOOLEAN DEFAULT TRUE,
      UNIQUE(user_id, group_id)
		)
	`

	_, err := conn.Exec(context.Background(), sql)
	return err
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
      error TEXT,
      created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
      updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
	`

	_, err := conn.Exec(context.Background(), sql)
	return err
}
