package nordpool

import (
	"fmt"

	"github.com/jaanek/hassext/data"
)

func ParseNordpoolPrices(state data.DataValue) (*NordpoolPrices, error) {
	entityId := "sensor.nordpool_mwh_ee_eur_3_10_02"
	var prices NordpoolPrices = NordpoolPrices{
		Id: entityId,
	}
	var errors data.Errors

	// get prices
	entityPath := fmt.Sprintf("$[?(@.entity_id == \"%v\")]", "sensor.nordpool_mwh_ee_eur_3_10_02")
	parent := state.GetObject(entityPath)
	if parent == nil {
		return nil, fmt.Errorf("Nordpool Entity state not found! entity_id: %v", entityId)
	}
	entityData := data.NewDataValue(parent)
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
		// return nil, errors
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
