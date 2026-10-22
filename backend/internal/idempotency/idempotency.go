// Package idempotency fait d'une requête d'écriture une commande qui ne s'applique qu'une fois : le client joint
// une clé (en-tête Idempotency-Key) ; si la même clé revient, la réponse du premier passage est rendue telle
// quelle, sans rien refaire. C'est ce qui permet de rejouer sans risque une action faite hors ligne, quand la
// connexion a pu couper pendant l'envoi sans qu'on sache si le serveur l'a reçue.
//
// La clé est facultative : sans elle, la requête s'exécute comme avant. Elle est propre à son auteur, et la même
// clé avec une autre requête (autre chemin, autre corps) est refusée.
package idempotency

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/auth"
	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
	"github.com/ashandcompany/heros-sos/backend/internal/httpx"
)

// En-têtes de la requête et de la réponse.
const (
	KeyHeader             = "Idempotency-Key"
	ClientCreatedAtHeader = "X-Client-Created-At"
	// ReplayedHeader vaut « true » sur une réponse rejouée depuis le premier passage.
	ReplayedHeader = "Idempotent-Replayed"
)

// Codes d'erreur de l'API.
const (
	CodeInvalidKey = "invalid_idempotency_key"
	CodeKeyReused  = "idempotency_key_reused"
)

// futureTolerance absorbe le décalage d'horloge entre l'appareil et le serveur : une date du client plus loin
// dans le futur que cela est ramenée à l'instant de réception.
const futureTolerance = time.Minute

// errRejected annule la transaction d'une requête qui n'a pas réussi : ni l'état, ni la clé ne sont conservés,
// et la requête pourra se rejouer.
var errRejected = errors.New("idempotency : requête rejetée")

// Deps regroupe les dépendances du middleware.
type Deps struct {
	DB     *gorm.DB
	Tx     database.Transactor
	Clock  clock.Clock
	Logger *slog.Logger
	// MaxBodyBytes borne le corps lu pour calculer l'empreinte : au-delà, la requête passe sans clé (le
	// handler répondra 413).
	MaxBodyBytes int64
}

// Middleware rend les routes d'écriture idempotentes.
type Middleware struct {
	Deps
}

// New crée le middleware.
func New(deps Deps) *Middleware { return &Middleware{Deps: deps} }

// stored est la réponse conservée d'une commande déjà traitée.
type stored struct {
	RequestHash string
	StatusCode  int
	ContentType string
	Response    []byte
}

// Wrap protège un handler. Il doit être placé APRÈS l'authentification (la clé est celle d'un utilisateur) :
// c.protected(idempotency.Wrap(handler)).
//
// Sans en-tête Idempotency-Key, le handler s'exécute tel quel. Avec une clé, tout se passe dans une transaction :
// la clé est réservée d'abord (un doublon simultané attend la fin du premier), le handler s'exécute dans cette
// transaction, et la réponse n'est conservée que si elle réussit. Un échec annule l'état ET la clé : le client
// rejoue sa commande avec la même clé sans rien craindre.
func (m *Middleware) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawKey := r.Header.Get(KeyHeader)
		if rawKey == "" {
			next(w, r)
			return
		}
		key, err := uuid.Parse(rawKey)
		if err != nil || key == uuid.Nil {
			httpx.WriteError(w, httpx.BadRequest("L'en-tête "+KeyHeader+" doit être un UUID."))
			return
		}
		clientCreatedAt, err := m.clientCreatedAt(r)
		if err != nil {
			httpx.WriteError(w, httpx.BadRequest("L'en-tête "+ClientCreatedAtHeader+" doit être une date RFC 3339."))
			return
		}
		principal, ok := auth.PrincipalFrom(r.Context())
		if !ok {
			next(w, r)
			return
		}
		body, oversize, err := m.readBody(r)
		if err != nil {
			httpx.WriteError(w, httpx.BadRequest("Corps de requête illisible."))
			return
		}
		if oversize {
			next(w, r)
			return
		}
		m.run(w, r, next, principal.UserID, events.Command{ID: key, ClientCreatedAt: clientCreatedAt}, hashRequest(r, body))
	}
}

func (m *Middleware) clientCreatedAt(r *http.Request) (*time.Time, error) {
	raw := r.Header.Get(ClientCreatedAtHeader)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return nil, err
	}
	if now := m.Clock.Now(); parsed.After(now.Add(futureTolerance)) {
		parsed = now
	}
	return &parsed, nil
}

