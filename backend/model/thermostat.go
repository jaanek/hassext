package model

import "time"

type Thermostat struct {
	Id                 int       `json:"id" db:"id" gorm:"primary_key"`
	Name               string    `json:"tname" db:"tname"`
	TargetTemperature  float32   `json:"target_temperature" db:"target_temperature"`
	CurrentTemperature float32   `json:"current_temperature" db:"current_temperature"`
	LastUpdate         time.Time `json:"last_update" db:"last_update"`
}

func (c *Thermostat) TableName() string {
	return "thermostat"
}
