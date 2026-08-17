package game

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/economy"
	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
)

type Lobby struct {
	rooms     map[string]*Room
	mu        sync.Mutex
	store     *economy.Store // ссылка на хранилище
	betAmount int            // фиксированная ставка для этого лобби
}

func NewLobby(store *economy.Store, betAmount int) *Lobby {
	return &Lobby{
		rooms:     make(map[string]*Room),
		store:     store,
		betAmount: betAmount,
	}
}

// JoinOrCreateRoom находит комнату с 1 игроком или создаёт новую
func (l *Lobby) JoinOrCreateRoom(client *Client) *Room {
	// 1. Предварительная проверка баланса
	// Если монет не хватает, отправляем ошибку напрямую клиенту и не пускаем в лобби
	profile := l.store.GetProfile(client.PlayerName)
	if profile.Balance < l.betAmount {
		errMsg, _ := json.Marshal(models.EventMessage{
			Type:    "ERROR",
			Message: fmt.Sprintf("Недостаточно золота! Нужна ставка: %d монет.", l.betAmount),
		})

		select {
		case client.Send <- errMsg:
		default:
		}

		return nil // возвращаем nil, игрок не добавлен ни в одну комнату
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	//2. Ищем существующую комнату, где ждёт 1 игрок
	for _, room := range l.rooms {
		room.Mu.Lock()
		if len(room.Clients) == 1 && !room.IsStarted {
			room.Clients[client] = true
			room.Players = append(room.Players, client.PlayerName)
			room.Mu.Unlock()
			// Запускаем игру, так как набралось 2 игрока
			room.StartGame()
			return room
		}
		room.Mu.Unlock()
	}

	//2. Если свободных комнат нет - создаём новую
	roomID := fmt.Sprintf("room_%d", len(l.rooms)+1)
	room := NewRoom(roomID, l, l.betAmount, l.store)
	room.Clients[client] = true
	room.Players = append(room.Players, client.PlayerName)

	l.rooms[roomID] = room
	return room
}

// RemoveRoom удаляет пустую комнату из лобби
func (l *Lobby) RemoveRoom(roomID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	delete(l.rooms, roomID)
}
