package main

import (
	"fmt"
	"time"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/game"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
)

func botMove(bot *models.Player) {
	fmt.Printf("\n--- Ход соперника (%s) ---\n", bot.Name)
	time.Sleep(2 * time.Second)
	fmt.Printf("%s Бросает кубики...", bot.Name)
	time.Sleep(2 * time.Second)
	botDice := game.RollDice(6)
	fmt.Println("Сопернику выпало:")
	fmt.Println(botDice)
	time.Sleep(2 * time.Second)
	botCounts := game.CountDice(botDice)
	botScore := game.CalculateScore(botCounts)
	isZonk := game.IsZonk(botScore)
	if isZonk {
		fmt.Printf("Потрачено:/\n%s потерял очки за этот раунд. Текущий счёт: %d\n", bot.Name, bot.Bank)
	} else {
		bot.Bank += botScore
		fmt.Printf("%s забрал очки: %d\nНесгораемый банк: %d\n", bot.Name, botScore, bot.Bank)
	}
}

func removeFirst(slice []int, value int) []int {
	for i, v := range slice {
		if v == value {
			return append(slice[:i], slice[i+1:]...)
		}
	}
	return slice
}

func main() {
	player := models.Player{Name: "Инджрих", Bank: 0}
	bot := models.Player{Name: "Пьяница Йозеф", Bank: 0}
	roundCount := 1
	roundScore := 0
	var choice string
	isStart := true
	diceCount := 6
	for {
		//если начало игры - спрашиваем имя, ожидаем ввод, приветствуем и запрашиваем бросок кубиков
		if isStart {
			fmt.Print("Монеты жмут кошель?...\n")
			time.Sleep(1 * time.Second)
			fmt.Print("Нетерпится с ними расстаться?...\n")
			time.Sleep(1 * time.Second)
			fmt.Print("Тогда скорее присаживайся!\nКак тебя зовут? (Введи своё имя)  ")
			fmt.Scan(&player.Name)
			time.Sleep(1 * time.Second)
			fmt.Printf("Добро пожаловать в игру, %s!\n", player.Name)
			time.Sleep(1 * time.Second)
			fmt.Println("Чтобы начать, нужно бросить кубики!")
			time.Sleep(1 * time.Second)
			/*fmt.Println("Правила простые:\nИгра делится на раунды.\nРаунд начинается с бросков кубиков.\n1 равна 100 очкам,\n5 - 50 очков,\nКомбинация из трёх одинаковых кубиков = n*100.\nВ конце броска подсчитываем сумму.\nЕсли выпала комбинация и/или 1 и/или 5 - Очки суммируются.\nМожно записать очки в банк и передать ход сопернику, либо бросить кубики ещё раз.\n Если после броска сумма очков будет равна 0 - очки раунда обнуляются и ход передаётся сопернику.\nЕсли сумма больше 0 - она приплюсовывается к предыдущей.\nВыигрывает тот, кто первый наберёт 3000 очков!")
			time.Sleep(1 * time.Second)*/
			fmt.Println("Нажмите 'y', чтобы бросить кубики, 'q' чтобы выйти  ")
			isStart = false
		} else if roundScore != 0 {
			fmt.Printf("Осталось костей: %d\n", diceCount)
			fmt.Println("Перебрасываем ('y') или записываем на счёт ('p')? (q, чтобы выйти)")
		} else {
			time.Sleep(1 * time.Second)
			fmt.Println("Нажми 'y' чтобы сделать бросок (q, чтобы выйти)")
		}
		fmt.Scan(&choice)
		switch choice {
		case "q":
			fmt.Print("Игра окончена\n")
			return
		case "p":
			player.Bank += roundScore
			fmt.Printf("Вы забрали очки! Ваш счёт: %d\n", player.Bank)
			if player.Bank >= 3000 {
				fmt.Printf("Вы победили!\n")
				return
			} else {
				//передаём ход сопернику
				botMove(&bot)
				time.Sleep(2 * time.Second)
				if bot.Bank >= 3000 {
					fmt.Printf("%s победил!", bot.Name)
					return
				}
				fmt.Printf("Общий счёт (Раунд %d):\n%s набрал %d очков\n%s набрал %d очков\n", roundCount, player.Name, player.Bank, bot.Name, bot.Bank)
				time.Sleep(2 * time.Second)
				roundCount += 1
				roundScore = 0
				diceCount = 6
				fmt.Printf("------------Раунд %d---------------\n", roundCount)
			}
		case "y":
			fmt.Println("Бросаем кубики...")
			time.Sleep(2 * time.Second)
			dice := game.RollDice(diceCount)
			fmt.Println("Вам выпало:")
			counts := game.CountDice(dice)
			hasPriceDice := game.CheckPriceDice(counts)
			if !hasPriceDice {
				fmt.Printf("Потрачено:/\nВы потеряли очки за этот раунд. Текущий счёт: %d\n", player.Bank)
				roundScore = 0
				//передаём ход сопернику
				botMove(&bot)
				time.Sleep(2 * time.Second)
				if bot.Bank >= 3000 {
					fmt.Printf("%s победил!", bot.Name)
					return
				}
			} else {
				currentDices := make([]int, 0)
				for {
					var currentDice int
					fmt.Printf("Кости на столе: %d\n", dice)
					fmt.Println("Ваш выбор? (введите число или '0' чтобы сделать ещё бросок)")
					fmt.Scan(&currentDice)
					if currentDice == 0 {
						counts := game.CountDice(currentDices)
						score := game.CalculateScore(counts)
						roundScore += score
						break
					}
					//Проверяем наличие кости
					oldLen := len(dice)
					newDice := removeFirst(dice, currentDice)
					if len(newDice) == oldLen {
						fmt.Println("Такой кости нет на столе! Выбери другую.")
						continue
					}
					//Фиксируем выбор
					dice = newDice
					currentDices = append(currentDices, currentDice)
					diceCount -= 1
					fmt.Println("Ваш выбор:", currentDices)
					counts := game.CountDice(dice)
					hasPriceDice := game.CheckPriceDice(counts)
					//Проверка на горячие кости (если выпало сразу 6 призовых костей, то очки записываются и игрок может кидать ещё раз)
					if diceCount == 0 {
						fmt.Println("Горячие кости! Вы забрали сразу 6 кубиков!")
						diceCount = 6
						counts := game.CountDice(currentDices)
						score := game.CalculateScore(counts)
						roundScore += score
						break
					} else if !hasPriceDice {
						fmt.Println("На столе больше нет призовых костей")
						counts := game.CountDice(currentDices)
						score := game.CalculateScore(counts)
						roundScore += score
						break
					}
				}
			}
		default:
			fmt.Printf("Неожиданный ввод.\nНажми 'y', чтобы бросить кубики,\n'p', чтобы записать очки и передать ход,\n'q' чтобы выйти")
		}
	}
}
