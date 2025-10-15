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

// ✅ Add new blog content
func (c *Content) Add() error {
	if c.Featured == "" {
		c.Featured = "false"
	}

	// Upload image to Cloudinary
	imageURL, err := utils.UploadImage(c.Image)
	if err != nil {
		return fmt.Errorf("failed to upload image: %w", err)
	}

	query := `
	INSERT INTO blog_data (language, type, image, title, body, meta_tag, created_at, featured)
	VALUES (?, ?, ?, ?, ?, ?, datetime('now'), ?);
	`

	res, err := db.DB.ExecContext(context.Background(), query,
		c.Language,
		c.Type,
		imageURL, // Save full Cloudinary URL
		c.Title,
		c.Body,
		c.Tag,
		c.Featured,
	)
	if err != nil {
		return fmt.Errorf("failed to insert content: %w", err)
	}

	id, _ := res.LastInsertId()
	c.ID = id

	row := db.DB.QueryRowContext(context.Background(), "SELECT created_at FROM blog_data WHERE id = ?", id)
	if err := row.Scan(&c.CreatedAt); err != nil {
		return fmt.Errorf("could not get created_at: %w", err)
	}

	return nil
}

// ✅ Update blog content
func (c *Content) Update() error {
	imagePath := c.Image
	if !strings.HasPrefix(imagePath, "http") {
		var err error
		imagePath, err = utils.UploadImage(imagePath)
		if err != nil {
			return fmt.Errorf("failed to upload image: %w", err)
		}
	}

	query := `
	UPDATE blog_data
	SET image = ?, title = ?, body = ?, meta_tag = ?, featured = ?
	WHERE id = ?;
	`
	_, err := db.DB.ExecContext(context.Background(), query,
		imagePath, c.Title, c.Body, c.Tag, c.Featured, c.ID)
	if err != nil {
		return fmt.Errorf("failed to update blog_data: %w", err)
	}

	// Refresh FTS manually (triggers usually handle this)
	_, _ = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(blog_search, rowid, title, body)
		 VALUES('delete', ?, '', '');`, c.ID)
	_, _ = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(rowid, title, body) VALUES (?, ?, ?)`,

		c.ID, c.Title, c.Body)

	return nil
}

// ✅ Delete from both tables
func (c *Content) Delete() error {
	_, err := db.DB.ExecContext(context.Background(),
		"DELETE FROM blog_data WHERE id = ?", c.ID)
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}
	_, _ = db.DB.ExecContext(context.Background(),
		`INSERT INTO blog_search(blog_search, rowid, title, body)
		 VALUES('delete', ?, '', '');`, c.ID)
	return nil
}

// ✅ Get single blog
func GetById(id int64) (Content, error) {
	query := `
	SELECT id, language, type, image, title, body, meta_tag, created_at, featured
	FROM blog_data WHERE id = ?;
	`
	row := db.DB.QueryRowContext(context.Background(), query, id)

	var c Content
	if err := row.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title, &c.Body, &c.Tag, &c.CreatedAt, &c.Featured); err != nil {
		if err == sql.ErrNoRows {
			return Content{}, fmt.Errorf("blog not found")
		}
		return Content{}, fmt.Errorf("failed to get content: %w", err)
	}

	// ⚠️ No need to rebuild Cloudinary URL — it's already a full URL
	return c, nil
}

// ✅ Get all contents (with optional filters & search)
func GetContents(title string, page int, language string, category string, featured string) ([]Content, error) {
	const limit = 10
	offset := (page - 1) * limit

	var (
		rows *sql.Rows
		err  error
	)

	// 🔍 If search keyword is given
	if title != "" {
		match := title + "*"

		query := `
			SELECT d.id, d.language, d.type, d.image, d.title, d.body,
				   d.created_at, d.featured, bm25(blog_search) AS score
			FROM blog_search
			JOIN blog_data d ON d.id = blog_search.rowid
			WHERE blog_search MATCH ?
			  AND (? = '' OR d.language = ?)
			  AND (? = '' OR d.type = ?)
			  AND (? = '' OR d.featured = ?)
			ORDER BY score ASC
			LIMIT ? OFFSET ?;
		`

		rows, err = db.DB.QueryContext(context.Background(),
			query,
			match,
			language, language,
			category, category,
			featured, featured,
			limit, offset)
	} else {
		// 🧾 Normal list (no search)
		query := `
			SELECT id, language, type, image, title, body, created_at, featured
			FROM blog_data
			WHERE (? = '' OR language = ?)
			  AND (? = '' OR type = ?)
			  AND (? = '' OR featured = ?)
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?;
		`

		rows, err = db.DB.QueryContext(context.Background(),
			query,
			language, language,
			category, category,
			featured, featured,
			limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %w", err)
	}
	defer rows.Close()

	var contents []Content
	for rows.Next() {
		var c Content
		if title != "" {
			err = rows.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title,
				&c.Body, &c.CreatedAt, &c.Featured, &c.Score)
		} else {
			err = rows.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title,
				&c.Body, &c.CreatedAt, &c.Featured)
			c.Score = nil
		}
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// ⚠️ No BuildURL needed — use stored full Cloudinary link
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
