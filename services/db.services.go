package services

import (
	"bot/telegram/utils"
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5"
)

/**
que quiero hacer?
Basicamente quiero crear una funcion que se conecte a postgres
Para eso inicio a una conexion por defecto que tenga postgres.

Despues de que tenga la conexion establecida, quiero crear una base de datos general
para el bot de telegram.

La funcion debe devolver de forma generica una conexion porque en una se usa para crear
la base de datos general y para otra la usare para crear los tenants.
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

	return newConn, err
}

func databaseExists(conn *pgx.Conn, dbName string) (bool, error) {
	var exists bool
	err := conn.QueryRow(context.Background(), "SELECT EXISTS (SELECT datname FROM pg_catalog.pg_database WHERE datname = $1)", dbName).Scan(&exists)
	return exists, err
}

func createDatabase(conn *pgx.Conn, dbName string) error {
	_, err := conn.Exec(context.Background(), fmt.Sprintf("CREATE DATABASE %s", dbName))
	return err
}
