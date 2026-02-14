package handlers

import (
	"auth/internal/service"
	"auth/internal/storage"
	. "auth/pkg"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

// RegHandler структура-обёртка над storage.ManagerDB, реализующая ServeHTTP,
// из-за чего может использоваться в методе Handle, нужна для доступа к БД в обработчиках.
type RegHandler struct {
	mt storage.ManagerTable
}

func NewRegHandler(mt storage.ManagerTable) *RegHandler {
	return &RegHandler{mt: mt}
}

// ServeHTTP метод RegHandler, для реализации интерфейса Handle.
func (rgh *RegHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func(r *http.Request) {
		r.Body.Close()
	}(r)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	body, err := io.ReadAll(r.Body)

	if err != nil {
		msg := "Ошибка чтения тела запросы"
		http.Error(w, msg, http.StatusBadRequest)
		slog.Error(msg,
			"ошибка", err,
			"логин", "",
			"handler", "reg")
		Reply(w, "Ошибка чтения тела запросы", "", "")
		return
	}

	var user storage.User
	if err := json.Unmarshal(body, &user); err != nil {
		slog.Error("Ошибка при получении данных пользователя",
			"ошибка", err,
			"логин", user.Login,
			"handler", "reg")
		Reply(w, "Ошибка при получении данных пользователя", "", "")
		return
	}

	exist, err := rgh.mt.LoginExists(ctx, user.Login)
	if err != nil {
		slog.Error("ошибка при проверки сущ. пользователя",
			"ошибка", err,
			"логин", user.Login,
			"handler", "reg")
		Reply(w, fmt.Sprintf("ошибка: %w при проверки сущ. пользователя: %s", err, user.Login), user.Login, "")
		return
	}
	if exist {
		Reply(w, "этот логин занят", user.Login, "")
		return
	}

	if len(user.Pass) < 5 {
		Reply(w, "пароль должен быть не менее 5 символов", user.Login, "")
		return
	}

	if len(user.Login) < 3 {
		Reply(w, "логин должен быть не менее 3 символов", user.Login, "")
		return
	}

	err = rgh.mt.AddUser(ctx, user)
	if err != nil {
		slog.Error("ошибка добавления логина",
			"ошибка", err,
			"логин", user.Login,
			"handler", "auth")
		Reply(w, fmt.Sprintf("ошибка: %w добавления логина: %s", err, user.Login), user.Login, "")
		return
	}

	slog.Info("пользователь успешно зарегистрирован",
		"логин", user.Login,
		"handler", "reg")

	// создаём JWT токен
	strToken, err := service.CreateJWt(user.Login)
	if err != nil {
		slog.Error("ошибка подписания JWT токена",
			"ошибка", err,
			"логин", user.Login,
			"handler", "auth")
		Reply(w, fmt.Sprintf("ошибка подписания JWT токена: %w", err), user.Login, strToken)
		return
	}
	Reply(w, "регистрация успешна", user.Login, strToken)
	slog.Info("пользователь успешно зарегистрирован",
		"логин", user.Login,
		"handler", "reg")
}
