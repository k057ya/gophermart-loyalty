package model

import "time"

type Order struct {
	ID      int `json:"number"`
	UserID  int
	Status  OrderStatus // status
	Accrual int         // hide when empty
	Date    time.Time   // uploaded_at
}
