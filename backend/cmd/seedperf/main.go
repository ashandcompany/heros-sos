// Commande seedperf : ajoute des incidents en masse (5 millions par défaut) à une base déjà remplie
// par cmd/seed, pour mesurer les performances à grand volume.
//
//	go run ./cmd/seedperf                      5 000 000 d'incidents
//	go run ./cmd/seedperf -incidents 500000    un essai plus petit
//	go run ./cmd/seedperf -max-size-gb 8       s'arrête (entre deux lots) quand la base dépasse 8 Go
//
// Les incidents sont rattachés aux citoyens, héros et vilains existants : lancer `make seed` d'abord.
// La commande refuse de tourner en production. Elle ne supprime rien, mais elle ajoute plusieurs Go de
// données : à utiliser sur une base dédiée (`make seed-perf`, base herossos_perf), pas sur celle du
// développement quotidien.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ashandcompany/heros-sos/backend/internal/config"
	"github.com/ashandcompany/heros-sos/backend/internal/database"
	"github.com/ashandcompany/heros-sos/backend/internal/perfseed"
)

const bytesPerGB = 1 << 30

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx); err != nil {
		fmt.Fprintln(os.Stderr, "erreur :", err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	incidents := flag.Int64("incidents", perfseed.DefaultIncidents, "nombre d'incidents à créer")
	batch := flag.Int64("batch", perfseed.DefaultBatchSize, "taille d'un lot (une transaction)")
	appendMode := flag.Bool("append", false, "ajoute même si la base contient déjà beaucoup d'incidents")
	maxSizeGB := flag.Float64("max-size-gb", 0, "arrête la génération quand la base dépasse cette taille (0 : sans limite)")
	flag.Parse()

	cfg, err := config.FromEnv()
	if err != nil {
		return fmt.Errorf("configuration invalide :\n%w", err)
	}
	if cfg.IsProduction() {
		return errors.New("seedperf refusé en production : il ajoute des millions d'incidents fictifs")
	}

	db, err := database.Open(ctx, cfg.Database, slog.New(slog.DiscardHandler))
	if err != nil {
		return err
	}
	defer func() { _ = database.Close(db) }()

	fmt.Printf("Création de %d incidents (lots de %d)…\n", *incidents, *batch)
	began := time.Now()
	err = perfseed.Load(ctx, db, perfseed.Options{
		Incidents: *incidents, BatchSize: *batch, Append: *appendMode,
		MaxDatabaseBytes: int64(*maxSizeGB * bytesPerGB),
		Progress: func(p perfseed.Progress) {
			rate := float64(p.Done) / p.Elapsed.Seconds()
			fmt.Printf("  %d / %d incidents · %.0f par seconde · base : %.2f Go\n",
				p.Done, p.Total, rate, float64(p.DatabaseBytes)/bytesPerGB)
		},
	})
	if err != nil {
		return err
	}

	fmt.Println("Statistiques et carte de visibilité (VACUUM ANALYZE)…")
	if err := perfseed.Settle(ctx, db); err != nil {
		return err
	}
	sizes, err := perfseed.Sizes(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("Terminé en %s.\n\n%-22s %12s %10s\n", time.Since(began).Round(time.Second), "table", "lignes", "taille")
	for _, size := range sizes {
		fmt.Printf("%-22s %12d %8.2f Go\n", size.Name, size.Rows, float64(size.TotalBytes)/bytesPerGB)
	}
	return nil
}
