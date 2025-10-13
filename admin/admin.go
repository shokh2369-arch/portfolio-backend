package admin

import (
	"errors"
	"net/mail"
	"strings"

	"example.com/portfolio/db"
	"example.com/portfolio/utils"
)

type Login struct {
	Login    string `json:"login" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type Address struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (a *Address) SignUp() error {
	_, err := mail.ParseAddress(a.Email)
	query := "INSERT INTO signUp (username, email, password) VALUES (?, ?, ?) "
	stmt, err := db.DB.Prepare(query)

	if err != nil {
		return err
	}
	defer stmt.Close()

	hashedpass, err := utils.HashPassword(a.Password)

	_, err = stmt.Exec(a.Username, a.Email, hashedpass)
	if err != nil {
		return err
	}
	return nil

}

func (s *Address) Login() error {
	query := "SELECT username, email, password FROM signUp"
	var args = []interface{}{}

	var isEmail bool = strings.Contains(s.Email, "@")
	if isEmail {
		query += " WHERE email = ?"
		args = append(args, s.Email)

	}
	if !isEmail {
		query += " WHERE username = ?"
		args = append(args, s.Username)
	}

	row := db.DB.QueryRow(query, args...)

	var retrievedpass string

	err := row.Scan(&s.Username, &s.Email, &retrievedpass)
	if err != nil {
		return err
	}
	valid := utils.CheckPassword(retrievedpass, s.Password)
	if !valid {
		return errors.New("Invalid credential")
	}
	return nil
}
