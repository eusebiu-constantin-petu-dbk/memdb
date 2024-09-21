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
		panic("no leader arg supplied, can not run without a leader")
	}

	leader := os.Args[2]

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	db := db.NewReplica(logger)

	replicaServer := server.NewReplicaServer(db, port, leader, logger)

	replicaServer.RunServer()
}
