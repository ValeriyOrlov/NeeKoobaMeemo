package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// InitDB подключается к PostgreSQL, настраивает пул и инициализирует схему
func InitDB(connStr string) (*sql.DB, error) {
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("ошибка открытия БД: %w", err)
	}

	// Настройки пула соединений
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Проверяем реальную доступность СУБД
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("база данных недоступна: %w", err)
	}

	// Автоматическая инициализация таблиц при успешном подключении
	if err := createSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("ошибка создания таблиц: %w", err)
	}

	return db, nil
}

// createSchema выполняет миграции "на лету" при запуске сервера
func createSchema(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS player_profiles (
		id SERIAL PRIMARY KEY,
		username VARCHAR(50) UNIQUE NOT NULL,
		avatar VARCHAR(50) NOT NULL DEFAULT 'monk',
		balance INT NOT NULL DEFAULT 500,
		wins INT NOT NULL DEFAULT 0,
		games INT NOT NULL DEFAULT 0,
		last_weekly_claim TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err := db.Exec(query)
	return err
}
