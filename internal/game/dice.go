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

func CheckPriceDice(counts map[int]int) bool {
	for num, count := range counts {
		if num == 1 || num == 5 {
			return true
		} else if count >= 3 {
			return true
		}
	}
	return false
}

// containsAllDice проверяет, что все выбранные кубики есть на столе
func containsAllDice(tableDice []int, selectedDice []int) bool {
	tableCopy := make([]int, len(tableDice))
	copy(tableCopy, tableDice)

	for _, die := range selectedDice {
		found := false
		for i, tableDie := range tableCopy {
			if tableDie == die {
				// Убираем найденный кубик из копии стола
				tableCopy = append(tableCopy[:i], tableCopy[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

func removeSelectedDice(tableDice []int, selectedDice []int) ([]int, bool) {
	newTable := make([]int, len(tableDice))
	copy(newTable, tableDice)

	for _, d := range selectedDice {
		found := false
		for i, td := range newTable {
			if td == d {
				// Удаляем найденный кубик из оставшихся на столе
				newTable = append(newTable[:i], newTable[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			// Если мы не нашли кубик, значит клиент прислал неверные данные
			return tableDice, false
		}
	}
	return newTable, true
}
