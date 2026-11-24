package model

import "time"

type User struct {
	Id       int64
	Login    string
	Password string `json:"-"`
	Date     time.Time
}
