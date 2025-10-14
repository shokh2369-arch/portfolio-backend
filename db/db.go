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
		created_at TEXT DEFAULT (datetime('now')),
		featured TEXT
	);

	CREATE VIRTUAL TABLE IF NOT EXISTS blog_search USING fts5(
		title,
		body,
		content='blog_data',
		content_rowid='id',
		tokenize='porter'
	);

	-- Trigger: when a blog is added, insert into blog_search
	CREATE TRIGGER IF NOT EXISTS blog_data_ai AFTER INSERT ON blog_data BEGIN
		INSERT INTO blog_search(rowid, title, body)
		VALUES (new.id, new.title, new.body);
	END;

	-- Trigger: when a blog is updated, update the FTS record
	CREATE TRIGGER IF NOT EXISTS blog_data_au AFTER UPDATE ON blog_data BEGIN
		UPDATE blog_search
		SET title = new.title,
			body = new.body
		WHERE rowid = new.id;
	END;

	-- Trigger: when a blog is deleted, remove it from FTS
	CREATE TRIGGER IF NOT EXISTS blog_data_ad AFTER DELETE ON blog_data BEGIN
		DELETE FROM blog_search WHERE rowid = old.id;
	END;
	`

	ctx := context.Background()
	_, err := DB.ExecContext(ctx, query)
	if err != nil {
		log.Fatalf("❌ Could not create blog tables: %v", err)
	}

	// 🧠 Ensure blog_search is synced with any existing data
	_, err = DB.ExecContext(ctx, `
	INSERT INTO blog_search(rowid, title, body)
	SELECT id, title, body
	FROM blog_data
	WHERE id NOT IN (SELECT rowid FROM blog_search);
	`)
	if err != nil {
		log.Printf("⚠️ Could not sync blog_search with blog_data: %v", err)
	}

	fmt.Println("✅ Content tables and FTS index initialized successfully.")
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
