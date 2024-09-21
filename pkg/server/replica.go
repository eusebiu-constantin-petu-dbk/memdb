package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"memdb/pkg/db"
	dbErrs "memdb/pkg/errors"
	"net/http"
	"time"
)

type ReplicaServer struct {
	leader string
	db     db.Replica
	port   string
	server *http.Server
	logger *slog.Logger
}

func NewReplicaServer(replica db.Replica, port string, leader string, logger *slog.Logger) *ReplicaServer {
	return &ReplicaServer{
		db:     replica,
		leader: leader,
		port:   port,
		logger: logger,
	}
}

func (sv *ReplicaServer) requestLeaderSync() error {
	// wait for leader to become available before syncing
	for {
		resp, err := http.Get(sv.leader + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			break
		}

		sv.logger.Info("waiting for leader to become available...", "leader", sv.leader)
		time.Sleep(2 * time.Second)
	}

	resp, err := http.Get(sv.leader + "/sync")
	if err != nil {
		sv.logger.Error("failed to make GET request to sync from leader", "leader", sv.leader, "error", err)

		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		sv.logger.Error("failed to sync from leader", "leader", sv.leader, "status_code", resp.StatusCode)

		return dbErrs.ErrorOnSync
	}

	wordsCounts := make(map[string]int)

	if err := json.NewDecoder(resp.Body).Decode(&wordsCounts); err != nil {
		sv.logger.Error("failed to decode sync response from leader", "leader", sv.leader, "status_code", resp.StatusCode)

		return err
	}

	sv.db.SetWordsCounts(wordsCounts)

	return nil
}

func (sv *ReplicaServer) updateHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		updates := make(map[string]int)

		if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
			sv.logger.Error("failed to serialize sync response body", "error", err)

			http.Error(w, "invalid sync request", http.StatusBadRequest)

			return
		}

		for key, val := range updates {
			sv.db.AddWordCount(key, val)
		}

		w.WriteHeader(http.StatusAccepted)
	})
}

func (sv *ReplicaServer) getHandler() http.Handler {
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

func (sv *ReplicaServer) healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
}

func (sv *ReplicaServer) RunServer() {
	router := http.NewServeMux()

	router.Handle("/health", recoverMiddleware(sv.healthHandler()))
	router.Handle("/wordcount", recoverMiddleware(sv.getHandler()))
	router.Handle("/update", recoverMiddleware(sv.updateHandler()))

	sv.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", sv.port),
		Handler: router,
	}

	if err := sv.requestLeaderSync(); err != nil {
		sv.logger.Error("failed to sync from leader, running out of sync", "error", err)
	}

	sv.logger.Info("server listening", "port", sv.port)

	if err := sv.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		sv.logger.Error("failed to start server", "error", err)
	}
}

func (sv *ReplicaServer) Shutdown(ctx context.Context) error {
	return sv.server.Shutdown(ctx)
}
