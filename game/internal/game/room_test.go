package game

import (
	"testing"
	"time"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/game/internal/models"
)

// MockStore заглушка для экономики
type MockStore struct{}

func (m *MockStore) DeductBalance(player string, amount int) error       { return nil }
func (m *MockStore) UpdateBalance(player string, amount int)             {}
func (m *MockStore) AddGameResult(player string, winner bool, delta int) {}
func (m *MockStore) UpdateAvatar(username string, avatar string) error   { return nil }
func (m *MockStore) GetProfile(player string) *models.PlayerProfile      { return &models.PlayerProfile{} }
func (m *MockStore) GetAllProfiles() []*models.PlayerProfile             { return nil }

func TestRoom_StartGame(t *testing.T) {
	store := &MockStore{}
	lobby := NewLobby(store)
	room := NewRoom("test_room", lobby, 100, 1000, store)

	client1 := &Client{PlayerName: "Player1", Send: make(chan []byte, 10)}
	client2 := &Client{PlayerName: "Player2", Send: make(chan []byte, 10)}

	room.AddClient(client1) // Становится создателем (Players[0])
	room.AddClient(client2) // Попадает в ожидание (PendingClient)

	// Даем горутинам время
	time.Sleep(10 * time.Millisecond)

	// ВАЖНО: Player1 должен принять заявку Player2
	room.HandleAction(client1, models.ActionMessage{Type: "ACCEPT_JOIN"})

	time.Sleep(10 * time.Millisecond)

	room.Mu.Lock()
	defer room.Mu.Unlock()

	if !room.IsStarted {
		t.Errorf("Ожидалось, что игра начнется после добавления двух игроков")
	}
	if len(room.Players) != 2 {
		t.Errorf("Ожидалось 2 игрока, получено %d", len(room.Players))
	}
	if room.Pot != 200 {
		t.Errorf("Ожидался банк 200, получено %d", room.Pot)
	}
}

func TestRoom_HandleAction_RollAndBank(t *testing.T) {
	store := &MockStore{}
	lobby := NewLobby(store)
	room := NewRoom("test_room", lobby, 100, 1000, store)

	client1 := &Client{PlayerName: "Player1", Send: make(chan []byte, 10)}
	client2 := &Client{PlayerName: "Player2", Send: make(chan []byte, 10)}

	room.AddClient(client1)
	room.AddClient(client2)
	time.Sleep(10 * time.Millisecond)

	// Player1 принимает заявку Player2
	room.HandleAction(client1, models.ActionMessage{Type: "ACCEPT_JOIN"})
	time.Sleep(10 * time.Millisecond)

	// Ходит Player1
	room.HandleAction(client1, models.ActionMessage{Type: "ROLL"})

	room.Mu.Lock()
	if !room.HasRolled {
		t.Errorf("Ожидался флаг HasRolled = true")
	}
	// Имитируем успешный бросок, подменяя кубики на выигрышные для теста Банка
	room.RoundScore = 300
	room.HasRolled = false
	room.Mu.Unlock()

	// Player1 банкует
	room.HandleAction(client1, models.ActionMessage{Type: "BANK"})
	time.Sleep(10 * time.Millisecond)

	room.Mu.Lock()
	if room.Banks[0] != 300 {
		t.Errorf("Ожидалось 300 очков в банке Player1, получено %d", room.Banks[0])
	}
	if room.CurrentTurn != 1 {
		t.Errorf("Ход должен был перейти к Player2 (индекс 1), текущий: %d", room.CurrentTurn)
	}
	room.Mu.Unlock()
}
