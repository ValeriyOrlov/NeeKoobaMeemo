package game

func CalculateScore(counts map[int]int) int {
	// Создаем копию карты, чтобы не мутировать исходные данные
	c := make(map[int]int)
	for k, v := range counts {
		c[k] = v
	}

	points := 0

	// Проверка на большой стрит (1-2-3-4-5-6)
	if c[1] >= 1 && c[2] >= 1 && c[3] >= 1 && c[4] >= 1 && c[5] >= 1 && c[6] >= 1 {
		points += 1500
		for i := 1; i <= 6; i++ {
			c[i]--
		}
	} else {
		// Проверка на малый стрит (1-2-3-4-5)
		if c[1] >= 1 && c[2] >= 1 && c[3] >= 1 && c[4] >= 1 && c[5] >= 1 {
			points += 500
			for i := 1; i <= 5; i++ {
				c[i]--
			}
			// Проверка на малый стрит (2-3-4-5-6)
		} else if c[2] >= 1 && c[3] >= 1 && c[4] >= 1 && c[5] >= 1 && c[6] >= 1 {
			points += 500
			for i := 2; i <= 6; i++ {
				c[i]--
			}
		}
	}

	// Подсчет оставшихся троек и одиночных кубиков
	for num, count := range c {
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
