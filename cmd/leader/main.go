package main

import (
	"log/slog"
	"memdb/pkg/db"
	"memdb/pkg/server"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		panic("no port supplied on cmd arguments")
	}

	port := os.Args[1]

	if len(os.Args) < 3 {
		panic("no replicas args supplied, can not run without a minimum of 1 replica")
	}

	rootDir := "/tmp/memdb"

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	db := db.NewLeader(rootDir, logger)
	leaderServer := server.NewLeaderServer(db, port, logger)

	replicas := os.Args[2:]

	for _, replica := range replicas {
		leaderServer.AddReplica(replica)
	}

	leaderServer.RunServer()
}
