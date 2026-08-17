package game

import (
	"fmt"
	"sync"
)

type Lobby struct {
	rooms map[string]*Room
	mu    sync.Mutex
}

func NewLobby() *Lobby {
	return &Lobby{
		rooms: make(map[string]*Room),
	}
}

// JoinOrCreateRoom находит комнату с 1 игроком или создаёт новую
func (l *Lobby) JoinOrCreateRoom(client *Client) *Room {
	l.mu.Lock()
	defer l.mu.Unlock()

	//1. Ищем существующую комнату, где ждёт 1 игрок
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
	room := NewRoom(roomID, l)
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
