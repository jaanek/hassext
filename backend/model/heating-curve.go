package model

type HeatingCurveItem struct {
	Id                  int `json:"id" db:"id" gorm:"primary_key"`
	ExternalTemperature int `json:"external_temperature" db:"external_temperature"`
	TargetTemperature   int `json:"target_temperature" db:"target_temperature"`
}

func (c *HeatingCurveItem) TableName() string {
	return "heating_curve"
}
