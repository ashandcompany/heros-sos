package server

import (
	"context"
	"net/http"
	"time"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

type healthResponse struct {
	Status string `json:"status"`
}

const (
	statusOK          = "ok"
	statusUnavailable = "unavailable"
)

// healthHandler expose les sondes de santé : live (le processus répond) et ready (la base
// répond), utilisées par Docker et l'orchestrateur.
type healthHandler struct {
	db          *gorm.DB
	pingTimeout time.Duration
}

func (h healthHandler) live(w http.ResponseWriter, _ *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, healthResponse{Status: statusOK})
}

func (h healthHandler) ready(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), h.pingTimeout)
	defer cancel()
	if err := database.Ping(ctx, h.db); err != nil {
		httpx.WriteJSON(w, http.StatusServiceUnavailable, healthResponse{Status: statusUnavailable})
		return
	}
	httpx.WriteJSON(w, http.StatusOK, healthResponse{Status: statusOK})
}
