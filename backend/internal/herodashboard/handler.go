package herodashboard

import (
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
	"github.com/ashandcompany/heros-sos/backend/internal/incident"
)

// Handler expose le tableau de bord des héros.
type Handler struct {
	service *Service
	logger  *slog.Logger
}

// NewHandler crée le handler.
func NewHandler(service *Service, logger *slog.Logger) *Handler {
	return &Handler{service: service, logger: logger}
}

type typeResponse struct {
	Code  string `json:"code"`
	Label string `json:"label"`
}

type typeCountResponse struct {
	Type  typeResponse `json:"type"`
	Count int64        `json:"count"`
}

type globalResponse struct {
	Interventions      int64               `json:"interventions"`
	IncidentsByType    []typeCountResponse `json:"incidentsByType"`
	RescuedCitizens    int64               `json:"rescuedCitizens"`
	VillainsIdentified int64               `json:"villainsIdentified"`
	VillainsImprisoned int64               `json:"villainsImprisoned"`
}

type enemyResponse struct {
	ID         uuid.UUID `json:"id"`
	Alias      string    `json:"alias"`
	Encounters int64     `json:"encounters"`
}

type personalResponse struct {
	Interventions           int64           `json:"interventions"`
	SuccessfulInterventions int64           `json:"successfulInterventions"`
	FailedInterventions     int64           `json:"failedInterventions"`
	AverageRating           float64         `json:"averageRating"`
	RatingsCount            int64           `json:"ratingsCount"`
	Videos                  int64           `json:"videos"`
	PublishedVideos         int64           `json:"publishedVideos"`
	MainEnemies             []enemyResponse `json:"mainEnemies"`
}

// Response est le corps de GET /hero/dashboard (type HeroDashboard du frontend).
type Response struct {
	Global             globalResponse      `json:"global"`
	Personal           personalResponse    `json:"personal"`
	MyOngoingIncidents []incident.Response `json:"myOngoingIncidents"`
	RecentIncidents    []incident.Response `json:"recentIncidents"`
}

func newResponse(d Dashboard) Response {
	byType := make([]typeCountResponse, len(d.Global.IncidentsByType))
	for i, item := range d.Global.IncidentsByType {
		byType[i] = typeCountResponse{Type: typeResponse{Code: item.Code, Label: item.Label}, Count: item.Count}
	}
	enemies := make([]enemyResponse, len(d.Personal.MainEnemies))
	for i, enemy := range d.Personal.MainEnemies {
		enemies[i] = enemyResponse(enemy)
	}
	return Response{
		Global: globalResponse{
			Interventions: d.Global.Interventions, IncidentsByType: byType, RescuedCitizens: d.Global.RescuedCitizens,
			VillainsIdentified: d.Global.VillainsIdentified, VillainsImprisoned: d.Global.VillainsImprisoned,
		},
		Personal: personalResponse{
			Interventions: d.Personal.Interventions, SuccessfulInterventions: d.Personal.SuccessfulInterventions,
			FailedInterventions: d.Personal.FailedInterventions, AverageRating: d.Personal.AverageRating,
			RatingsCount: d.Personal.RatingsCount, Videos: d.Personal.Videos,
			PublishedVideos: d.Personal.PublishedVideos, MainEnemies: enemies,
		},
		MyOngoingIncidents: incidentResponses(d.MyOngoingIncidents),
		RecentIncidents:    incidentResponses(d.RecentIncidents),
	}
}

func incidentResponses(items []incident.Incident) []incident.Response {
	responses := make([]incident.Response, len(items))
	for i, item := range items {
		responses[i] = incident.NewResponse(item)
	}
	return responses
}

// Get traite GET /hero/dashboard.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	principal, _ := auth.PrincipalFrom(r.Context())
	dashboard, err := h.service.Get(r.Context(), principal.UserID)
	if err != nil {
		httpx.Fail(w, r, h.logger, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, newResponse(dashboard))
}
