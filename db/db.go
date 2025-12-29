package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// DB holds the database connection
var DB *sql.DB

// InitDB initializes the database connection from environment variables
func InitDB() error {
	// Get database connection string from environment
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Build connection string from individual variables
		host := os.Getenv("DB_HOST")
		port := os.Getenv("DB_PORT")
		user := os.Getenv("DB_USER")
		password := os.Getenv("DB_PASSWORD")
		dbname := os.Getenv("DB_NAME")
		sslmode := os.Getenv("DB_SSLMODE")

		if host == "" || user == "" || dbname == "" {
			return fmt.Errorf("database connection variables not set. Set DATABASE_URL or DB_HOST, DB_USER, DB_NAME")
		}

		if port == "" {
			port = "5432"
		}
		if sslmode == "" {
			sslmode = "disable"
		}

		connStr = fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
			host, port, user, password, dbname, sslmode)
	}

	var err error
	DB, err = sql.Open("pgx", connStr)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	// Test the connection
	ctx := context.Background()
	if err := DB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	log.Printf("✓ Database connection established successfully")
	return nil
}

// CloseDB closes the database connection
func CloseDB() error {
	if DB != nil {
		return DB.Close()
	}
	return nil
}
