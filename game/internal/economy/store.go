package economy

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/ValeriyOrlov/NeeKoobaMeemo/internal/models"
	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

type PlayerStore interface {
	GetProfile(username string) *models.PlayerProfile
	UpdateBalance(username string, delta int)
	AddGameResult(username string, isWinner bool, moneyDelta int)
	GetAllProfiles() []*models.PlayerProfile
	DeductBalance(username string, amount int) error
	UpdateAvatar(username string, avatar string) error
}

type DBStore struct {
	db *sql.DB
}

func NewDBStore(db *sql.DB) *DBStore {
	return &DBStore{db: db}
}

// GetProfile получает профиль или создает новый с еженедельной проверкой бонуса
func (s *DBStore) GetProfile(username string) *models.PlayerProfile {
	profile := &models.PlayerProfile{}

	query := `
		SELECT username, balance, wins, games, avatar, last_weekly_claim 
		FROM player_profiles 
		WHERE username = $1`

	err := s.db.QueryRow(query, username).Scan(
		&profile.Username,
		&profile.Balance,
		&profile.Wins,
		&profile.Games,
		&profile.Avatar,
		&profile.LastWeeklyClaim,
	)

	// Если игрока еще нет в БД — создаем его
	if errors.Is(err, sql.ErrNoRows) {
		now := time.Now()
		insertQuery := `
			INSERT INTO player_profiles (username, balance, wins, games, avatar, last_weekly_claim)
			VALUES ($1, 1000, 0, 0, 'fat_cat', $2)
			RETURNING username, balance, wins, games, avatar, last_weekly_claim`

		_ = s.db.QueryRow(insertQuery, username, now).Scan(
			&profile.Username,
			&profile.Balance,
			&profile.Wins,
			&profile.Games,
			&profile.Avatar,
			&profile.LastWeeklyClaim,
		)
		return profile
	}

	// Проверяем еженедельное подкрепление
	if isNewWeek(profile.LastWeeklyClaim) {
		profile.LastWeeklyClaim = time.Now()
		if profile.Balance < 1000 {
			profile.Balance += 500
			if profile.Balance > 1000 {
				profile.Balance = 1000
			}
		}

		updateQuery := `
			UPDATE player_profiles 
			SET balance = $1, last_weekly_claim = $2 
			WHERE username = $3`
		_, _ = s.db.Exec(updateQuery, profile.Balance, profile.LastWeeklyClaim, username)
	}

	return profile
}

// UpdateBalance обновляет баланс игрока на указанное значение
func (s *DBStore) UpdateBalance(username string, delta int) {
	query := `
		UPDATE player_profiles 
		SET balance = balance + $1 
		WHERE username = $2`
	_, _ = s.db.Exec(query, delta, username)
}

// AddGameResult записывает результаты сыгранной партии
func (s *DBStore) AddGameResult(username string, isWinner bool, moneyDelta int) {
	winIncrement := 0
	if isWinner {
		winIncrement = 1
	}

	query := `
		UPDATE player_profiles 
		SET balance = balance + $1, 
		    games = games + 1, 
		    wins = wins + $2 
		WHERE username = $3`
	_, _ = s.db.Exec(query, moneyDelta, winIncrement, username)
}

// GetAllProfiles возвращает топ игроков, отсортированный по балансу
func (s *DBStore) GetAllProfiles() []*models.PlayerProfile {
	query := `
		SELECT username, balance, wins, games, avatar, last_weekly_claim 
		FROM player_profiles 
		ORDER BY balance DESC`

	rows, err := s.db.Query(query)
	if err != nil {
		return []*models.PlayerProfile{}
	}
	defer rows.Close()

	var profiles []*models.PlayerProfile
	for rows.Next() {
		p := &models.PlayerProfile{}
		if err := rows.Scan(&p.Username, &p.Balance, &p.Wins, &p.Games, &p.Avatar, &p.LastWeeklyClaim); err == nil {
			profiles = append(profiles, p)
		}
	}

	return profiles
}

// DeductBalance безопасно списывает золото при условии его достаточности
func (s *DBStore) DeductBalance(username string, amount int) error {
	query := `
		UPDATE player_profiles 
		SET balance = balance - $1 
		WHERE username = $2 AND balance >= $1`

	res, err := s.db.Exec(query, amount, username)
	if err != nil {
		return err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil || rowsAffected == 0 {
		return fmt.Errorf("недостаточно золота или профиль не найден")
	}

	return nil
}

// UpdateAvatar обновляет имя аватара пользователя
func (s *DBStore) UpdateAvatar(username string, avatar string) error {
	query := `
		UPDATE player_profiles 
		SET avatar = $1 
		WHERE username = $2`

	res, err := s.db.Exec(query, avatar, username)
	if err != nil {
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("профиль не найден")
	}

	return nil
}

func isNewWeek(lastClaim time.Time) bool {
	if lastClaim.IsZero() {
		return true
	}
	now := time.Now()
	nowYear, nowWeek := now.ISOWeek()
	lastYear, lastWeek := lastClaim.ISOWeek()

	// наступил новый год или более поздняя неделя в текущем году
	return nowYear > lastYear || (nowYear == lastYear && nowWeek > lastWeek)
}
