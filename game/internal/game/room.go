package game

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/models"
	"github.com/gorilla/websocket"
)

// Client - обертка над WebSocket-соединением игрока
type Client struct {
	Conn       *websocket.Conn
	PlayerName string
	Send       chan []byte
}

// Room - комната для двух игроков с состоянием игры
type Room struct {
	ID      string
	Clients map[*Client]bool
	Mu      sync.Mutex // защита от гонки данных при одновременных запросах
	lobby   *Lobby     // Ссылка на родительский менеджер комнат

	// Экономика
	BetAmount int                 // Размер ставки
	TargetScore int		      // Очки для победы
	Pot       int                 // общий банк
	Store     economy.PlayerStore // ссылка на хранилище

	// Игровое состояние комнаты
	IsStarted   bool                   // Старт игры
	CurrentTurn int                    // индекс текущего игрока (0 или 1)
	Players     []string               // Имена двух игроков
	Dice        []int                  // Текущие кубики на столе
	DiceCount   int                    // Количество кубиков для следующего броска
	RoundScore  int                    // Очки за текущий раунд
	Banks       []int                  // Несгораемые банки игроков [Банк_0, Банк_1]
	HasRolled   bool                   // Флаг броска
	TurnTimer   *time.Timer            // Таймер на ход 90 секунд
	Disconnects map[string]*time.Timer // Таймеры переподключения

	//Подключающийся игрок
	PendingClient *Client
	PendingPlayer string
}

func NewRoom(id string, lobby *Lobby, betAmount int, targetScore int, store economy.PlayerStore) *Room {
	return &Room{
		ID:          id,
		Clients:     make(map[*Client]bool),
		Disconnects: make(map[string]*time.Timer),
		lobby:       lobby,
		DiceCount:   6,
		Banks:       make([]int, 2),
		BetAmount:   betAmount,
		TargetScore: targetScore,
		Store:       store,
	}
}

func (r *Room) sendToClient(client *Client, event models.EventMessage) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Ошибка маршаллинга сообщения: %v", err)
		return
	}

	select {
	case client.Send <- data:
	default:
		log.Printf("Не удалось отправить сообщение клиенту %s: буфер переполнен или канал закрыт", client.PlayerName)
	}
}

func (r *Room) AddClient(client *Client) {
	playerName := client.PlayerName

	// 1. Проверяем наличие игрока и списание средств
	r.Mu.Lock()
	found := false
	for _, p := range r.Players {
		if p == playerName {
			found = true
			break
		}
	}
	// Если игрок новый - считываем золото транзакций до добавления в активный состав
	if !found {
		if len(r.Players) >= 2 {
			r.Mu.Unlock()
			r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Комната заполнена!"})
			client.Conn.Close()
			return
		}

	// Если это второй игрок (подключается к создателю) - отправляем запрос
		if len(r.Players) == 1 && !r.IsStarted {
			r.PendingClient = client
			r.PendingPlayer = playerName
			
			// Находим клиента-создателя для отправки запроса
			for c := range r.Clients {
				if c.PlayerName == r.Players[0] {
					r.sendToClient(c, models.EventMessage{
						Type:    "JOIN_REQUEST",
						Message: playerName,
					})
					break
				}
			}
			r.Mu.Unlock()
			return
		}

		// Вызываем списание вне жесткой блокировки состояния комнаты
		r.Mu.Unlock()
		if err := r.Store.DeductBalance(playerName, r.BetAmount); err != nil {
			r.sendToClient(client, models.EventMessage{
				Type:    "ERROR",
				Message: "Недостаточно золота для входа в игру!",
			})
			client.Conn.Close()
			return
		}
		r.Mu.Lock()
		r.Players = append(r.Players, playerName)
	}

	r.Clients[client] = true
	// Считаем, сколько именно активных WS-соединений сейчас в комнате
	activeConnections := len(r.Clients)
	isStarted := r.IsStarted

	// 2. Логика переподключения
	isReconnection := false
	if timer, exists := r.Disconnects[playerName]; exists {
		timer.Stop() // останавливаем таймер сдачи
		delete(r.Disconnects, playerName)
		isReconnection = true
	}

	// 3. Отправляем состояние игры, если это реконнект
	if isReconnection && isStarted {
		currentBanks := map[string]int{
			r.Players[0]: r.Banks[0],
			r.Players[1]: r.Banks[1],
		}

		restoreMsg := models.EventMessage{
			Type:         "GAME_RESTORED",
			Message:      "Вы успешно переподключились к игре!",
			Banks:        currentBanks,
			ActivePlayer: r.Players[r.CurrentTurn],
			Pot:          r.Pot,
			Dice:         r.Dice,
			Score:        r.RoundScore,
		}
		r.Mu.Unlock()

		// Отправляем восстановленное состояние лично этому игроку
		r.sendToClient(client, restoreMsg)

		// Оповещаем противника, что игрок вернулся
		r.Broadcast(models.EventMessage{
			Type:    "SYSTEM",
			Message: fmt.Sprintf("Игрок %s вернулся в игру!", playerName),
		})
		return
	}

	r.Mu.Unlock()

	// Оповещаем всех (включая только что зашедшего), что кто-то присоединился
	r.Broadcast(models.EventMessage{
		Type:    "SYSTEM",
		Message: fmt.Sprintf("Игрок %s подключился к столу!", client.PlayerName),
	})

	// Если оба игрока успешно установили WebSocket - соединение и игра ещё не идет
	r.Mu.Lock()
	shouldStart := activeConnections == 2 && !r.IsStarted
	r.Mu.Unlock()

	if shouldStart {
		r.StartGame()
	}
}

