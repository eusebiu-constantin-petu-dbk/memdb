package db

import (
	"log/slog"
	"sync"
)

type BaseReplica struct {
	wordCount map[string]int
	lock      sync.RWMutex
	logger    *slog.Logger
}

func NewReplica(logger *slog.Logger) *BaseReplica {
	return &BaseReplica{
		wordCount: make(map[string]int),
		logger:    logger,
	}
}

func (db *BaseReplica) GetWordCount(word string) int {
	db.lock.RLock()
	defer db.lock.RUnlock()

	count := db.wordCount[word]
	return count
}

func (db *BaseReplica) AddWordCount(word string, count int) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.wordCount[word] += count
}

func (db *BaseReplica) SetWordsCounts(wordCounts map[string]int) {
	db.lock.Lock()
	defer db.lock.Unlock()

	db.wordCount = wordCounts
}
