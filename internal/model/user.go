package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID          int64 `json:"id" db:"id"`
	Credentials
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

type Credentials struct {
	Login 	 string `json:"login" db:"login"`
	Password string `json:"-" db:"password"`
}

type Token struct {
	UserID int64 `json:"user_id" db:"user_id"`
}

func (c *Credentials) HashPassword(p string) (string, error) {
	var hash []byte
	hash, err := bcrypt.GenerateFromPassword([]byte(p), 10)
	return string(hash), err
}

func (c *Credentials) Check(p string) error {
	return bcrypt.CompareHashAndPassword([]byte(c.Password), []byte(p))
}
