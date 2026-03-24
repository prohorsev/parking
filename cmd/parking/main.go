package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/parking/api/internal/aggregator"
	"github.com/parking/api/internal/api"
	"github.com/parking/api/internal/config"
	"github.com/parking/api/internal/provider/providera"
	"github.com/parking/api/internal/provider/providerb"
	"github.com/parking/api/internal/queue"
	pgstore "github.com/parking/api/internal/repository/postgres"
)

func main() {
	cfg := config.Load()

	db, err := pgstore.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer func() {
		if err = db.Close(); err != nil {
			log.Printf("db close error: %v", err)
		}
	}()

	if err = pgstore.RunMigrations(db, cfg.MigrationsPath); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	repo := pgstore.NewRepository(db)

	mqConn, err := queue.Connect(cfg.RabbitMQURL)
	if err != nil {
		log.Fatalf("rabbitmq connect: %v", err)
	}
	defer func() {
		if err = mqConn.Close(); err != nil {
			log.Printf("rabbitmq close error: %v", err)
		}
	}()

	publisher, err := queue.NewPublisher(mqConn)
	if err != nil {
		log.Fatalf("rabbitmq publisher: %v", err)
	}
	defer publisher.Close()

	consumer, err := queue.NewConsumer(mqConn, repo)
	if err != nil {
		log.Fatalf("rabbitmq consumer: %v", err)
	}
	defer consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err = consumer.Start(ctx); err != nil {
		log.Fatalf("rabbitmq consumer start: %v", err)
	}

	addrA := providera.StartMockServer(cfg.ProviderAPort)
	addrB := providerb.StartMockServer(cfg.ProviderBPort)

	agg := aggregator.New(
		repo,
		providera.NewClient(addrA),
		providerb.NewClient(addrB),
	)

	go agg.StartRefreshLoop(ctx, cfg.RefreshInterval)

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      api.NewRouter(agg, publisher),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("API listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()

	<-quit

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutCancel()
	if err = srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}
