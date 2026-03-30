package handler

import (
	"context"
	"database/sql"
	"net/http"
)

type Pinger interface {
	Ping(ctx context.Context) error
}

type DBPinger struct {
	db *sql.DB
}

func NewDBPinger(db *sql.DB) *DBPinger {
	return &DBPinger{db: db}
}

func (p *DBPinger) Ping(ctx context.Context) error {
	return p.db.PingContext(ctx)
}

type Healthcheck struct {
	pinger Pinger
}

func NewHealthcheck(p Pinger) *Healthcheck {
	return &Healthcheck{pinger: p}
}

func (h *Healthcheck) Ping(w http.ResponseWriter, r *http.Request) {
	if h.pinger == nil {
		http.Error(w, "Database not configured", http.StatusInternalServerError)
		return
	}

	if err := h.pinger.Ping(r.Context()); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
