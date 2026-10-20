package film

import "errors"

var (
	ErrNotFound      = errors.New("film : ressource introuvable")
	ErrAlreadyLinked = errors.New("film : association déjà présente")
	ErrAlreadyReview = errors.New("film : association déjà modérée")
	ErrIMDBExists    = errors.New("film : identifiant IMDb déjà enregistré")
	ErrNotAllowed    = errors.New("film : action non autorisée")
)
