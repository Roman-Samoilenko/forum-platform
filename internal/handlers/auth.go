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

// AuthHandler структура-обёртка над storage.ManagerDB, реализующая ServeHTTP,
// из-за чего может использоваться в методе Handle, нужна для доступа к БД в обработчиках.
type AuthHandler struct {
	mt storage.ManagerTable
}

func NewAuthHandler(mt storage.ManagerTable) *AuthHandler {
	return &AuthHandler{mt: mt}
}

// при получении логина и пароля, если логина нет в БД, то говорим что такого сочетания нет и предлагаем зарегистрироваться,
// если логин есть, то проверяем пароль, если верный, то возвращаем JWT токен, иначе ошибку.

// ServeHTTP метод AuthHandler, для реализации интерфейса Handle.
func (a *AuthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
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
			"handler", "auth")
		Reply(w, "Ошибка чтения тела запросы", "", "")
		return
	}
	var user storage.User
	if err := json.Unmarshal(body, &user); err != nil {
		slog.Error("Ошибка при получении данных пользователя",
			"ошибка", err,
			"логин", user.Login,
			"handler", "auth")
		Reply(w, "Ошибка при получении данных пользователя", "", "")
		return
	}

	exist, err := a.mt.LoginExists(ctx, user.Login)
	if err != nil {
		slog.Error("ошибка при проверки сущ. пользователя",
			"ошибка", err,
			"логин", user.Login,
			"handler", "auth")
		Reply(w, fmt.Sprintf("ошибка: %w при проверки сущ. пользователя: %s", err, user.Login), user.Login, "")
		return
	}
	if !exist {
		Reply(w, "такого сочетания пароля и логина нет", user.Login, "")
		return
	} else {
		correct, err := a.mt.CheckPassHash(ctx, user.Pass, user.Login)
		if err != nil {
			slog.Error("ошибка проверка пароля",
				"ошибка", err,
				"логин", user.Login,
				"handler", "auth")
			Reply(w, fmt.Sprintf("ошибка проверка пароля: %w", err), user.Login, "")
			return
		}
		if !correct {
			Reply(w, "неправильный пароль для этого логина", user.Login, "")
			return
		}
		strToken, err := service.CreateJWt(user.Login)
		if err != nil {
			slog.Error("ошибка подписания JWT токена",
				"ошибка", err,
				"логин", user.Login,
				"handler", "auth")
			Reply(w, fmt.Sprintf("ошибка подписания JWT токена: %w", err), user.Login, strToken)
			return
		}
		Reply(w, "вход успешен", user.Login, strToken)
	}
	slog.Info("пользователь успешно авторизован",
		"логин", user.Login,
		"handler", "auth")
}
