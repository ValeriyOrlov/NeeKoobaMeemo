package game

func CalculateScore(counts map[int]int) int {
	score := 0

	for num, count := range counts {
		if count >= 3 {
			if num == 1 {
				score += 1000
			} else {
				score += num * 100
			}
			count -= 3
		}
		if num == 1 {
			score += count * 100
		} else if num == 5 {
			score += count * 50
		}
	}
	return score
}

func IsZonk(score int) bool {
	return score == 0
}
