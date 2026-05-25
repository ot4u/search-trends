package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ot4/search-trends/internal/domain/antifraud"
	domainstoplist "github.com/ot4/search-trends/internal/domain/stoplist"
	"github.com/ot4/search-trends/internal/domain/trending"
	"github.com/ot4/search-trends/internal/infrastructure/cache"
	"github.com/ot4/search-trends/internal/infrastructure/config"
	"github.com/ot4/search-trends/internal/infrastructure/metrics"
	httptransport "github.com/ot4/search-trends/internal/transport/http"
	transportnats "github.com/ot4/search-trends/internal/transport/nats"
	"github.com/ot4/search-trends/internal/usecase/ingest"
	stoplistusecase "github.com/ot4/search-trends/internal/usecase/stoplist"
	topusecase "github.com/ot4/search-trends/internal/usecase/top"
	"github.com/ot4/search-trends/pkg/logger"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}

	log := logger.New(cfg.LogLevel).With(
		logger.String("service", "search-trends"),
		logger.String("version", "dev"),
	)

	registry := metrics.New()
	stoplistStore := domainstoplist.NewStore(cfg.Stoplist)
	window := trending.NewWindow(cfg.WindowSeconds, time.Now().UTC())
	snapshotStore := cache.NewStore(cfg.WindowSeconds)
	processor := ingest.NewProcessor(ingest.ProcessorConfig{
		Window:           window,
		Stoplist:         stoplistStore,
		Detector:         antifraud.NewDetector(cfg.AntiFraudThresholdPerSecond, 10, 1),
		Snapshots:        snapshotStore,
		SnapshotLimits:   cfg.SnapshotLimits,
		MaxUniqueQueries: cfg.MaxUniqueQueries,
		MaxFutureSkew:    cfg.MaxFutureSkew,
		Observer:         registry,
	})

	ingestService := ingest.NewService(ingest.ServiceConfig{
		Logger:    log,
		Processor: processor,
		QueueSize: cfg.QueueSize,
		TickEvery: time.Second,
	})
	ingestService.Start()

	topService := topusecase.NewService(snapshotStore)
	stoplistService := stoplistusecase.NewService(stoplistStore)

	router := httptransport.NewRouter(httptransport.Dependencies{
		Logger:          log,
		Metrics:         registry,
		TopService:      topService,
		StoplistService: stoplistService,
		QueueSize:       ingestService.QueueSize,
	})

	server := &http.Server{
		Addr:         cfg.HTTP.Addr,
		Handler:      router,
		ReadTimeout:  cfg.HTTP.ReadTimeout,
		WriteTimeout: cfg.HTTP.WriteTimeout,
		IdleTimeout:  cfg.HTTP.IdleTimeout,
	}
	consumer := transportnats.NewConsumer(transportnats.Config{
		URL:           cfg.NATS.URL,
		Stream:        cfg.NATS.Stream,
		Subject:       cfg.NATS.Subject,
		Durable:       cfg.NATS.Durable,
		FetchBatch:    cfg.NATS.FetchBatch,
		FetchTimeout:  cfg.NATS.FetchTimeout,
		AckWait:       cfg.NATS.AckWait,
		MaxAckPending: cfg.NATS.MaxAckPending,
		AutoProvision: cfg.NATS.AutoProvision,
	}, ingestService, registry, log)

	rootCtx, stopSignals := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stopSignals()

	httpErrCh := make(chan error, 1)
	go func() {
		httpErrCh <- server.ListenAndServe()
	}()

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	consumerErrCh := make(chan error, 1)
	go func() {
		consumerErrCh <- consumer.Run(consumerCtx)
	}()

	log.Info(
		"service started",
		logger.String("http_addr", cfg.HTTP.Addr),
		logger.String("nats_url", cfg.NATS.URL),
	)

	consumerDone := false

	select {
	case <-rootCtx.Done():
		log.Info("shutdown signal received")
	case err := <-httpErrCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", logger.Err(err))
		}
	case err := <-consumerErrCh:
		consumerDone = true
		if err != nil {
			log.Error("nats consumer failed", logger.Err(err))
		}
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("http shutdown failed", logger.Err(err))
	}

	cancelConsumer()
	if !consumerDone {
		if err := <-consumerErrCh; err != nil && !errors.Is(err, context.Canceled) {
			log.Error("consumer shutdown failed", logger.Err(err))
		}
	}

	ingestService.CloseInput()
	ingestService.Wait()

	log.Info("service stopped")
	os.Exit(0)
}
