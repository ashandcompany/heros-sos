package database_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/testutil"
)

const insertAudit = `INSERT INTO audit_logs (id, action) VALUES (?, ?)`

func countAudit(t *testing.T, db *gorm.DB) int64 {
	t.Helper()
	var count int64
	if err := db.Table("audit_logs").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	return count
}

func TestMigrationsApplyRollbackAndReapply(t *testing.T) {
	url := testutil.DatabaseURL(t)

	if err := database.Migrate(url); err != nil {
		t.Fatalf("Migrate : %v", err)
	}
	if err := database.Migrate(url); err != nil {
		t.Fatalf("Migrate rejoué (doit être sans effet) : %v", err)
	}
	if err := database.Rollback(url); err != nil {
		t.Fatalf("Rollback : %v", err)
	}
	if err := database.Migrate(url); err != nil {
		t.Fatalf("Migrate après Rollback : %v", err)
	}
}

func TestTransactorCommitsOnSuccess(t *testing.T) {
	db := testutil.NewDB(t)
	transactor := database.NewTransactor(db)

	err := transactor.WithinTx(context.Background(), func(ctx context.Context) error {
		return database.Conn(ctx, db).Exec(insertAudit, uuid.New(), "test.commit").Error
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := countAudit(t, db); got != 1 {
		t.Fatalf("lignes = %d, attendu 1", got)
	}
}

func TestTransactorRollsBackOnError(t *testing.T) {
	db := testutil.NewDB(t)
	transactor := database.NewTransactor(db)
	boom := errors.New("échec métier")

	err := transactor.WithinTx(context.Background(), func(ctx context.Context) error {
		if err := database.Conn(ctx, db).Exec(insertAudit, uuid.New(), "test.rollback").Error; err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("erreur = %v, attendu l'erreur métier", err)
	}
	if got := countAudit(t, db); got != 0 {
		t.Fatalf("lignes = %d, la transaction aurait dû être annulée", got)
	}
}

func TestTransactorRollsBackOnPanic(t *testing.T) {
	db := testutil.NewDB(t)
	transactor := database.NewTransactor(db)

	func() {
		defer func() { _ = recover() }()
		_ = transactor.WithinTx(context.Background(), func(ctx context.Context) error {
			_ = database.Conn(ctx, db).Exec(insertAudit, uuid.New(), "test.panic").Error
			panic("catastrophe")
		})
	}()

	if got := countAudit(t, db); got != 0 {
		t.Fatalf("lignes = %d, la transaction aurait dû être annulée", got)
	}
}

func TestTransactorJoinsOuterTransaction(t *testing.T) {
	db := testutil.NewDB(t)
	transactor := database.NewTransactor(db)
	boom := errors.New("annule tout")

	err := transactor.WithinTx(context.Background(), func(ctx context.Context) error {
		inner := transactor.WithinTx(ctx, func(ctx context.Context) error {
			return database.Conn(ctx, db).Exec(insertAudit, uuid.New(), "test.nested").Error
		})
		if inner != nil {
			return inner
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("erreur = %v", err)
	}
	if got := countAudit(t, db); got != 0 {
		t.Fatalf("la transaction imbriquée doit être annulée avec l'externe (lignes = %d)", got)
	}
}

func TestIsUniqueViolation(t *testing.T) {
	db := testutil.NewDB(t)
	id := uuid.New()

	if err := db.Exec(insertAudit, id, "a").Error; err != nil {
		t.Fatal(err)
	}
	err := db.Exec(insertAudit, id, "b").Error
	if !database.IsUniqueViolation(err, "") || !database.IsUniqueViolation(err, "audit_logs_pkey") {
		t.Errorf("la violation de clé primaire doit être reconnue : %v", err)
	}
	if database.IsUniqueViolation(err, "autre_contrainte") {
		t.Error("une autre contrainte ne doit pas correspondre")
	}
	if database.IsUniqueViolation(errors.New("autre"), "") || database.IsUniqueViolation(nil, "") {
		t.Error("une erreur étrangère n'est pas une violation d'unicité")
	}
}

func TestConnIsSafeForConcurrentTransactions(t *testing.T) {
	db := testutil.NewDB(t)
	transactor := database.NewTransactor(db)

	const workers = 16
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- transactor.WithinTx(context.Background(), func(ctx context.Context) error {
				return database.Conn(ctx, db).Exec(insertAudit, uuid.New(), "test.concurrent").Error
			})
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Error(err)
		}
	}
	if got := countAudit(t, db); got != workers {
		t.Fatalf("lignes = %d, attendu %d", got, workers)
	}
}
