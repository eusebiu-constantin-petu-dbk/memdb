package db_test

import (
	"log/slog"
	"memdb/pkg/db"
	"os"
	"testing"
)

func TestCountWords(t *testing.T) {
	rootDir := t.TempDir()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db := db.NewLeader(rootDir, logger)

	text := "hello world hello"
	expectedCounts := map[string]int{"hello": 2, "world": 1}

	counts := db.CountWords(text)

	for word, count := range expectedCounts {
		if counts[word] != count {
			t.Errorf("expected %d for word %s, got %d", count, word, counts[word])
		}
	}
}

func TestGetWordCount(t *testing.T) {
	rootDir := t.TempDir()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db := db.NewLeader(rootDir, logger)

	db.CountWords("hello world hello")

	tests := []struct {
		word     string
		expected int
	}{
		{"hello", 2},
		{"world", 1},
		{"nonexistent", 0},
	}

	for _, tt := range tests {
		count := db.GetWordCount(tt.word)
		if count != tt.expected {
			t.Errorf("expected %d for word %s, got %d", tt.expected, tt.word, count)
		}
	}
}

func TestGetWordsCounts(t *testing.T) {
	rootDir := t.TempDir()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	db := db.NewLeader(rootDir, logger)

	db.CountWords("hello world hello")

	expectedCounts := map[string]int{"hello": 2, "world": 1}
	counts := db.GetWordsCounts()

	for word, count := range expectedCounts {
		if counts[word] != count {
			t.Errorf("expected %d for word %s, got %d", count, word, counts[word])
		}
	}
}
