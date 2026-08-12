package game

import "math/rand/v2"

func RollDice(count int) []int {
	var result []int

	for i := 0; i < count; i++ {
		n := rand.IntN(6) + 1
		result = append(result, n)
	}

	return result
}

func CountDice(dice []int) map[int]int {
	counts := make(map[int]int)

	for _, value := range dice {
		counts[value]++
	}

	return counts
}

func CheckPriceDice(dices map[int]int) bool {
	hasPriceDice := false
	for num, count := range dices {
		if num == 1 || num == 5 {
			hasPriceDice = true
		} else if count >= 3 {
			hasPriceDice = true
		}
	}
	return hasPriceDice
}