// Broadcast отправляет сообщение всем игрокам в комнате
func (r *Room) Broadcast(event models.EventMessage) {
	data, err := json.Marshal(event)
	if err != nil {
		log.Printf("Ошибка маршалинга при рассылке: %v", err)
		return
	}

	r.Mu.Lock()
	defer r.Mu.Unlock()

	for client := range r.Clients {
		select {
		case client.Send <- data:
		default:
			delete(r.Clients, client)
		}
	}
}

// === ВСПОМОГАТЕЛЬНЫЕ МЕТОДЫ ТАЙМЕРА И ЗАВЕРШЕНИЯ ИГРЫ ===

// resetTurnTimerLocked перезапускает таймер на 90 секунд.
// ВАЖНО: Вызывать только при заблокированном r.Mu!
func (r *Room) resetTurnTimerLocked() {
	if r.TurnTimer != nil {
		r.TurnTimer.Stop()
	}

	activePlayer := r.Players[r.CurrentTurn]

	r.TurnTimer = time.AfterFunc(90*time.Second, func() {
		r.Mu.Lock()
		// Если игра уже закончилась или ход сменился раньше времени
		if !r.IsStarted || r.Players[r.CurrentTurn] != activePlayer {
			r.Mu.Unlock()
			return
		}

		r.HasRolled = false

		// Автоматически сохраняем отложенные очки в банк (даже если там 0)
		bankedScore := r.RoundScore
		r.Banks[r.CurrentTurn] += r.RoundScore

		var events []models.EventMessage
		var gameResults []struct {
			player string
			winner bool
			delta  int
		}

		// Проверяем, не набрал ли игрок 3000 очков за счет этого авто-сохранения
		if r.Banks[r.CurrentTurn] >= 3000 {
			r.stopTurnTimerLocked()
			r.IsStarted = false

			for _, pName := range r.Players {
				isWinner := (pName == activePlayer)
				moneyDelta := 0
				if isWinner {
					moneyDelta = r.Pot
				}
				gameResults = append(gameResults, struct {
					player string
					winner bool
					delta  int
				}{pName, isWinner, moneyDelta})
				r.Store.AddGameResult(pName, isWinner, moneyDelta)
			}

			events = append(events, models.EventMessage{
				Type:    "GAME_OVER",
				Message: fmt.Sprintf("Время вышло! Отложенные очки принесли победу! Игрок %s набрал %d очков!", activePlayer, r.Banks[r.CurrentTurn]),
			})
		} else {
			// Если очков для победы не хватило — просто передаем ход противнику
			r.RoundScore = 0
			r.DiceCount = 6
			r.CurrentTurn = (r.CurrentTurn + 1) % 2
			r.Dice = nil

			r.resetTurnTimerLocked() // Сбрасываем таймер уже для нового игрока

			currentBanks := map[string]int{
				r.Players[0]: r.Banks[0],
				r.Players[1]: r.Banks[1],
			}
			events = append(events, models.EventMessage{
				Type:         "TURN_CHANGED",
				Message:      fmt.Sprintf("⏰ Время вышло! Игрок %s забанковал %d очков. Ход перешёл к %s", activePlayer, bankedScore, r.Players[r.CurrentTurn]),
				Banks:        currentBanks,
				ActivePlayer: r.Players[r.CurrentTurn],
				Score:        bankedScore,
			})
		}
		r.Mu.Unlock() // Обязательно отпускаем мьютекс ДО рассылки Broadcast

		// Выполняем запись результатов в хранилище без блокировки состояния комнаты
		for _, res := range gameResults {
			r.Store.AddGameResult(res.player, res.winner, res.delta)
		}

		// Рассылаем события о смене хода или победе всем игрокам
		for _, e := range events {
			r.Broadcast(e)
		}
	})
}

