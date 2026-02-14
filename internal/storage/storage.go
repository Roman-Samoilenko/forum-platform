package storage

import (
	"context"
	"database/sql"
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// managerStorage интерфейс для установки и закрытия соединения с БД.
type managerStorage interface {
	Init() error
	Close() error
}

// ManagerTable интерфейс работы с таблицей пользователей.
type ManagerTable interface {
	AddUser(ctx context.Context, user User) error
	CheckPassHash(ctx context.Context, pass string, login string) (bool, error)
	LoginExists(ctx context.Context, login string) (bool, error)
}

// ManagerDB представляет основной интерфейс для слоя данных.
// Объединяет функциональность managerStorage и managerTable.
// Используется в API для абстракции от конкретной реализации БД, обеспечивая независимость
// бизнес-логики от деталей хранения данных и позволяя легко менять тип БД.
type ManagerDB interface {
	managerStorage
	ManagerTable
}

// BaseStorage структура, которая встраивается в конкретные реализации ManagerDB,
// для их доступа к общим методам.
type BaseStorage struct {
	db *sql.DB
}

// Close закрывает соединение с БД.
func (s *BaseStorage) Close() error {
	err := s.db.Close()
	if err != nil {
		return err
	}
	return nil
}

// AddUser метод добавления пользователя в БД.
func (s *BaseStorage) AddUser(ctx context.Context, user User) error {
	resCh := make(chan error)
	defer close(resCh)

	SQLadd := "INSERT INTO users (login, pass_hash) VALUES ($1, $2)"
	passHash, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	go func() {
		_, err := s.db.ExecContext(ctx, SQLadd, user.Login, passHash)
		resCh <- err
	}()

	select {
	case err := <-resCh:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// CheckPassHash метод проверки пароля.
func (s *BaseStorage) CheckPassHash(ctx context.Context, pass string, login string) (bool, error) {
	var pasHashFromTable []byte
	err := s.db.QueryRowContext(ctx, "SELECT pass_hash FROM users WHERE login = $1", login).Scan(&pasHashFromTable)
	if err != nil {
		return false, err
	}

	err = bcrypt.CompareHashAndPassword(pasHashFromTable, []byte(pass))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// LoginExists проверяет, есть ли логин в БД.
func (s *BaseStorage) LoginExists(ctx context.Context, login string) (bool, error) {
	resChan := make(chan struct {
		exists bool
		err    error
	})
	defer close(resChan)

	go func() {
		var exists bool
		err := s.db.QueryRowContext(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE login = $1)", login).Scan(&exists)
		resChan <- struct {
			exists bool
			err    error
		}{exists, err}
	}()

	select {
	case res := <-resChan:
		return res.exists, res.err
	case <-ctx.Done():
		return false, ctx.Err()
	}
}
