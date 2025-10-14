package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/tursodatabase/libsql-client-go/libsql"
)

var DB *sql.DB

func Initdb() {
	dbURL := os.Getenv("dbURL")
	if dbURL == "" {
		log.Fatal("❌ dbURL environment variable is not set")
	}
	authToken := os.Getenv("authToken")
	if authToken == "" {
		log.Fatal("❌ authToken environment variable is not set")
	}

	// Create a connector using URL and auth token
	connector, err := libsql.NewConnector(dbURL, libsql.WithAuthToken(authToken))
	if err != nil {
		log.Fatalf("❌ Could not create Turso connector: %v", err)
	}

	// Open a *sql.DB instance using the connector
	DB = sql.OpenDB(connector)

	log.Println("✅ Connected to Turso successfully!")

	// Create tables
	infotable()
	content()
	signUp()
}

func infotable() {
	query := `
	CREATE TABLE IF NOT EXISTS info(
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		lastname TEXT NOT NULL,
		phone TEXT NOT NULL,
		description TEXT,
		telegram TEXT NOT NULL,
		ip TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := DB.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatalf("❌ Could not create info table: %v", err)
	}
}

func content() {
	query := `
	CREATE TABLE IF NOT EXISTS blog_data (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		language TEXT,
		type TEXT,
		image TEXT,
		title TEXT,
		body TEXT,
		meta_tag TEXT,
		created_at TEXT,
		featured TEXT
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS blog_search USING fts5(
		title,
		body,
		content='blog_data',
		content_rowid='id'
	);
	`

	_, err := DB.Exec(query)
	if err != nil {
		fmt.Println("Error creating content tables:", err)
	} else {
		fmt.Println("✅ Content tables initialized successfully.")
	}

	_, err = DB.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatalf("❌ Could not create blog table: %v", err)
	}
}

func signUp() {
	query := `
	CREATE TABLE IF NOT EXISTS signUp(
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password TEXT NOT NULL
	);
	`

	_, err := DB.ExecContext(context.Background(), query)
	if err != nil {
		log.Fatalf("❌ Could not create signUp table: %v", err)
	}
}
