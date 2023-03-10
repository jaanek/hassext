package nordpool

import (
	"time"
)

type NordpoolPrices struct {
	Id           string `json:"entity_id"`
	State        string `json:"state"`
	Updated      time.Time
	CurrentPrice float64   `json:"currentPrice"`
	Average      float64   `json:"average"`
	Peak         float64   `json:"peak"`
	Min          float64   `json:"min"`
	Max          float64   `json:"max"`
	Unit         string    `json:"unit"`
	Currency     string    `json:"currency"`
	Country      string    `json:"country"`
	Today        []float64 `json:"today"`
	Tomorrow     []float64 `json:"tomorrow"`
}
