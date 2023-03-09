package homeassistant

import (
	"fmt"
	"time"

	"github.com/jaanek/hassext/data"
)

type NordpoolPrices struct {
	EntityState
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

func (m *homeassistant) GetNordpoolPrices() NordpoolPrices {
	return m.nordpoolPrices
}

func (m *homeassistant) parseNordpoolPrices() (*NordpoolPrices, error) {
	entityId := "sensor.nordpool_mwh_ee_eur_3_10_02"
	var prices NordpoolPrices = NordpoolPrices{
		EntityState: EntityState{
			Id: entityId,
		},
	}
	var errors data.Errors

	// get prices
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", "sensor.nordpool_mwh_ee_eur_3_10_02")
	json := m.stateData.GetObject(entityPath)
	if json == nil {
		return nil, fmt.Errorf("Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.Data{Value: json}
	prices.State = entityData.GetString("$.state", &errors)
	prices.CurrentPrice = entityData.GetFloat64("$.attributes.current_price", &errors)
	prices.Average = entityData.GetFloat64("$.attributes.average", &errors)
	prices.Peak = entityData.GetFloat64("$.attributes.peak", &errors)
	prices.Min = entityData.GetFloat64("$.attributes.min", &errors)
	prices.Max = entityData.GetFloat64("$.attributes.max", &errors)
	prices.Unit = entityData.GetString("$.attributes.unit", &errors)
	prices.Currency = entityData.GetString("$.attributes.currency", &errors)
	prices.Country = entityData.GetString("$.attributes.country", &errors)
	if errors.HasAny() {
		return nil, errors
	}
	todayData := entityData.GetArray("$.attributes.today.*")
	for _, t := range todayData {
		v, ok := t.(float64)
		if ok {
			prices.Today = append(prices.Today, v)
		}
	}
	tomorrowData := entityData.GetArray("$.attributes.tomorrow.*")
	for _, t := range tomorrowData {
		v, ok := t.(float64)
		if ok {
			prices.Tomorrow = append(prices.Tomorrow, v)
		}
	}
	return &prices, nil
}
