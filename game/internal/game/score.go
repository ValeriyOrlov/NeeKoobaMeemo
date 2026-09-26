package game

func CalculateScore(selectedDice []int) (int, []int) {
	counts := make(map[int]int)
	for _, val := range selectedDice {
		counts[val]++
	}

	points := 0

	// Большой стрит (1-2-3-4-5-6)
	if counts[1] >= 1 && counts[2] >= 1 && counts[3] >= 1 && counts[4] >= 1 && counts[5] >= 1 && counts[6] >= 1 {
		points += 1500
		for i := 1; i <= 6; i++ {
			counts[i]--
		}
	} else {
		// 2. Малый стрит (1-2-3-4-5)
		if counts[1] >= 1 && counts[2] >= 1 && counts[3] >= 1 && counts[4] >= 1 && counts[5] >= 1 {
			points += 500
			for i := 1; i <= 5; i++ {
				counts[i]--
			}
		} else if counts[2] >= 1 && counts[3] >= 1 && counts[4] >= 1 && counts[5] >= 1 && counts[6] >= 1 {
			// 3. Малый стрит (2-3-4-5-6)
			points += 500
			for i := 2; i <= 6; i++ {
				counts[i]--
			}
		}
	}

	// 4. Тройки и шестерки одинаковых кубиков
	for num, count := range counts {
		triples := count / 3 // Вычисляем количество троек (если кубиков 6, triples будет 2)
		if triples > 0 {
			if num == 1 {
				points += triples * 1000
			} else {
				points += triples * num * 100
			}
			counts[num] -= triples * 3
		}
	}

	// 5. Подсчет оставшихся единиц и пятерок
	if counts[1] > 0 {
		points += counts[1] * 100
		counts[1] = 0
	}
	if counts[5] > 0 {
		points += counts[5] * 50
		counts[5] = 0
	}

	// 6. Сбор всех оставшихся (непризовых) кубиков
	var invalidDice []int
	for num, remaining := range counts {
		for i := 0; i < remaining; i++ {
			invalidDice = append(invalidDice, num)
		}
	}

	return points, invalidDice
}

func IsZonk(score int) bool {
	return score == 0
}
