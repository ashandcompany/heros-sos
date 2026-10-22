// Commande replay : rejoue le journal d'événements pour vérifier qu'il raconte la même histoire que la base,
// ou recalcule les compteurs de lecture.
//
//	go run ./cmd/replay                     rejoue le journal et compare le statut de chaque incident à la base
//	go run ./cmd/replay -rebuild-counters   recalcule les compteurs de lecture depuis les tables
//
// Le code de sortie est 1 si un incident diffère de son journal.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ashandcompany/heros-sos/backend/internal/clock"
	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/events"
)

const maxReportedMismatches = 20

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	rebuild := flag.Bool("rebuild-counters", false, "recalcule les compteurs de lecture depuis les tables")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}
	db, err := database.Open(ctx, cfg.Database, slog.New(slog.DiscardHandler))
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	if *rebuild {
		if err := events.RebuildCounters(ctx, db); err != nil {
			return err
		}
		fmt.Println("Compteurs recalculés depuis les tables.")
		return nil
	}

	checked, mismatches, err := events.Verify(ctx, db, events.NewStore(db, clock.System{}), cfg.Events.BatchSize)
	if err != nil {
		return err
	}
	fmt.Printf("%d incidents rejoués depuis le journal.\n", checked)
	if len(mismatches) == 0 {
		fmt.Println("Le journal et la base racontent la même histoire.")
		return nil
	}
	for i, mismatch := range mismatches {
		if i == maxReportedMismatches {
			fmt.Printf("… et %d autres.\n", len(mismatches)-i)
			break
		}
		fmt.Printf("  incident %s : le journal dit %q, la base dit %q\n", mismatch.IncidentID, mismatch.Journal, mismatch.State)
	}
	return fmt.Errorf("%d incidents diffèrent de leur journal", len(mismatches))
}
