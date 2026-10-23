package moderation

import (
	"context"
	"fmt"
	"net/http"

	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// Summary compte les éléments en attente de chaque section du back-office.
type Summary struct {
	Incidents        int64 `json:"incidents" gorm:"column:incidents"`
	HeroApplications int64 `json:"heroApplications" gorm:"column:hero_applications"`
	Videos           int64 `json:"videos" gorm:"column:videos"`
	Comments         int64 `json:"comments" gorm:"column:comments"`
	EmailChanges     int64 `json:"emailChanges" gorm:"column:email_changes"`
	DraftArticles    int64 `json:"draftArticles" gorm:"column:draft_articles"`
	DraftNewsletters int64 `json:"draftNewsletters" gorm:"column:draft_newsletters"`
}

// SummaryStore calcule le résumé en une seule requête.
type SummaryStore struct {
	db *gorm.DB
}

// NewSummaryStore crée le dépôt du résumé.
func NewSummaryStore(db *gorm.DB) *SummaryStore { return &SummaryStore{db: db} }

const summarySQL = `SELECT
	(SELECT COUNT(*) FROM incidents WHERE status = 'reported') AS incidents,
	(SELECT COUNT(*) FROM hero_profiles WHERE verification_status = 'pending') AS hero_applications,
	(SELECT COUNT(*) FROM hero_films WHERE status = 'pending') AS videos,
	(SELECT COUNT(*) FROM hero_review_replies WHERE status = 'pending') AS comments,
	(SELECT COUNT(*) FROM email_change_requests WHERE status = 'pending') AS email_changes,
	(SELECT COUNT(*) FROM articles WHERE status = 'brouillon') AS draft_articles,
	(SELECT COUNT(*) FROM newsletters WHERE sent_at IS NULL) AS draft_newsletters`

// Get retourne les compteurs courants.
func (s *SummaryStore) Get(ctx context.Context) (Summary, error) {
	var summary Summary
	if err := database.Conn(ctx, s.db).Raw(summarySQL).Scan(&summary).Error; err != nil {
		return Summary{}, fmt.Errorf("moderation : résumé : %w", err)
	}
	return summary, nil
}

// Summary traite GET /admin/moderation/summary.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.summary.Get(r.Context())
	if err != nil {
		h.fail(w, r, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, summary)
}
