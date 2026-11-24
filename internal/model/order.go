package model

import "time"

type Order struct {
	Id      int `json:"number"`
	UserId  int
	Status  OrderStatus // status
	Accrual int         // hide when empty
	Date    time.Time   // uploaded_at
}
