package economy

import (
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

// Тест на проверку корректного начисления еженедельного бонуса
func TestDBStore_GetProfile_WeeklyBonus(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать mock базы данных: %v", err)
	}
	defer db.Close()

	store := NewDBStore(db)
	username := "test_player"
	oldTime := time.Now().Add(-10 * 24 * time.Hour) // 10 дней назад (прошлая неделя)

	// 1. Ожидаем SELECT профиля (баланс 400, что меньше 1000)
	rows := sqlmock.NewRows([]string{"username", "balance", "wins", "games", "avatar", "last_weekly_claim"}).
		AddRow(username, 400, 2, 5, "fat_cat", oldTime)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT username, balance, wins, games, avatar, last_weekly_claim 
		FROM player_profiles 
		WHERE username = $1`)).
		WithArgs(username).
		WillReturnRows(rows)

	// 2. Ожидаем UPDATE с оптимистичной блокировкой (должен начислить +500)
	// Ожидаем, что rowsAffected вернет 1 (успешно обновилось)
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE player_profiles 
		SET balance = balance + $1, last_weekly_claim = $2 
		WHERE username = $3 AND last_weekly_claim = $4`)).
		WithArgs(500, sqlmock.AnyArg(), username, oldTime).
		WillReturnResult(sqlmock.NewResult(0, 1))

	profile := store.GetProfile(username)

	if profile == nil {
		t.Fatalf("ожидался профиль, получен nil")
	}

	// Баланс должен стать 400 + 500 = 900
	if profile.Balance != 900 {
		t.Errorf("ожидался баланс 900 после бонуса, получено %d", profile.Balance)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания sqlmock оправдались: %v", err)
	}
}

// Тест на защиту от гонки: если другой поток успел обновить дату раньше (rowsAffected == 0)
func TestDBStore_GetProfile_RaceConditionPrevention(t *testing.T) {
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("не удалось создать mock базы данных: %v", err)
	}
	defer db.Close()

	store := NewDBStore(db)
	username := "test_player"
	oldTime := time.Now().Add(-10 * 24 * time.Hour)

	// 1. SELECT профиля
	rows := sqlmock.NewRows([]string{"username", "balance", "wins", "games", "avatar", "last_weekly_claim"}).
		AddRow(username, 400, 2, 5, "fat_cat", oldTime)

	mock.ExpectQuery(regexp.QuoteMeta(`
		SELECT username, balance, wins, games, avatar, last_weekly_claim 
		FROM player_profiles 
		WHERE username = $1`)).
		WithArgs(username).
		WillReturnRows(rows)

	// 2. UPDATE возвращает 0 затронутых строк (другой поток уже обновил запись секунду назад)
	mock.ExpectExec(regexp.QuoteMeta(`
		UPDATE player_profiles 
		SET balance = balance + $1, last_weekly_claim = $2 
		WHERE username = $3 AND last_weekly_claim = $4`)).
		WithArgs(500, sqlmock.AnyArg(), username, oldTime).
		WillReturnResult(sqlmock.NewResult(0, 0)) // 0 затронутых строк!

	profile := store.GetProfile(username)

	if profile == nil {
		t.Fatalf("ожидался профиль, получен nil")
	}

	// Баланс НЕ должен увеличиться локально, так как обновление в БД отклонено
	if profile.Balance != 400 {
		t.Errorf("ожидался старый баланс 400 (бонус ушел другому потоку), получено %d", profile.Balance)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("не все ожидания sqlmock оправдались: %v", err)
	}
}
