package info

import (
	"errors"
	"strings"
	"time"

	"example.com/portfolio/db"
)

type About struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name" binding:"required"`
	Lastname    string    `json:"lastname" binding:"required"`
	Phone       string    `json:"phone" binding:"required"`
	Description string    `json:"description"`
	Telegram    string    `json:"telegram" binding:"required"`
	IP          string    `json:"ip"`
	CreatedAt   time.Time `json:"createdAt"`
}

func (a *About) Save(ip string) error {
	if !strings.HasPrefix(a.Telegram, "@") {
		return errors.New("the telegram username should start with '@'")
	}

	query := `
		INSERT INTO info (name, lastname, phone, description, telegram, ip) 
		VALUES (?, ?, ?, ?, ?, ?)
	`
	stmt, err := db.DB.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	res, err := stmt.Exec(a.Name, a.Lastname, a.Phone, a.Description, a.Telegram, ip)
	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	a.ID = id
	a.IP = ip
	a.CreatedAt = time.Now()

	return nil
}
func CanRequest(ip string) (bool, error) {
	var count int
	query := `
        SELECT COUNT(*) 
        FROM info 
        WHERE ip = ? AND DATE(created_at) = DATE('now')
    `
	err := db.DB.QueryRow(query, ip).Scan(&count)
	if err != nil {
		return false, err
	}
	return count < 2, nil
}
