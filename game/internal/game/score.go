package game

func CalculateScore(counts map[int]int) int {
	points := 0

	for num, count := range counts {
		if count >= 3 {
			if num == 1 {
				points += 1000
			} else {
				points += num * 100
			}
			count -= 3
		}
		if num == 1 {
			points += count * 100
		} else if num == 5 {
			points += count * 50
		}
	}

	return points
}

func IsZonk(score int) bool {
	return score == 0
}
