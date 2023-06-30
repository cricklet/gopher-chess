package accuracy

import "math"

func WinPercentage(centipawns int) float64 {
	return 50.0 + 50.0*(2.0/(1.0+math.Exp(-0.00368208*float64(centipawns)))-1.0)
}

func AccuracyForScores(a int, b int) float64 {
	winA := WinPercentage(a)
	winB := WinPercentage(b)
	return 103.1668*math.Exp(-0.04354*(winB-winA)) - 3.1669
}
