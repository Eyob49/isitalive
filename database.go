package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/Eyob49/isitalive/internal/database"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func OpenDB() (*sql.DB, error) {
	_ = godotenv.Load()

	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		return nil, fmt.Errorf("DB_URL environment variable is not set")
	}

	db, err := sql.Open("postgres", dbUrl)
	if err != nil {
		return nil, fmt.Errorf("error opening database pool: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("error connecting to the database: %w", err)
	}

	return db, nil
}

func OpenQueries() (*database.Queries, *sql.DB, error) {
	db, err := OpenDB()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open database: %w", err)
	}

	return database.New(db), db, nil
}
