package db

import "time"

const (
	BackupFile = "wordcounts.db"
	writeFreq  = 1 * time.Second
)

type Leader interface {
	CountWords(text string) map[string]int
	GetWordCount(word string) int
	GetWordsCounts() map[string]int
}

// Remote Replica
type Replica interface {
	GetWordCount(word string) int
	AddWordCount(word string, count int)
	SetWordsCounts(wordCounts map[string]int)
}

type LocalReplica interface {
	GetWordCount(word string) int
	Update() error
}
