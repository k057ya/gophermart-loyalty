package model

import "time"

type User struct {
	ID       int64
	Login    string
	Password string `json:"-"`
	Date     time.Time
}
