package model

import "time"

type Withdrawal struct {
	ID      int64
	OrderID int64
	Sum     float64
	Date    time.Time
}
