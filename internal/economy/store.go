package economy

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
)

type PlayerStore interface {
	GetProfile(username string) *PlayerProfile
	UpdateBalance(username string, delta int)
	AddGameResult(username string, isWinner bool, moneyDelta int)
	GetAllProfiles() []*PlayerProfile
	DeductBalance(username string, amount int) error
}

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
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile, exists := s.Profiles[username]; exists {
		profile.Balance += delta
		s.save()
	}
}

func (s *Store) AddGameResult(username string, isWinner bool, moneyDelta int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile, exists := s.Profiles[username]; exists {
		profile.Balance += moneyDelta
		profile.Games++
		if isWinner {
			profile.Wins++
		}
		s.save() // вызываем приватный метод сохранения в файл
	}
}

func (s *Store) GetAllProfiles() []*PlayerProfile {
	s.mu.RLock() // разрешаем параллельное чтение
	defer s.mu.RUnlock()

	profiles := make([]*PlayerProfile, 0, len(s.Profiles))
	for _, p := range s.Profiles {
		profiles = append(profiles, p)
	}

	// Сортируем по количеству золота (по убыванию)
	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Balance > profiles[j].Balance
	})

	return profiles
}

func (s *Store) DeductBalance(username string, amount int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile, exists := s.Profiles[username]
	if !exists {
		return fmt.Errorf("профиль не найден")
	}

	if profile.Balance < amount {
		return fmt.Errorf("недостаточно золота")
	}

	profile.Balance -= amount
	s.save()

	return nil
}
