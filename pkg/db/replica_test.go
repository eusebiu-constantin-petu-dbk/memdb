package db_test

import (
	"log/slog"
	"memdb/pkg/db"
	"os"
	"testing"
)

func TestReplicaGetWordCount(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	replica := db.NewReplica(logger)
	replica.SetWordsCounts(map[string]int{"test": 5})

	count := replica.GetWordCount("test")
	if count != 5 {
		t.Fatalf("Expected word count to be 5, got %d", count)
	}

	count = replica.GetWordCount("nonexistent")
	if count != 0 {
		t.Fatalf("Expected word count to be 0 for nonexistent word, got %d", count)
	}
}

func TestAddWordCount(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	replica := db.NewReplica(logger)
	replica.AddWordCount("test", 3)

	if count := replica.GetWordCount("test"); count != 3 {
		t.Fatalf("Expected word count to be 3, got %d", count)
	}

	replica.AddWordCount("test", 2)
	if count := replica.GetWordCount("test"); count != 5 {
		t.Fatalf("Expected word count to be 5, got %d", count)
	}
}

func TestSetWordsCounts(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	replica := db.NewReplica(logger)
	wordCounts := map[string]int{
		"hello": 1,
		"world": 2,
	}
	replica.SetWordsCounts(wordCounts)

	if count := replica.GetWordCount("hello"); count != 1 {
		t.Fatalf("Expected word count for 'hello' to be 1, got %d", count)
	}

	if count := replica.GetWordCount("world"); count != 2 {
		t.Fatalf("Expected word count for 'world' to be 2, got %d", count)
	}
}
