package handler

import (
	"database/sql"
	"net/http"
)

type Healthcheck struct {
	db *sql.DB
}

func NewHealthcheck(db *sql.DB) *Healthcheck {
	return &Healthcheck{db: db}
}

func (h *Healthcheck) Ping(w http.ResponseWriter, r *http.Request) {
	if h.db == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	err := h.db.Ping()
	if err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
