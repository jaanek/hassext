package nordpool

import "sort"

type SequentialHours struct {
	StartIndex int
	Limit      float64
}
type SequentialHoursAvg struct {
	StartIndex   int
	AveragePrice float64
	TotalPrice   float64
}

type SequentialHoursList []SequentialHours

func FindCheapestElectricityHours(hours []float64, nrOfSeqHours int) []SequentialHoursAvg {
	var weights = make([]SequentialHoursAvg, 0)
	for i := 0; i < len(hours); i++ {
		seqHours := []float64{}
		for j := i; j < len(hours); j++ {
			h := hours[j]
			seqHours = append(seqHours, h)

			// calculate average for the # of hours and save it
			if len(seqHours) == nrOfSeqHours {
				weights = append(weights, SequentialHoursAvg{
					StartIndex:   i,
					AveragePrice: calcAvg(seqHours),
					TotalPrice:   calcTotal(seqHours),
				})
				break // calculate next set of seq hours starting from next i
			}
		}
		// we cannot calculate any more averages
		if len(seqHours) < nrOfSeqHours {
			break
		}
	}

	// find the lowes weight
	sort.Slice(weights, func(i, j int) bool {
		return weights[i].TotalPrice < weights[j].TotalPrice
	})
	return weights
}

func FindCheapestElectricityHoursByIncLevel(hours []float64, nrOfSeqHours int) SequentialHoursList {
	all := []SequentialHours{}
	ascending := OrderHoursAscending(hours)
	for _, limit := range ascending {
		found := SequentialHours{}
		seqHours := []float64{}
		for j := 0; j < len(hours); j++ {
			h := hours[j]

			// skip the ones not below limit/threshold
			if h > limit {
				// reset the found
				found = SequentialHours{}
				seqHours = []float64{}
				continue
			}

			// add the hour below limit/threshold to the list
			seqHours = append(seqHours, h)
			if len(seqHours) == 1 {
				found.StartIndex = j
				found.Limit = limit
			} else if len(seqHours) == nrOfSeqHours {
				// found.EndIndex = j
				all = append(all, found)
				// reset the found
				found = SequentialHours{}
				seqHours = []float64{}
			}
		}
		// exit when we have found the cheapest
		if len(all) > 0 {
			break
		}
	}
	return all
}

func calcTotal(hours []float64) float64 {
	var total float64
	for _, h := range hours {
		total += h
	}
	return total
}

func calcAvg(hours []float64) float64 {
	var total float64
	for _, h := range hours {
		total += h
	}
	return total / float64(len(hours))
}

func FindMinMax(hours []float64) (min float64, max float64) {
	for _, h := range hours {
		if h < min {
			min = h
		}
		if h > max {
			max = h
		}
	}
	return
}

func OrderHoursAscending(hours []float64) []float64 {
	ordered := make([]float64, len(hours))
	copy(ordered, hours)
	sort.Float64s(ordered)
	return ordered
}
