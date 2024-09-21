package db

import (
	"encoding/json"
	"log/slog"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type BaseLeader struct {
	wordCount   map[string]int
	dblock      sync.RWMutex
	rootDir     string
	needsBackup bool
	logger      *slog.Logger
}

func NewLeader(rootDir string, logger *slog.Logger) *BaseLeader {
	db := &BaseLeader{
		wordCount: make(map[string]int),
		rootDir:   rootDir,
		logger:    logger,
	}

	if _, err := os.Stat(rootDir); os.IsNotExist(err) {
		if err := os.MkdirAll(rootDir, 0700); err != nil {
			logger.Error("failed to create database rootDir")
		}
	}

	// refactor constructor to return error
	_ = db.restore()

	go db.runBackup()

	return db
}

// CountWords increments the count of each word in the given text.
func (db *BaseLeader) CountWords(text string) map[string]int {
	words := strings.Fields(text)

	db.dblock.Lock()
	defer db.dblock.Unlock()

	wordsCounts := make(map[string]int)
	for _, word := range words {
		db.wordCount[word]++
		wordsCounts[word]++
	}

	db.needsBackup = true

	return wordsCounts
}

// GetWordCount returns the count of the given word.
func (db *BaseLeader) GetWordCount(word string) int {
	db.dblock.RLock()
	defer db.dblock.RUnlock()

	count, ok := db.wordCount[word]
	if !ok {
		return 0
	}

	return count
}

// GetWordsCounts returns the current word count data.
func (db *BaseLeader) GetWordsCounts() map[string]int {
	db.dblock.RLock()
	defer db.dblock.RUnlock()

	wordCounts := make(map[string]int)
	for k, v := range db.wordCount {
		wordCounts[k] = v
	}

	return wordCounts
}

func (db *BaseLeader) backup() error {
	data, err := json.Marshal(db.wordCount)
	if err != nil {
		return err
	}

	if err := os.WriteFile(path.Join(db.rootDir, BackupFile), data, 0644); err != nil {
		db.logger.Error("error writing backup file", "error", err)

		return err
	}

	db.needsBackup = false

	return nil
}

func (db *BaseLeader) runBackup() {
	for {
		db.dblock.Lock()

		if db.needsBackup {
			if err := db.backup(); err != nil {
				db.logger.Error("failed to backup database...", "error", err)
			}
		}

		db.dblock.Unlock()

		time.Sleep(writeFreq)
	}
}

func (db *BaseLeader) restore() error {
	data, err := os.ReadFile(path.Join(db.rootDir, BackupFile))
	if err != nil {
		db.logger.Info("No persistence file found, starting fresh.")

		return err
	}

	return json.Unmarshal(data, &db.wordCount)
}
