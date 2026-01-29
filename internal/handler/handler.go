package handler

import (
	"net/http"
)

type Storage interface {
	UpdateGauge(name string, value float64)
	UpdateCounter(name string, value int64)
}

type Handler struct {
	storage Storage
}

func NewHandler(storage Storage) *Handler {
	return &Handler{
		storage: storage,
	}
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	// Парсим запрос и валидируем
	metric, err := ParseAndValidate(r.URL.Path)
	// Выводим ошибки валидации если есть
	if err != nil {
		http.Error(w, err.Error(), err.Status())
		return
	}
	// Обновляем данные согласно типу метрику
	switch m := metric.(type) {
	case *GaugeMetric:
		h.storage.UpdateGauge(m.Name, m.Value)
	case *CounterMetric:
		h.storage.UpdateCounter(m.Name, m.Value)
	}

	w.WriteHeader(http.StatusOK)
}
