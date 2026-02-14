package pkg

import (
	"encoding/json"
	"net/http"
)

// Reply формирует и отправляет JSON-ответ клиенту, включая сообщение, логин и токен.
// Также устанавливает HTTP-only cookie с именем "auth_token".
func Reply(w http.ResponseWriter, msg string, login string, strToken string) {
	resp := map[string]string{
		"message": msg,
		"login":   login,
		"token":   strToken,
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    strToken,
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		MaxAge:   3600,
	})

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		http.Error(w, "Ошибка создания ответа", http.StatusInternalServerError)
	}
}
