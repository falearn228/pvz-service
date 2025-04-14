package db

import (
	"fmt"
	"log"

	"pvz-service/internal/config"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

// Database представляет соединение с базой данных
type Database struct {
	*sqlx.DB
}

// NewDatabase создает новое соединение с базой данных
func NewDatabase(config *config.DatabaseConfig) (*Database, error) {
	// Формируем строку подключения
	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)

	// Устанавливаем соединение
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Проверяем соединение
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to database")

	return &Database{db}, nil
}
