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

	// Make it an argument
	rootDir := "/tmp/memdb"

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	db := db.NewLocalReplica(rootDir, logger)
	localReplicaServer := server.NewLocalReplica(db, port, logger)

	localReplicaServer.RunServer()
}
