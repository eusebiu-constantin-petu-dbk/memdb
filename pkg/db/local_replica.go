package db

import (
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"sync"
)

type BaseLocalReplica struct {
	rootDir   string
	logger    *slog.Logger
	wordCount map[string]int
	lock      sync.RWMutex
}

func NewLocalReplica(rootDir string, logger *slog.Logger) *BaseLocalReplica {
	db := &BaseLocalReplica{
		rootDir:   rootDir,
		logger:    logger,
		wordCount: make(map[string]int),
	}

	return db
}

// GetWordCount returns the count of the given word.
func (db *BaseLocalReplica) GetWordCount(word string) int {
	db.lock.RLock()
	defer db.lock.RUnlock()

	return db.wordCount[word]
}

func (db *BaseLocalReplica) Update() error {
	var m map[string]int

	if err := db.getWordsCount(&m); err != nil {
		return err
	}

	db.lock.Lock()
	defer db.lock.Unlock()

	db.wordCount = m

	return nil
}

func (db *BaseLocalReplica) getWordsCount(v any) error {
	data, err := os.ReadFile(path.Join(db.rootDir, BackupFile))
	if err != nil {
		db.logger.Error("failed to read database file", "error", err)

		return err
	}

	return json.Unmarshal(data, v)
}
