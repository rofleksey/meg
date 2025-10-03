package meg

import "math"

func FixPrice(x float64) float64 {
	return math.Round(x*100) / 100
}