// stopTurnTimerLocked останавливает таймер.
// ВАЖНО: Вызывать только при заблокированном r.Mu!
func (r *Room) stopTurnTimerLocked() {
	if r.TurnTimer != nil {
		r.TurnTimer.Stop()
		r.TurnTimer = nil
	}
}

// finishGameBySurrenderLocked начисляет победу сопернику и завершает игру.
// ВАЖНО: Принимает r.Mu заблокированным, но САМ ОСВОБОЖДАЕТ его перед вызовом Broadcast!
func (r *Room) finishGameBySurrenderLocked(surrenderedPlayer string, reason string) {
	r.stopTurnTimerLocked()
	r.IsStarted = false
	r.lobby.RemoveRoom(r.ID)

	var winner string
	for _, p := range r.Players {
		if p != surrenderedPlayer {
			winner = p
			break
		}
	}

	var gameResults []struct {
		player string
		winner bool
		delta  int
	}

	// Начисляем куш победителю и обновляем статистику в БД
	for _, pName := range r.Players {
		isWinner := (pName == winner)
		moneyDelta := 0
		if isWinner {
			moneyDelta = r.Pot
		}
		gameResults = append(gameResults, struct {
			player string
			winner bool
			delta  int
		}{pName, isWinner, moneyDelta})
	}

	// Отпускаем мьютекс ДО рассылки, так как r.Broadcast внутри сам вызывает r.Mu.Lock()
	r.Mu.Unlock()

	for _, res := range gameResults {
		r.Store.AddGameResult(res.player, res.winner, res.delta)
	}

	r.Broadcast(models.EventMessage{
		Type:    "GAME_OVER",
		Message: fmt.Sprintf("%s Победил игрок %s!", reason, winner),
	})
}

func (r *Room) StartGame() {
	// Считывание данных профилей перед блокировкой состояния
	player1Profile := r.Store.GetProfile(r.Players[0])
	player2Profile := r.Store.GetProfile(r.Players[1])

	avatar1 := player1Profile.Avatar
	if avatar1 == "" {
		avatar1 = "monk"
	}

	avatar2 := player2Profile.Avatar
	if avatar2 == "" {
		avatar2 = "monk"
	}

	r.Mu.Lock()
	r.IsStarted = true
	r.CurrentTurn = 0
	r.Pot = r.BetAmount * len(r.Players)

	currentBanks := map[string]int{
		r.Players[0]: r.Banks[0],
		r.Players[1]: r.Banks[1],
	}

	avatarsMap := map[string]string{
		r.Players[0]: avatar1,
		r.Players[1]: avatar2,
	}
	// запускаем таймер на 90 секунд для первого игрока
	r.resetTurnTimerLocked()
	r.Mu.Unlock()

	r.Broadcast(models.EventMessage{
		Type:         "GAME_STARTED",
		Message:      fmt.Sprintf("Игра началась! Первым ходит %s", r.Players[0]),
		Banks:        currentBanks,
		ActivePlayer: r.Players[0],
		Pot:          r.Pot,
		Avatars:      avatarsMap,
	})
}

func (r *Room) Leave(client *Client) {
	r.Mu.Lock()

	// Безопасное удаление: закрываем канал только 1 раз здесь, чтобы избежать паники
	if _, ok := r.Clients[client]; ok {
		delete(r.Clients, client)
		close(client.Send)
	}
	isEmpty := len(r.Clients) == 0
	playerName := client.PlayerName

	// Если игра ЕЩЕ НЕ началась (кто-то вышел из лобби до старта)
	if !r.IsStarted {
		// Если комната пуста до старта игры — возвращаем взнос первому подключившемуся игроку
		r.Store.UpdateBalance(playerName, r.BetAmount)
		r.Mu.Unlock()

		if isEmpty {
			r.lobby.RemoveRoom(r.ID)
		} else {
			r.Broadcast(models.EventMessage{
				Type:    "PLAYER_LEFT",
				Message: fmt.Sprintf("Игрок %s покинул стол.", playerName),
			})
		}
		return
	}

	// Если игра ИДЁТ - даём 60 секунд на переподключение
	timer := time.AfterFunc(60*time.Second, func() {
		r.Mu.Lock()
		// Если таймера нет в мапе, значит игрок уже переподключился (AddClient его удалил)
		if _, exists := r.Disconnects[playerName]; !exists {
			r.Mu.Unlock()
			return
		}
		delete(r.Disconnects, playerName)

		// Игрок так и не вернулся - засчитываем техническое поражение
		r.finishGameBySurrenderLocked(playerName, fmt.Sprintf("Игрок %s покинул игру (Тайм-аут подключения).", playerName))
	})

	r.Disconnects[playerName] = timer
	r.Mu.Unlock()

	// Сообщаем противнику, что игрок отвалился
	r.Broadcast(models.EventMessage{
		Type:         "PLAYER_DISCONNECTED",
		Message:      fmt.Sprintf("Игрок %s отключился. Ожидание переподключения (60 сек)...", playerName),
		ActivePlayer: playerName,
	})
}

