package game

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/economy"
)

type RoomInfo struct {
	ID        string `json:"id"`
	Creator   string `json:"creator"`
	BetAmount int    `json:"bet_amount"`
	TergetScore int `json:"target_score"`
}

type Lobby struct {
	mu    sync.RWMutex
	Rooms map[string]*Room
	Store economy.PlayerStore
}

func NewLobby(store economy.PlayerStore) *Lobby {
	return &Lobby{
		Rooms: make(map[string]*Room),
		Store: store,
	}
}

// GenerateID создает случайную строку для ID комнаты
func generateID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func (l *Lobby) CreateRoom(creator string, betAmount int, targetScore int) (*Room, error) {
	err := l.Store.DeductBalance(creator, betAmount)
	if err != nil {
		return nil, err
	}

	id := generateID()
	room := NewRoom(id, l, betAmount, targetScore, l.Store)
	// Добавляем создателя в список игроков (пока он один)
	room.Players = append(room.Players, creator)

	l.mu.Lock()
	defer l.mu.Unlock()
	l.Rooms[id] = room

	return room, nil
}

func (l *Lobby) GetAvailableRooms() []RoomInfo {
	l.mu.RLock() // Используем RLock для параллельного чтения
	defer l.mu.RUnlock()

	var available []RoomInfo
	for _, room := range l.Rooms {
		// Показываем только те комнаты, где ждем второго игрока
		if !room.IsStarted && len(room.Players) == 1 {
			available = append(available, RoomInfo{
				ID:        room.ID,
				Creator:   room.Players[0],
				BetAmount: room.BetAmount,
			})
		}
	}
	return available
}

func (l *Lobby) RemoveRoom(id string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.Rooms, id)
}

func (l *Lobby) JoinRoom(roomID string, username string) (*Room, error) {
	l.mu.RLock()
	room, exists := l.Rooms[roomID]
	l.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("комната не найдена")
	}

	// Блокируем саму комнату, чтобы никто другой не успел занять место
	room.Mu.Lock()
	defer room.Mu.Unlock()
	// Если игрок уже записан в комнату (например, создатель), повторно деньги не списываем
	for _, p := range room.Players {
		if p == username {
			return room, nil
		}
	}
	// ПРоверяем, есть ли ещё место
	if room.IsStarted || len(room.Players) >= 2 {
		return nil, fmt.Errorf("комната уже заполнена или игра началась")
	}

	// Транзакция - пытаемся списать деньги
	err := l.Store.DeductBalance(username, room.BetAmount)
	if err != nil {
		return nil, err // деньги не списались - игрок не добавлен
	}

	// деньги успешно списаны - добавляем игрока
	room.Players = append(room.Players, username)

	return room, nil
}

func (l *Lobby) GetRoom(id string) (*Room, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	room, exists := l.Rooms[id]
	if !exists {
		return nil, fmt.Errorf("комната с ID %s не найдена", id)
	}

	return room, nil
}

func (l *Lobby) FindRoomByDisconnectedPlayer(username string) *Room {
	l.mu.RLock()
	defer l.mu.RUnlock()

	for _, room := range l.Rooms {
		room.Mu.Lock()
		_, exists := room.Disconnects[username]
		room.Mu.Unlock()

		if exists {
			return room
		}
	}
	return nil
}
