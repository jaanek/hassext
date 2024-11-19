package brain

import (
	"fmt"
	"time"

	"github.com/jaanek/hassext/dishwasher"
	"github.com/jaanek/hassext/homeassistant"
	"github.com/jaanek/hassext/nordpool"
)

// set start time for diswasher for tomorrow night
func (b *brain) SetDishwasherStartTime(tomorrowPrices []float64) error {
	// validate
	if len(tomorrowPrices) <= 0 {
		return nil
	}
	if len(tomorrowPrices) < 12 {
		return fmt.Errorf("Tomorrow's electricity prices cannot be less than 12 hours. Len of hours: %v", len(tomorrowPrices))
	}
	n := time.Now()
	// t1 := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, t.Nanosecond(), t.Location())

	// we concat today late night + tomorrow night till morning hours to get cheapest from those
	// hours := append([]float64(nil), todayPrices[23:]...) // from 23:00 -> 23:00
	// hours := append([]float64(nil), tomorrowPrices[0:7]...) // from 00:00 -> 06:00

	// get the cheapestAll 4 sequential hours from provided list. 4 hours is enough for dishwasher to finish in eco mode
	cheapestAll := nordpool.FindCheapestElectricityHours(tomorrowPrices, 4, 0, 7)
	if len(cheapestAll) > 0 {
		cheapest := cheapestAll[0]
		start := time.Date(n.Year(), n.Month(), n.Day(), cheapest.StartIndex, 0, 0, 0, n.Location())
		if b.dishwasherStart != start {
			err := b.ha.SetInputDateTime(string(dishwasher.ENTITY_TIME_DISHWASHER_START), start, homeassistant.INPUT_TIME)
			if err != nil {
				return err
			}
			b.dishwasherStart = start
		}
	}
	return nil
}
