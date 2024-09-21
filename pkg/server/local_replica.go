package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"memdb/pkg/db"
	"net/http"
)

type LocalReplica struct {
	db       db.LocalReplica
	port     string
	replicas []string
	server   *http.Server
	logger   *slog.Logger
}

func NewLocalReplica(replica db.LocalReplica, port string, logger *slog.Logger) *LocalReplica {
	return &LocalReplica{
		db:       replica,
		port:     port,
		replicas: []string{},
		logger:   logger,
	}
}

func (sv *LocalReplica) getHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		word := r.URL.Query().Get("word")
		if word == "" {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		response := make(map[string]int)

		wordCount := sv.db.GetWordCount(word)
		response[word] = wordCount

		data, err := json.Marshal(response)
		if err != nil {
			sv.logger.Error("failed to serialize GET response body", "error", err)

			http.Error(w, "failed to serialize GET response body", http.StatusInternalServerError)

			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if _, err = w.Write(data); err != nil {
			sv.logger.Error("failed to send sync data to replica")
		}
	})
}

func (sv *LocalReplica) updateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// ignore request just trigger a sync
		if err := sv.db.Update(); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
}

func (sv *LocalReplica) healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sv.logger.Info("GET /health (health check)")
		w.WriteHeader(http.StatusOK)
	})
}

func (sv *LocalReplica) RunServer() {
	router := http.NewServeMux()

	router.Handle("/wordcount", recoverMiddleware(sv.getHandler()))
	router.Handle("/health", recoverMiddleware(sv.healthHandler()))
	router.Handle("/update", recoverMiddleware(sv.updateHandler()))

	sv.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", sv.port),
		Handler: router,
	}

	sv.logger.Info("server listening", "port", sv.port)

	if err := sv.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		sv.logger.Error("failed to start server", "error", err)
	}
}

func (sv *LocalReplica) Shutdown(ctx context.Context) error {
	return sv.server.Shutdown(ctx)
}
