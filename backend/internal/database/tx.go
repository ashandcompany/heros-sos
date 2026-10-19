package database

import (
	"context"

	"gorm.io/gorm"
)

type txKey struct{}

// Conn retourne la connexion à utiliser pour ctx : la transaction en cours s'il y en a une,
// sinon le pool. Tous les dépôts passent par elle, ce qui leur permet de participer à une
// transaction ouverte par un service sans en connaître le détail.
func Conn(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}

// Transactor exécute une fonction dans une transaction. Les services en dépendent par
// interface pour pouvoir être testés sans base de données.
type Transactor interface {
	// WithinTx valide la transaction si fn réussit, l'annule si fn échoue ou panique.
	// Appelée dans une transaction déjà ouverte, fn y participe sans en créer une autre.
	WithinTx(ctx context.Context, fn func(ctx context.Context) error) error
}

// GormTransactor est le Transactor adossé à GORM.
type GormTransactor struct {
	db *gorm.DB
}

// NewTransactor crée un Transactor pour la base donnée.
func NewTransactor(db *gorm.DB) *GormTransactor {
	return &GormTransactor{db: db}
}

// WithinTx implémente Transactor.
func (t *GormTransactor) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(*gorm.DB); ok {
		return fn(ctx)
	}
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}
