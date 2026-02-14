package handlers

import (
	"net/http"
)

// Health отвечает на запросы проверки состояния сервиса.
func Health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
