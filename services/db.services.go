package services

import (
	"bot/telegram/utils"
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

/*
usersRankingTable
id (autoincrement)
user_id int
first_name string
last_name string
username string
karma int
*/

func CreateDbConnection(dbName string) (*pgx.Conn, error) {
	dbSchema := os.Getenv("DB_SCHEMA")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbDefaultName := os.Getenv("DB_DEFAULT_NAME")

	dbUrl := utils.CreateDbString(dbSchema, dbUser, dbPassword, dbHost, dbPort, dbDefaultName)

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
	newDbUrl := utils.CreateDbString(dbSchema, dbUser, dbPassword, dbHost, dbPort, dbName)

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

	return newConn, err
}

func databaseExists(conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(context.Background(), "SELECT EXISTS (SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", dbName).Scan(&exists)
	return exists, err
}

func createDatabase(conn *pgx.Conn, dbName string) error {
	// Safely quote the identifier
	dbName = pgx.Identifier{dbName}.Sanitize()

	_, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}

func createUsersRankingTable(conn *pgx.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS users_ranking (
			id SERIAL PRIMARY KEY,
			user_id INT NOT NULL UNIQUE,
			first_name VARCHAR(255),
			last_name VARCHAR(255),
			username VARCHAR(255),
			karma INT,
			last_karma_given TIMESTAMP
		)
	`

	_, err := conn.Exec(context.Background(), sql)
	return err
}

// TODO: implement this function but later.
func CreateGroupListTable(conn *pgx.Conn) error {
	sql := `
		CREATE TABLE IF NOT EXISTS all_groups (
		)
	`
	_, err := conn.Exec(context.Background(), sql)
	return err
}
