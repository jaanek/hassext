package model

import "time"

type FloorHeatingController struct {
	Id         int        `json:"id" db:"id" gorm:"primary_key"`
	Name       string     `json:"cname" db:"cname"`
	LastUpdate *time.Time `json:"last_state_change" db:"last_state_change"`
}

func (c *FloorHeatingController) TableName() string {
	return "floor_heating_controller"
}
