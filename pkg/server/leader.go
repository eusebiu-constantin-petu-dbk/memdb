package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"memdb/pkg/db"
	"net/http"
)

const (
	maxTextLength = 65535
)

type LeaderServer struct {
	db       db.Leader
	port     string
	replicas []string
	server   *http.Server
	logger   *slog.Logger
}

func NewLeaderServer(leader db.Leader, port string, logger *slog.Logger) *LeaderServer {
	return &LeaderServer{
		db:       leader,
		port:     port,
		replicas: []string{},
		logger:   logger,
	}
}

func (sv *LeaderServer) AddReplica(replica string) {
	sv.replicas = append(sv.replicas, replica)
}

// POST handler for counting words
func (sv *LeaderServer) countWordsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sv.logger.Info("POST /post (count words request)")
		text := r.FormValue("text")

		if err := validateInput(text); err != nil {
			http.Error(w, "No text provided", http.StatusBadRequest)
			return
		}

		updateBuffer := sv.db.CountWords(text)

		go sv.replicate(updateBuffer)

		w.WriteHeader(http.StatusAccepted)
	})
}

func (sv *LeaderServer) replicate(updateBuffer map[string]int) error {
	if len(updateBuffer) > 0 {
		sv.logger.Info("replicating to followers", "updateBuffer", updateBuffer)

		data, err := json.Marshal(updateBuffer)
		if err != nil {
			sv.logger.Error("error marshaling replication data", "error", err)
			return err
		}

		sv.sendToReplicas(data)
	}

	return nil
}

func (sv *LeaderServer) sendToReplicas(data []byte) {
	done := make(chan struct{})
	for _, replica := range sv.replicas {
		replica := replica
		go func() {
			// wait for a minimum of one replica to respond
			defer func() {
				select {
				case done <- struct{}{}:
				default:
				}
			}()

			body := bytes.NewReader(data)

			resp, err := http.Post(replica+"/update", "application/json", body)
			if err != nil {
				sv.logger.Error("failed to replicate updates to follower", "error", err)
				return
			}

			if resp.StatusCode != http.StatusAccepted {
				sv.logger.Error("failed to replicate", "status code", resp.StatusCode)
			}
		}()
	}

	// Wait for one goroutine to respond
	<-done
}

// GET handler for replica full sync
func (sv *LeaderServer) syncReplicaHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sv.logger.Info("GET /sync (replica full sync request)")

		wordsCounts := sv.db.GetWordsCounts()

		data, err := json.Marshal(wordsCounts)
		if err != nil {
			http.Error(w, "failed to serialize database", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if _, err = w.Write(data); err != nil {
			sv.logger.Error("failed to send sync data to replica", "error", err)
		}
	})
}

func (sv *LeaderServer) healthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sv.logger.Info("GET /health (health check)")
		w.WriteHeader(http.StatusOK)
	})
}

func (sv *LeaderServer) RunServer() {
	router := http.NewServeMux()

	router.Handle("/health", recoverMiddleware(sv.healthHandler()))
	router.Handle("/post", recoverMiddleware(sv.countWordsHandler()))
	router.Handle("/sync", recoverMiddleware(sv.syncReplicaHandler()))

	sv.server = &http.Server{
		Addr:    fmt.Sprintf(":%s", sv.port),
		Handler: router,
	}

	sv.logger.Info("server listening", "port", sv.port)

	if err := sv.server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		sv.logger.Error("failed to start server", "error", err)
	}
}

func (sv *LeaderServer) Shutdown(ctx context.Context) error {
	return sv.server.Shutdown(ctx)
}

func validateInput(text string) error {
	if text == "" {
		return http.ErrBodyNotAllowed
	}

	if len(text) > maxTextLength {
		return http.ErrContentLength
	}

	return nil
}
