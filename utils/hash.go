package utils

import (
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	byte, err := bcrypt.GenerateFromPassword([]byte(password), 14)
	return string(byte), err
}

func CheckPassword(hashedpass, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedpass), []byte(password))
	return err == nil
}
