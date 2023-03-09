package homeassistant

import "sort"

type SequentialHours struct {
	StartIndex int
	EndIndex   int
}

func FindCheapestElectricityHours(hours []float64, nrOfSeqHours int) []SequentialHours {
	all := []SequentialHours{}
	ascending := OrderHoursAscending(hours)
	// min, max := ascending[0], ascending[len(ascending)-1]
	for _, limit := range ascending {
		found := SequentialHours{}
		seqHours := []float64{}
		for j := 0; j < len(hours); j++ {
			h := hours[j]

			// skip the ones not below limit/threshold
			if h > limit {
				// if we had previous hours below limit/threshold and we are not at the end
				// if len(seqHours) > 0 && (j+1) < len(hours) {
				// 	// start from next
				// 	j = found.StartIndex
				// }

				// reset the found
				found = SequentialHours{}
				seqHours = []float64{}
				continue
			}

			// add the hour below limit/threshold to the list
			seqHours = append(seqHours, h)
			if len(seqHours) == 1 {
				found.StartIndex = j
			} else if len(seqHours) == nrOfSeqHours {
				found.EndIndex = j
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