// readBody lit le corps pour l'empreinte puis le remet à disposition du handler. oversize indique qu'il dépasse la
// limite : il est alors rendu intact (lu en partie + reste) et l'idempotence ne s'applique pas.
func (m *Middleware) readBody(r *http.Request) (body []byte, oversize bool, err error) {
	body, err = io.ReadAll(io.LimitReader(r.Body, m.MaxBodyBytes+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(body)) > m.MaxBodyBytes {
		r.Body = readCloser{Reader: io.MultiReader(bytes.NewReader(body), r.Body), Closer: r.Body}
		return nil, true, nil
	}
	r.Body = io.NopCloser(bytes.NewReader(body))
	return body, false, nil
}

type readCloser struct {
	io.Reader
	io.Closer
}

func hashRequest(r *http.Request, body []byte) string {
	sum := sha256.New()
	sum.Write([]byte(r.Method + "\n" + r.URL.Path + "\n"))
	sum.Write(body)
	return hex.EncodeToString(sum.Sum(nil))
}

func (m *Middleware) run(w http.ResponseWriter, r *http.Request, next http.HandlerFunc, userID uuid.UUID, command events.Command, hash string) {
	recorder := newRecorder()
	var replay *stored
	err := m.Tx.WithinTx(r.Context(), func(ctx context.Context) error {
		claimed, previous, err := m.claim(ctx, userID, command.ID, r, hash)
		if err != nil {
			return err
		}
		if !claimed {
			replay = previous
			return nil
		}
		next(recorder, r.WithContext(events.WithCommand(ctx, command)))
		if recorder.status < http.StatusOK || recorder.status >= http.StatusMultipleChoices {
			return errRejected
		}
		return m.save(ctx, userID, command.ID, recorder)
	})

	switch {
	case replay != nil:
		m.replay(w, replay, hash)
	case err == nil || errors.Is(err, errRejected):
		// Réussite validée, ou échec annulé : dans les deux cas, la réponse du handler part telle quelle.
		recorder.flush(w)
	default:
		m.Logger.ErrorContext(r.Context(), "idempotence : commande en échec", "command_id", command.ID, "error", err)
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, httpx.CodeInternal, "Une erreur interne est survenue."))
	}
}

// claim réserve la clé. Une clé déjà réservée par une transaction en cours fait attendre celle-ci jusqu'à sa fin
// (index unique) : le doublon n'exécute donc jamais l'action en même temps que l'original.
func (m *Middleware) claim(ctx context.Context, userID, commandID uuid.UUID, r *http.Request, hash string) (bool, *stored, error) {
	conn := database.Conn(ctx, m.DB)
	result := conn.Exec(`INSERT INTO processed_commands (user_id, command_id, method, path, request_hash, created_at)
		VALUES (?, ?, ?, ?, ?, ?) ON CONFLICT (user_id, command_id) DO NOTHING`,
		userID, commandID, r.Method, r.URL.Path, hash, m.Clock.Now())
	if result.Error != nil {
		return false, nil, fmt.Errorf("idempotency : réservation de la clé : %w", result.Error)
	}
	if result.RowsAffected == 1 {
		return true, nil, nil
	}
	var previous stored
	err := conn.Raw(`SELECT request_hash, status_code, content_type, response FROM processed_commands
		WHERE user_id = ? AND command_id = ?`, userID, commandID).Row().Scan(
		&previous.RequestHash, &previous.StatusCode, &previous.ContentType, &previous.Response)
	if err != nil {
		return false, nil, fmt.Errorf("idempotency : lecture de la commande : %w", err)
	}
	return false, &previous, nil
}

func (m *Middleware) save(ctx context.Context, userID, commandID uuid.UUID, recorder *recorder) error {
	// Une réponse sans contenu (204) est un tableau vide, pas NULL : la colonne est NOT NULL.
	response := append([]byte{}, recorder.body.Bytes()...)
	err := database.Conn(ctx, m.DB).Exec(`UPDATE processed_commands SET status_code = ?, content_type = ?, response = ?
		WHERE user_id = ? AND command_id = ?`,
		recorder.status, recorder.header.Get("Content-Type"), response, userID, commandID).Error
	if err != nil {
		return fmt.Errorf("idempotency : conservation de la réponse : %w", err)
	}
	return nil
}

// replay rend la réponse du premier passage, ou refuse une clé réutilisée pour une autre requête.
func (m *Middleware) replay(w http.ResponseWriter, previous *stored, hash string) {
	if previous.RequestHash != hash {
		httpx.WriteError(w, httpx.NewError(http.StatusUnprocessableEntity, CodeKeyReused,
			"Cette clé d'idempotence a déjà servi pour une autre requête."))
		return
	}
	w.Header().Set(ReplayedHeader, "true")
	if previous.ContentType != "" {
		w.Header().Set("Content-Type", previous.ContentType)
	}
	w.WriteHeader(previous.StatusCode)
	_, _ = w.Write(previous.Response)
}

// PurgeOlderThan supprime les commandes de plus de retention : passé ce délai, un rejeu n'est plus attendu.
func (m *Middleware) PurgeOlderThan(ctx context.Context, retention time.Duration) (int64, error) {
	result := database.Conn(ctx, m.DB).Exec(`DELETE FROM processed_commands WHERE created_at < ?`, m.Clock.Now().Add(-retention))
	if result.Error != nil {
		return 0, fmt.Errorf("idempotency : purge : %w", result.Error)
	}
	return result.RowsAffected, nil
}

// recorder met en mémoire la réponse du handler : elle n'est envoyée qu'une fois la transaction terminée.
type recorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newRecorder() *recorder { return &recorder{header: make(http.Header), status: http.StatusOK} }

func (r *recorder) Header() http.Header { return r.header }

func (r *recorder) WriteHeader(status int) { r.status = status }

func (r *recorder) Write(data []byte) (int, error) { return r.body.Write(data) }

func (r *recorder) flush(w http.ResponseWriter) {
	for name, values := range r.header {
		for _, value := range values {
			w.Header().Add(name, value)
		}
	}
	w.WriteHeader(r.status)
	_, _ = w.Write(r.body.Bytes())
}
