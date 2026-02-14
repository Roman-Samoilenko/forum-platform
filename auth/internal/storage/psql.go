package storage

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Psql реализация managerTable с БД PostgreSQL.
type Psql struct {
	BaseStorage
}

// NewPsql конструктор для Psql.
func NewPsql() *Psql {
	return &Psql{
		BaseStorage: BaseStorage{},
	}
}

// Init открывает соединение с БД и создаёт таблицу.
func (p *Psql) Init() error {
	pass := os.Getenv("PSQL_PASS")
	host := os.Getenv("PSQL_HOST")
	port := os.Getenv("PSQL_PORT")
	user := os.Getenv("PSQL_USER")
	dbname := os.Getenv("PSQL_NAME")

	reqInfo := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, pass, dbname,
	)

	var db *sql.DB
	var err error
	db, err = sql.Open("postgres", reqInfo)
	if err != nil {
		for range 11 {
			fmt.Println("Попытка подключения к БД...")
			time.Sleep(4 * time.Second)
			db, err = sql.Open("postgres", reqInfo)
		}
		if err != nil {
			return fmt.Errorf("ошибка при открытии БД: %w", err)
		}
	}

	if err = db.Ping(); err != nil {
		return fmt.Errorf("БД не пингуется %w", err)
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		login TEXT NOT NULL UNIQUE,
		pass_hash TEXT NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("ошибка при создании таблицы: %w, SQL запрос: %s", err, createTableSQL)
	}

	p.db = db

	return nil
}
