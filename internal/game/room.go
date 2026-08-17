package game

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
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

	// Игровое состояние комнаты
	IsStarted   bool     // Старт игры
	CurrentTurn int      // индекс текущего игрока (0 или 1)
	Players     []string // Имена двух игроков
	Dice        []int    // Текущие кубики на столе
	DiceCount   int      // Количество кубиков для следующего броска
	RoundScore  int      // Очки за текущий раунд
	Banks       []int    // Несгораемые банки игроков [Банк_0, Банк_1]
	HasRolled   bool     // Флаг броска
}

func NewRoom(id string, lobby *Lobby) *Room {
	return &Room{
		ID:        id,
		Clients:   make(map[*Client]bool),
		lobby:     lobby,
		DiceCount: 6,
		Banks:     make([]int, 2),
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
			close(client.Send)
			delete(r.Clients, client)
		}
	}
}

func (r *Room) StartGame() {
	r.Mu.Lock()
	r.IsStarted = true
	r.CurrentTurn = 0
	r.Mu.Unlock()
	currentBanks := map[string]int{
		r.Players[0]: r.Banks[0],
		r.Players[1]: r.Banks[1],
	}

	r.Broadcast(models.EventMessage{
		Type:         "GAME_STARTED",
		Message:      fmt.Sprintf("Игра началась! Первым ходит %s", r.Players[0]),
		Banks:        currentBanks,
		ActivePlayer: r.Players[0],
	})
}

func (r *Room) Leave(client *Client) {
	r.Mu.Lock()
	delete(r.Clients, client)
	isEmpty := len(r.Clients) == 0
	r.Mu.Unlock()

	if isEmpty {
		r.lobby.RemoveRoom(r.ID)
	} else {
		r.Broadcast(models.EventMessage{
			Type:    "PLAYER_LEFT",
			Message: fmt.Sprintf("Игрок %s покинул игру.", client.PlayerName),
		})
	}
}

func (r *Room) sendPrivateError(client *Client, msg string) {
	data, _ := json.Marshal(models.EventMessage{
		Type:    "ERROR",
		Message: msg,
	})
	select {
	case client.Send <- data:
	default:
	}
}

// HandleAction обрабатывает игровой ход одного из игроков
func (r *Room) HandleAction(client *Client, action models.ActionMessage) {
	r.Mu.Lock()

	// 1. Если игра ещё не началась (ждём второго игрока)
	if !r.IsStarted {
		r.Mu.Unlock()
		r.sendPrivateError(client, "Ожидаем второго игрока...")
		return
	}

	// 2. Проверяем, чей сейчас ход
	activePlayer := r.Players[r.CurrentTurn]
	if client.PlayerName != activePlayer {
		r.Mu.Unlock()
		r.sendPrivateError(client, "Сейчас ход вашего соперника!")
		return
	}

	// 3. Выполнение действий
	var events []models.EventMessage // список событий для рассылки

	switch action.Type {
	case "ROLL":
		if r.HasRolled {
			r.Mu.Unlock()
			r.sendPrivateError(client, "Бросок уже сделан! Сначала отложите призовые кубики.")
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
				Type:    "ZONK",
				Message: "Пу-пу-пуууууу:/ Очки раунда сгорают.",
				Dice:    r.Dice,
				Score:   0,
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
			r.sendPrivateError(client, "Ошибка: попытка выбрать неверные кубики!")
			return
		}

		// 2. Считаем очки
		counts := CountDice(action.Dice)
		addedScore := CalculateScore(counts)

		if addedScore == 0 {
			r.Mu.Unlock()
			r.sendPrivateError(client, "Эти кубики не приносят очков!")
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
				r.sendPrivateError(client, "Ошибка: попытка выбрать неверные кубики!")
				return
			}

			counts := CountDice(action.Dice)
			addedScore := CalculateScore(counts)

			if addedScore == 0 {
				r.Mu.Unlock()
				r.sendPrivateError(client, "Эти кубики не приносят очков!")
				return
			}

			r.RoundScore += addedScore
			r.Dice = remainingDice
			r.DiceCount -= len(action.Dice)
		}

		// Если очки раунда всё ещё равны 0 (ничего не выбрано), сообщаем о необходимости выбрать кубики
		if r.RoundScore == 0 {
			r.Mu.Unlock()
			r.sendPrivateError(client, "Необходимо выбрать призовые кубики!")
			return
		}

		r.HasRolled = false
		r.Banks[r.CurrentTurn] += r.RoundScore

		// Проверка на победу!
		if r.Banks[r.CurrentTurn] >= 3000 {
			events = append(events, models.EventMessage{
				Type:    "GAME_OVER",
				Message: fmt.Sprintf("Игрок %s победил, набрав %d очков!", activePlayer, r.Banks[r.CurrentTurn]),
			})
		} else {
			r.RoundScore = 0
			r.DiceCount = 6
			r.CurrentTurn = (r.CurrentTurn + 1) % 2
			currentBanks := map[string]int{
				r.Players[0]: r.Banks[0],
				r.Players[1]: r.Banks[1],
			}
			events = append(events, models.EventMessage{
				Type:         "TURN_CHANGED",
				Message:      fmt.Sprintf("Ход перешёл к %s", r.Players[r.CurrentTurn]),
				Banks:        currentBanks,
				ActivePlayer: r.Players[r.CurrentTurn],
			})
		}
	}

	r.Mu.Unlock() // Освобождаем мьютекс до рассылки

	// Рассылаем все накопленные события всем игрокам
	for _, e := range events {
		r.Broadcast(e)
	}
}
