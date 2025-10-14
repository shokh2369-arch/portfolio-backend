package db

import (
	"context"
	"database/sql"
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

	-- Trigger for INSERT
	CREATE TRIGGER IF NOT EXISTS blog_data_ai AFTER INSERT ON blog_data BEGIN
		INSERT INTO blog_search(rowid, title, body)
		VALUES (new.id, new.title, new.body);
	END;

	-- Trigger for DELETE
	CREATE TRIGGER IF NOT EXISTS blog_data_ad AFTER DELETE ON blog_data BEGIN
		INSERT INTO blog_search(blog_search, rowid, title, body)
		VALUES('delete', old.id, old.title, old.body);
	END;

	-- Trigger for UPDATE
	CREATE TRIGGER IF NOT EXISTS blog_data_au AFTER UPDATE ON blog_data BEGIN
		INSERT INTO blog_search(blog_search, rowid, title, body)
		VALUES('delete', old.id, old.title, old.body);
		INSERT INTO blog_search(rowid, title, body)
		VALUES (new.id, new.title, new.body);
	END;
	`

	_, err := DB.Exec(query)
	if err != nil {
		log.Fatalf("Error creating tables or triggers: %v", err)
	}

	// ✅ Backfill blog_search if it's empty
	var count int
	err = DB.QueryRow(`SELECT count(*) FROM blog_search;`).Scan(&count)
	if err != nil {
		log.Printf("Error checking blog_search count: %v", err)
		return
	}

	if count == 0 {
		log.Println("blog_search is empty — backfilling data...")
		_, err = DB.Exec(`INSERT INTO blog_search(rowid, title, body)
			SELECT id, title, body FROM blog_data;`)
		if err != nil {
			log.Printf("Error backfilling blog_search: %v", err)
		} else {
			log.Println("✅ blog_search successfully backfilled.")
		}
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
