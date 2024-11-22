package model

import "time"

type SMS struct {
	ID                    uint       `json:"id" db:"id"`
	UUID                  string     `json:"uuid" db:"uuid"`
	Created               time.Time  `json:"created" db:"created"`
	To                    string     `json:"cto" db:"cto"`
	Message               string     `json:"message" db:"message"`
	Finished              *time.Time `json:"finished" db:"finished"`
	Retries               *int       `json:"retries" db:"retries"`
	Result                *string    `json:"cresult" db:"cresult"`
	PriceAlertTriggeredID *uint      `json:"pa_triggered_id" db:"pa_triggered_id"`
}

func (u *SMS) TableName() string {
	return "sms"
}
