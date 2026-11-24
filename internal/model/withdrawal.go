package model

import "time"

type Withdrawal struct {
	Id      int64
	OrderId int64
	Sum     float64
	Date    time.Time
}
