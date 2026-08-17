package economy

import (
	"encoding/json"
	"os"
	"sync"
)

type PlayerProfile struct {
	Username string `json:"username"`
	Balance  int    `json:"balance"` // Текущее золото
	Wins     int    `json:"wins"`    // Количество побед
	Games    int    `json:"games"`   // Всего сыграно партий
}

type Store struct {
	mu       sync.RWMutex
	filePath string
	Profiles map[string]*PlayerProfile
}

func NewStore(filePath string) *Store {
	store := &Store{
		filePath: filePath,
		Profiles: make(map[string]*PlayerProfile),
	}
	store.load()
	return store
}

func (s *Store) load() {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.filePath)
	if err != nil {
		// если файла нет, просто начинаем с пустой мапы
		return
	}
	json.Unmarshal(data, &s.Profiles)
}

// Сохранение данных в файл
func (s *Store) save() {
	data, _ := json.MarshalIndent(s.Profiles, "", " ")
	os.WriteFile(s.filePath, data, 0644)
}

// Получить профиль или создать новый (даем стартовый капитал)
func (s *Store) GetProfile(username string) *PlayerProfile {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, exists := s.Profiles[username]
	if !exists {
		profile = &PlayerProfile{
			Username: username,
			Balance:  1000, // Стартовые 1000 монет для новичков
			Wins:     0,
			Games:    0,
		}
		s.Profiles[username] = profile
		s.save()
	}
	return profile
}

// Обновление баланса после победы/поражения
func (s *Store) UpdateBalance(username string, delta int) {
	s.mu.Unlock()

	if profile, exists := s.Profiles[username]; exists {
		profile.Balance += delta
		s.save()
	}
}
