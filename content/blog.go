package content

import (
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

func (c *Content) Add() error {
	if c.Featured == "" {
		c.Featured = "false"
	}

	query := `
	INSERT INTO blog_data (language, type, image, title, body, meta_tag, created_at, featured)
	VALUES (?, ?, ?, ?, ?, ?, datetime('now'), ?);
	`

	res, err := db.DB.Exec(query,
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

	row := db.DB.QueryRow("SELECT created_at FROM blog_data WHERE rowid = ?", id)
	if err := row.Scan(&c.CreatedAt); err != nil {
		return fmt.Errorf("could not find blog_data with this ID: %w", err)
	}

	return nil
}

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
	_, err := db.DB.Exec(query,
		imagePath,
		c.Title,
		c.Body,
		c.Tag,
		c.Featured,
		c.ID,
	)
	return err
}

func (c *Content) Delete() error {
	query := "DELETE FROM blog_data WHERE id = ?"
	_, err := db.DB.Exec(query, c.ID)
	if err != nil {
		return fmt.Errorf("failed to delete content: %w", err)
	}
	return nil
}

func GetById(id int64) (Content, error) {
	query := "SELECT id, language, type, image, title, body, meta_tag, created_at, featured FROM blog_data WHERE id = ?"
	row := db.DB.QueryRow(query, id)

	var c Content
	err := row.Scan(&c.ID, &c.Language, &c.Type, &c.Image, &c.Title, &c.Body, &c.Tag, &c.CreatedAt, &c.Featured)
	if err != nil {
		return Content{}, err
	}
	return c, nil
}

func GetContents(title string, page int, language string, category string, featured string) ([]Content, error) {
	var contents []Content
	limit := 10
	offset := (page - 1) * limit

	var rows *sql.Rows
	var err error

	if strings.TrimSpace(title) != "" {
		// 🔍 Improved FTS5 search (matches title or body)
		query := `
			SELECT d.id, d.language, d.type, d.image, d.title, d.body, d.created_at, d.featured, bm25(s) AS score
			FROM blog_search s
			JOIN blog_data d ON d.id = s.rowid
			WHERE s MATCH ?
			  AND d.language = ?
			  AND d.type = ?
			  AND d.featured = ?
			ORDER BY score ASC
			LIMIT ? OFFSET ?;
		`

		match := fmt.Sprintf("%s*", strings.TrimSpace(title))
		rows, err = db.DB.Query(query, match, language, category, featured, limit, offset)
	} else {
		// 🧾 Normal query without search
		query := `
			SELECT id, language, type, image, title, body, created_at, featured
			FROM blog_data
			WHERE language = ?
			  AND type = ?
			  AND featured = ?
			ORDER BY created_at DESC
			LIMIT ? OFFSET ?;
		`
		rows, err = db.DB.Query(query, language, category, featured, limit, offset)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to execute SQL: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var c Content
		if strings.TrimSpace(title) != "" {
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
