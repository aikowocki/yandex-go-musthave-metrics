package handler

import (
	"context"
	"net/http"
)

// Pinger — интерфейс для проверки доступности внешнего ресурса (например, БД).
type Pinger interface {
	Ping(ctx context.Context) error
}

// HealthHandler обрабатывает запросы проверки здоровья сервиса (ping).
type HealthHandler struct {
	pinger Pinger
}

// NewHealthHandler создаёт обработчик health-check с указанным pinger (обычно — подключение к БД).
func NewHealthHandler(pinger Pinger) *HealthHandler {
	return &HealthHandler{pinger: pinger}
}

func (h *HealthHandler) Ping(w http.ResponseWriter, r *http.Request) {
	if h.pinger == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if err := h.pinger.Ping(r.Context()); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
