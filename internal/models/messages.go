package models

// Сообщения от клиента серверу
type ActionMessage struct {
	Type       string `json:"type"`
	Dice       []int  `json:"dice,omitempty"` // используется для SELECT_DICE
	PlayerName string `json:"player_name,omitempty"`
}

// Сообщения от сервера клиенту
type EventMessage struct {
	Type         string         `json:"type"`
	Message      string         `json:"message,omitempty"`
	Dice         []int          `json:"dice,omitempty"`  // Кубики на столе
	Score        int            `json:"score,omitempty"` // Очки за раунд
	Banks        map[string]int `json:"banks,omitempty"` // Банки обоих игроков (имя, очки)
	ActivePlayer string         `json:"active_player,omitempty"`
}
