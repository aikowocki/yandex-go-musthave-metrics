package handler

import (
	"database/sql"
	"net/http"
)

type HealthcheckHandler struct {
	db *sql.DB
}

func NewHealthcheckHandler(db *sql.DB) *HealthcheckHandler {
	return &HealthcheckHandler{db: db}
}

func (h *HealthcheckHandler) Ping(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	if err != nil {
		http.Error(w, "db down", http.StatusInternalServerError)
	}
}
