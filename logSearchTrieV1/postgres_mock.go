package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// MockPostgresDB simulates PostgreSQL database operations
// Instead of using a real database, it uses an in-memory map
// so we can focus on the core business logic, the logging function
type MockPostgresDB struct {
	// map[serialID]SearchRecord
	searches map[int64]SearchRecord
	nextID   int64
	mutex    sync.RWMutex
}

type SearchRecord struct {
	ID              int64
	Word            string
	FirstSearchedAt time.Time
	LastUpdatedAt   time.Time
	SearchCount     int
}

// NewMockPostgresDB creates a new mock PostgreSQL database
func NewMockPostgresDB() *MockPostgresDB {
	return &MockPostgresDB{
		searches: make(map[int64]SearchRecord),
		nextID:   1,
	}
}

// CreateTable simulates creating the searches table
func (db *MockPostgresDB) CreateTable() error {
	log.Println("Mock PostgreSQL: CREATE TABLE searches (id SERIAL PRIMARY KEY, word VARCHAR UNIQUE, first_searched_at TIMESTAMP, last_updated_at TIMESTAMP, search_count INTEGER DEFAULT 1)")
	return nil
}

// InsertOrReplace simulates INSERT ... ON CONFLICT UPDATE
func (db *MockPostgresDB) InsertOrReplace(word string, firstSearched, lastUpdated time.Time) (int64, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if word already exists
	for id, record := range db.searches {
		if record.Word == word {
			record.LastUpdatedAt = lastUpdated
			record.SearchCount++
			db.searches[id] = record
			log.Printf("Mock PostgreSQL: UPDATE searches SET last_updated_at='%s', search_count=%d WHERE word='%s'",
				lastUpdated.Format(time.RFC3339), record.SearchCount, word)
			return id, nil
		}
	}

	// Insert new record
	id := db.nextID
	db.nextID++

	db.searches[id] = SearchRecord{
		ID:              id,
		Word:            word,
		FirstSearchedAt: firstSearched,
		LastUpdatedAt:   lastUpdated,
		SearchCount:     1,
	}

	log.Printf("Mock PostgreSQL: INSERT INTO searches (word, first_searched_at, last_updated_at) VALUES ('%s', '%s', '%s') RETURNING id=%d",
		word, firstSearched.Format(time.RFC3339), lastUpdated.Format(time.RFC3339), id)

	return id, nil
}

// Update simulates updating an existing record
func (db *MockPostgresDB) Update(id int64, newWord string, lastUpdated time.Time) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	record, exists := db.searches[id]
	if !exists {
		return fmt.Errorf("record with id %d not found", id)
	}

	record.Word = newWord
	record.LastUpdatedAt = lastUpdated
	record.SearchCount++
	db.searches[id] = record

	log.Printf("Mock PostgreSQL: UPDATE searches SET word='%s', last_updated_at='%s', search_count=%d WHERE id=%d",
		newWord, lastUpdated.Format(time.RFC3339), record.SearchCount, id)

	return nil
}

// GetAllSearchedWords simulates SELECT word FROM searches ORDER BY word
func (db *MockPostgresDB) GetAllSearchedWords() ([]string, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	words := make([]string, 0, len(db.searches))
	for _, record := range db.searches {
		words = append(words, record.Word)
	}

	log.Printf("Mock PostgreSQL: SELECT word FROM searches ORDER BY word - returned %d records", len(words))
	return words, nil
}

// GetAllRecords returns all stored search records
func (db *MockPostgresDB) GetAllRecords() []SearchRecord {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	records := make([]SearchRecord, 0, len(db.searches))
	for _, record := range db.searches {
		records = append(records, record)
	}

	log.Printf("Mock PostgreSQL: SELECT * FROM searches - returned %d records", len(records))
	return records
}

// Close simulates closing database connections
func (db *MockPostgresDB) Close() error {
	log.Println("Mock PostgreSQL: Database connection closed")
	return nil
}
