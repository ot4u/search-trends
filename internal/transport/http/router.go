package httptransport

import (
	"encoding/json"
	"errors"
	nethttp "net/http"
	"strconv"
	"time"

	chi "github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"

	"github.com/ot4/search-trends/internal/infrastructure/metrics"
	stoplistusecase "github.com/ot4/search-trends/internal/usecase/stoplist"
	topusecase "github.com/ot4/search-trends/internal/usecase/top"
	"github.com/ot4/search-trends/pkg/logger"
)

type Dependencies struct {
	Logger          logger.Logger
	Metrics         *metrics.Metrics
	TopService      topusecase.Reader
	StoplistService stoplistusecase.Manager
	QueueSize       func() int
}

func NewRouter(deps Dependencies) nethttp.Handler {
	if deps.Logger == nil {
		deps.Logger = logger.Nop()
	}
	if deps.QueueSize == nil {
		deps.QueueSize = func() int { return 0 }
	}

	router := chi.NewRouter()
	router.Use(chimiddleware.RequestID)
	router.Use(chimiddleware.RealIP)
	router.Use(chimiddleware.Recoverer)
	router.Use(loggingMiddleware(deps.Logger, deps.Metrics, deps.QueueSize))

	router.Get("/healthz", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
		writeJSON(w, nethttp.StatusOK, map[string]string{"status": "ok"})
	})
	router.Handle("/metrics", deps.Metrics.Handler())

	router.Route("/api/v1", func(api chi.Router) {
		api.Get("/trending", func(w nethttp.ResponseWriter, r *nethttp.Request) {
			limit := 10
			if raw := r.URL.Query().Get("limit"); raw != "" {
				parsed, err := strconv.Atoi(raw)
				if err != nil {
					writeError(w, nethttp.StatusBadRequest, "invalid limit")
					return
				}
				limit = parsed
			}

			snapshot, err := deps.TopService.GetSnapshot(limit)
			if err != nil {
				switch {
				case errors.Is(err, topusecase.ErrInvalidLimit), errors.Is(err, topusecase.ErrLimitTooLarge):
					writeError(w, nethttp.StatusBadRequest, err.Error())
				default:
					writeError(w, nethttp.StatusServiceUnavailable, err.Error())
				}
				return
			}

			writeJSON(w, nethttp.StatusOK, snapshot)
		})

		api.Get("/stoplist", func(w nethttp.ResponseWriter, _ *nethttp.Request) {
			writeJSON(w, nethttp.StatusOK, map[string]any{
				"items": deps.StoplistService.List(),
			})
		})

		api.Post("/stoplist", func(w nethttp.ResponseWriter, r *nethttp.Request) {
			var req struct {
				Word string `json:"word"`
			}

			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				writeError(w, nethttp.StatusBadRequest, "invalid json body")
				return
			}

			word, changed, err := deps.StoplistService.Add(req.Word)
			if err != nil {
				if errors.Is(err, stoplistusecase.ErrEmptyWord) || errors.Is(err, stoplistusecase.ErrWordMustBeToken) {
					writeError(w, nethttp.StatusBadRequest, err.Error())
					return
				}
				writeError(w, nethttp.StatusInternalServerError, err.Error())
				return
			}

			status := nethttp.StatusCreated
			if !changed {
				status = nethttp.StatusOK
			}

			writeJSON(w, status, map[string]any{
				"word":    word,
				"changed": changed,
			})
		})

		api.Delete("/stoplist/{word}", func(w nethttp.ResponseWriter, r *nethttp.Request) {
			word, changed, err := deps.StoplistService.Remove(chi.URLParam(r, "word"))
			if err != nil {
				if errors.Is(err, stoplistusecase.ErrEmptyWord) || errors.Is(err, stoplistusecase.ErrWordMustBeToken) {
					writeError(w, nethttp.StatusBadRequest, err.Error())
					return
				}
				writeError(w, nethttp.StatusInternalServerError, err.Error())
				return
			}

			writeJSON(w, nethttp.StatusOK, map[string]any{
				"word":    word,
				"changed": changed,
			})
		})
	})

	return router
}

type statusWriter struct {
	nethttp.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

func loggingMiddleware(log logger.Logger, metrics *metrics.Metrics, queueSize func() int) func(nethttp.Handler) nethttp.Handler {
	return func(next nethttp.Handler) nethttp.Handler {
		return nethttp.HandlerFunc(func(w nethttp.ResponseWriter, r *nethttp.Request) {
			start := time.Now()
			writer := &statusWriter{ResponseWriter: w, status: nethttp.StatusOK}

			next.ServeHTTP(writer, r)

			routePattern := chi.RouteContext(r.Context()).RoutePattern()
			if routePattern == "" {
				routePattern = r.URL.Path
			}

			latency := time.Since(start)
			requestID := chimiddleware.GetReqID(r.Context())
			log.Info(
				"http request completed",
				logger.String("request_id", requestID),
				logger.String("method", r.Method),
				logger.String("path", routePattern),
				logger.Int("status", writer.status),
				logger.Any("latency", latency.String()),
				logger.Int("queue_size", queueSize()),
			)

			if metrics != nil {
				metrics.ObserveHTTP(r.Method, routePattern, writer.status, latency)
			}
		})
	}
}

func writeJSON(w nethttp.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w nethttp.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{
		"error": message,
	})
}
