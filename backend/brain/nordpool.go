package brain

import (
	"time"

	"github.com/jaanek/hassext/data"
	"github.com/jaanek/hassext/nordpool"
)

func (b *brain) Nordpool(state data.DataValue) {
	// get nordpool prices
	prices, err := nordpool.ParseNordpoolPrices(state)
	if err != nil {
		b.errors <- err
	} else {
		b.nordpoolPrices = *prices
		b.nordpoolPrices.Updated = time.Now()
		b.lo.Info("Nordpool", "prices", b.nordpoolPrices)
	}
}
