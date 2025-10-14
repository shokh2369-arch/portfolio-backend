package content

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"example.com/portfolio/db"
	"example.com/portfolio/utils"
)

type Content struct {
	ID        int64    `json:"id"`
	Language  string   `json:"language"`
	Type      string   `json:"type"`
	Image     string   `json:"image"`
	Title     string   `json:"title"`
	Body      string   `json:"body"`
	Tag       string   `json:"meta_tag,omitempty"`
	CreatedAt string   `json:"created_at"`
	Featured  string   `json:"featured,omitempty"`
	Score     *float64 `json:"score,omitempty"`
}

// Add new content and sync with FTS table
func (c *Content) Add() error {
	if c.Featured == "" {
		c.Featured = "false"
	}

	query := `
	INSERT INTO blog_data (language, type, image, title, body, meta_tag, created_at, featured)
	VALUES (?, ?, ?, ?, ?, ?, datetime('now'), ?);
	`

	res, err := db.DB.ExecContext(context.Background(), query,
		c.Language,
		c.Type,
		utils.ImageUploadPath(c.Image),
		c.Title,
		c.Body,
		c.Tag,
		c.Featured,
	)
	if err != nil {
		return fmt.Errorf("failed to insert content: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}
	c.ID = id

	// Get created_at
	row := db.DB.QueryRowContext(context.Background(), "SELECT created_at FROM blog_data WHERE id = ?", id)
	if err := row.Scan(&c.CreatedAt); err != nil {
		return fmt.Errorf("could not find blog_data with this ID: %w", err)
	}

	// Sync new record into FTS
	_, err = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(rowid, title, body) VALUES (?, ?, ?)`,
		c.ID, c.Title, c.Body,
	)
	if err != nil {
		return fmt.Errorf("failed to update blog_search index: %w", err)
	}

	return nil
}

// Update existing content and refresh FTS index
func (c *Content) Update() error {
	imagePath := c.Image
	if !strings.HasPrefix(imagePath, "http") {
		imagePath = utils.ImageUploadPath(imagePath)
	}

	query := `
	UPDATE blog_data
	SET image = ?, title = ?, body = ?, meta_tag = ?, featured = ?
	WHERE id = ?;
	`
	_, err := db.DB.ExecContext(context.Background(), query,
		imagePath,
		c.Title,
		c.Body,
		c.Tag,
		c.Featured,
		c.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update blog_data: %w", err)
	}

	// Delete old FTS record
	_, _ = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(blog_search, rowid, title, body)
		 VALUES('delete', ?, '', '');`, c.ID)

	// Reinsert updated FTS record
	_, err = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(rowid, title, body) VALUES (?, ?, ?)`,
		c.ID, c.Title, c.Body,
	)
	if err != nil {
		return fmt.Errorf("failed to reinsert FTS record: %w", err)
	}

	return nil
}

// Delete content and remove from FTS index
func (c *Content) Delete() error {
	_, err := db.DB.ExecContext(context.Background(), "DELETE FROM blog_data WHERE id = ?", c.ID)
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}

	// Delete from FTS index
	_, _ = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(blog_search, rowid, title, body)
		 VALUES('delete', ?, '', '');`, c.ID)

	return nil
}

// Get single blog post by ID
func GetById(id int64) (Content, error) {
	query := `
	SELECT id, language, type, image, title, body, meta_tag, created_at, featured
	FROM blog_data
	WHERE id = ?;
	`
	row := db.DB.QueryRowContext(context.Background(), query, id)

	var c Content
	err := row.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title, &c.Body, &c.Tag, &c.CreatedAt, &c.Featured)
	if err != nil {
		if err == sql.ErrNoRows {
			return Content{}, fmt.Errorf("blog not found")
		}
		return Content{}, fmt.Errorf("failed to get content: %w", err)
	}

	c.Image = utils.Url(c.Image)
	return c, nil
}

// Get list of blogs (with optional search)
func GetContents(title string, page int, language string, category string, featured string) ([]Content, error) {
	const limit = 10
	offset := (page - 1) * limit

	var (
		rows *sql.Rows
		err  error
	)

	if title != "" {
		// 🔍 Full-text search mode
		query := `
			SELECT blog_data.id, blog_data.language, blog_data.type, blog_data.image, blog_data.title,
				   blog_data.body, blog_data.created_at, blog_data.featured,
				   bm25(blog_search) AS score
			FROM blog_search
			JOIN blog_data ON blog_data.id = blog_search.rowid
			WHERE blog_search MATCH ?
			  AND blog_data.language = ?
			  AND blog_data.type = ?
			  AND blog_data.featured = ?
			ORDER BY score ASC
			LIMIT ? OFFSET ?;
		`
		match := fmt.Sprintf("title:%s*", title)
		rows, err = db.DB.QueryContext(context.Background(), query, match, language, category, featured, limit, offset)
	} else {
		// 🧾 Normal (non-search) mode
		query := `
			SELECT id, language, type, image, title, body, created_at, featured
			FROM blog_data
			WHERE language = ?
			  AND type = ?
			  AND featured = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?;
		`
		rows, err = db.DB.QueryContext(context.Background(), query, language, category, featured, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %w", err)
	}
	defer rows.Close()

	var contents []Content
	for rows.Next() {
		var c Content
		if title != "" {
			if err := rows.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title, &c.Body, &c.CreatedAt, &c.Featured, &c.Score); err != nil {
				return nil, err
			}
		} else {
			if err := rows.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title, &c.Body, &c.CreatedAt, &c.Featured); err != nil {
				return nil, err
			}
		}
		c.Image = utils.Url(c.Image)
		contents = append(contents, c)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if len(contents) == 0 {
		return nil, fmt.Errorf("no contents found")
	}

	return contents, nil
}