// HandleAction обрабатывает игровой ход одного из игроков
func (r *Room) HandleAction(client *Client, action models.ActionMessage) {
	r.Mu.Lock()

	// 1. Если игра ещё не началась (ждём второго игрока)
	if !r.IsStarted {
		r.Mu.Unlock()
		r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Ожидаем второго игрока..."})
		return
	}

	// === ДЕЙСТВИЯ, ДОСТУПНЫЕ В ЛЮБОЙ МОМЕНТ (Даже не в свой ход) ===
	switch action.Type {
	case "CHAT":
		r.Mu.Unlock()
		r.Broadcast(models.EventMessage{
			Type:         "CHAT",
			Message:      action.Message,
			ActivePlayer: client.PlayerName,
		})
		return

	case "SURRENDER":
		r.finishGameBySurrenderLocked(client.PlayerName, fmt.Sprintf("Игрок %s сдался.", client.PlayerName))
		return


	case "ACCEPT_JOIN":
		if r.PendingPlayer == "" || client.PlayerName != r.Players[0] {
			r.Mu.Unlock()
			return
		}
		r.Mu.Unlock() // Отпускаем мьютекс для транзакции
		if err := r.Store.DeductBalance(r.PendingPlayer, r.BetAmount); err != nil {
			r.sendToClient(r.PendingClient, models.EventMessage{Type: "ERROR", Message: "У игрока недостаточно золота!"})
			r.PendingClient.Conn.Close()
			r.Mu.Lock()
			r.PendingClient = nil
			r.PendingPlayer = ""
			r.Mu.Unlock()
			return
		}

		r.Mu.Lock()
		r.Players = append(r.Players, r.PendingPlayer)
		r.Clients[r.PendingClient] = true
		r.PendingClient = nil
		r.PendingPlayer = ""
		r.Mu.Unlock()
		r.StartGame()
		return

	case "REJECT_JOIN":
		if r.PendingPlayer == "" || client.PlayerName != r.Players[0] {
			r.Mu.Unlock()
			return
		}
		r.sendToClient(r.PendingClient, models.EventMessage{Type: "JOIN_REJECTED"})
		r.PendingClient.Conn.Close()
		r.PendingClient = nil
		r.PendingPlayer = ""
		r.Mu.Unlock()
		return
	}

	// 2. Проверка очередности хода
	activePlayer := r.Players[r.CurrentTurn]
	if client.PlayerName != activePlayer {
		r.Mu.Unlock()
		r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Сейчас ход вашего соперника!"})
		return
	}

	// 3. Выполнение действий
	var events []models.EventMessage // список событий для рассылки
	var gameResults []struct {
		player string
		winner bool
		delta  int
	}

	switch action.Type {
	case "ROLL":
		if r.HasRolled {
			r.Mu.Unlock()
			r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Бросок уже сделан! Сначала отложите призовые кубики."})
			return
		}
		r.Dice = RollDice(r.DiceCount)
		r.HasRolled = true
		counts := CountDice(r.Dice)

		// Проверка на отсутствие призовых костей
		if !CheckPriceDice(counts) {
			r.HasRolled = false
			r.RoundScore = 0
			r.DiceCount = 6
			r.CurrentTurn = (r.CurrentTurn + 1) % 2 // ход переходит
			currentBanks := map[string]int{
				r.Players[0]: r.Banks[0],
				r.Players[1]: r.Banks[1],
			}

			events = append(events, models.EventMessage{
				Type:         "ZONK",
				Message:      "Пу-пу-пуууууу:/ Очки раунда сгорают.",
				Dice:         r.Dice,
				Score:        0,
				ActivePlayer: r.Players[r.CurrentTurn],
			})
			events = append(events, models.EventMessage{
				Type:         "TURN_CHANGED",
				Message:      fmt.Sprintf("Ход перешел к %s", r.Players[r.CurrentTurn]),
				Banks:        currentBanks,
				ActivePlayer: r.Players[r.CurrentTurn],
			})
		} else {
			events = append(events, models.EventMessage{
				Type:         "DICE_ROLLED",
				Dice:         r.Dice,
				Score:        r.RoundScore,
				ActivePlayer: activePlayer,
			})
		}
	case "SELECT_DICE":
		// 1. Проверяем валидность выбора
		remainingDice, valid := removeSelectedDice(r.Dice, action.Dice)
		if !valid {
			r.Mu.Unlock()
			r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Ошибка: попытка выбрать неверные кубики!"})
			return
		}

		// 2. Считаем очки
		counts := CountDice(action.Dice)
		addedScore := CalculateScore(counts)

		if addedScore == 0 {
			r.Mu.Unlock()
			r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Эти кубики не приносят очков!"})
			return
		}

		// 3. Обновляем состояние
		r.Dice = remainingDice
		r.RoundScore += addedScore
		r.DiceCount -= len(action.Dice)
		r.HasRolled = false

		// 4. Проверка на Мааае почтение
		if len(r.Dice) == 0 {
			r.DiceCount = 6
			events = append(events, models.EventMessage{
				Type:         "MY_RESPECTS",
				Message:      "Мааааё почтение! Бросайте ещё 6 кубиков!",
				Score:        r.RoundScore,
				ActivePlayer: activePlayer,
			})
		} else {
			events = append(events, models.EventMessage{
				Type:         "DICE_SELECTED",
				Dice:         r.Dice,
				Score:        r.RoundScore,
				ActivePlayer: activePlayer,
			})
		}
	case "BANK":
		// Если игрок выделил кубики прямо перед нажатием "В банк", обрабатываем их автоматически
		if len(action.Dice) > 0 {
			remainingDice, valid := removeSelectedDice(r.Dice, action.Dice)
			if !valid {
				r.Mu.Unlock()
				r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Ошибка: попытка выбрать неверные кубики!"})
				return
			}

			counts := CountDice(action.Dice)
			addedScore := CalculateScore(counts)

			if addedScore == 0 {
				r.Mu.Unlock()
				r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Эти кубики не приносят очков!"})
				return
			}

			r.RoundScore += addedScore
			r.Dice = remainingDice
			r.DiceCount -= len(action.Dice)
		}

		// Если очки раунда всё ещё равны 0 (ничего не выбрано), сообщаем о необходимости выбрать кубики
		if r.RoundScore == 0 {
			r.Mu.Unlock()
			r.sendToClient(client, models.EventMessage{Type: "ERROR", Message: "Необходимо выбрать призовые кубики!"})
			return
		}

		r.HasRolled = false
		bankedScore := r.RoundScore
		bankedPlayer := activePlayer

		r.Banks[r.CurrentTurn] += r.RoundScore

		// Проверка на победу!
		if r.Banks[r.CurrentTurn] >= 3000 {
			r.stopTurnTimerLocked()
			r.IsStarted = false
			r.lobby.RemoveRoom(r.ID)
			// 1. Выдаём куш победителю и засчитываем игру обоим
			for _, pName := range r.Players {
				isWinner := (pName == activePlayer)
				moneyDelta := 0
				if isWinner {
					moneyDelta = r.Pot // Победитель забирает банк
				}

				// Обновляем всё за один вызов
				gameResults = append(gameResults, struct {
					player string
					winner bool
					delta  int
				}{pName, isWinner, moneyDelta})
			}
			r.Dice = nil

			events = append(events, models.EventMessage{
				Type:    "GAME_OVER",
				Message: fmt.Sprintf("Игрок %s победил, набрав %d очков!", activePlayer, r.Banks[r.CurrentTurn]),
			})
		} else {
			r.RoundScore = 0
			r.DiceCount = 6
			r.Dice = nil
			r.CurrentTurn = (r.CurrentTurn + 1) % 2

			r.resetTurnTimerLocked() // сброс таймера при передаче хода
			currentBanks := map[string]int{
				r.Players[0]: r.Banks[0],
				r.Players[1]: r.Banks[1],
			}
			events = append(events, models.EventMessage{
				Type:         "TURN_CHANGED",
				Message:      fmt.Sprintf("🏦 Игрок %s забанковал %d очков! Ход перешёл к %s", bankedPlayer, bankedScore, r.Players[r.CurrentTurn]),
				Banks:        currentBanks,
				ActivePlayer: r.Players[r.CurrentTurn],
				Score:        bankedScore,
			})
		}
	}

	r.Mu.Unlock() // Освобождаем мьютекс до рассылки

	// Рассылаем все накопленные события всем игрокам
	for _, e := range events {
		r.Broadcast(e)
	}
}
