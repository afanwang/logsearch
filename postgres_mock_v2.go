package main

import (
	"fmt"
	"sync"
	"time"
)

// MockPostgresDBV2 simulates PostgreSQL database operations for Version 2
type MockPostgresDBV2 struct {
	// map[recordId]UserSearchRecord
	userSearches map[string]UserSearchRecord
	nextID       int64
	mutex        sync.RWMutex
}

type UserSearchRecord struct {
	ID int64
	// user_id for logged-in; anon_id for guest
	UserIdentifier  string
	SearchWord      string
	FirstSearchedAt time.Time
	LastUpdatedAt   time.Time
	SearchCount     int
}

// NewMockPostgresDBV2 creates a new mock PostgreSQL database for Version 2
func NewMockPostgresDBV2() *MockPostgresDBV2 {
	return &MockPostgresDBV2{
		userSearches: make(map[string]UserSearchRecord),
		nextID:       1,
	}
}

// CreateTable simulates creating the user_searches table
func (db *MockPostgresDBV2) CreateTable() error {
	// log.Println("CREATE TABLE user_searches (id SERIAL PRIMARY KEY, user_identifier VARCHAR, search_word VARCHAR, first_searched_at TIMESTAMP, last_updated_at TIMESTAMP, search_count INTEGER DEFAULT 1, UNIQUE(user_identifier, search_word))")
	return nil
}

// InsertOrUpdateUserSearch simulates INSERT ... ON CONFLICT UPDATE
func (db *MockPostgresDBV2) InsertOrUpdateUserSearch(userIdentifier, word string, firstSearched, lastUpdated time.Time) (int64, error) {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Check if this user-word combination already exists
	for _, record := range db.userSearches {
		if record.UserIdentifier == userIdentifier && record.SearchWord == word {
			// Update existing record
			record.LastUpdatedAt = lastUpdated
			record.SearchCount++
			db.userSearches[fmt.Sprintf("%d", record.ID)] = record

			// log.Printf("UPDATE user_searches SET last_updated_at='%s', search_count=%d WHERE user_identifier='%s' AND search_word='%s'",
			//	lastUpdated.Format(time.RFC3339), record.SearchCount, userIdentifier, word)

			return record.ID, nil
		}
	}

	// Insert new record
	id := db.nextID
	db.nextID++

	db.userSearches[fmt.Sprintf("%d", id)] = UserSearchRecord{
		ID:              id,
		UserIdentifier:  userIdentifier,
		SearchWord:      word,
		FirstSearchedAt: firstSearched,
		LastUpdatedAt:   lastUpdated,
		SearchCount:     1,
	}

	// log.Printf("INSERT INTO user_searches (user_identifier, search_word, first_searched_at, last_updated_at) VALUES ('%s', '%s', '%s', '%s') RETURNING id=%d",
	//	userIdentifier, word, firstSearched.Format(time.RFC3339), lastUpdated.Format(time.RFC3339), id)

	return id, nil
}

// GetUserSearches returns all searches for a specific user
func (db *MockPostgresDBV2) GetUserSearches(userIdentifier string) ([]string, error) {
	db.mutex.RLock()
	defer db.mutex.RUnlock()

	var words []string
	for _, record := range db.userSearches {
		if record.UserIdentifier == userIdentifier {
			words = append(words, record.SearchWord)
		}
	}

	// log.Printf("SELECT search_word FROM user_searches WHERE user_identifier='%s' ORDER BY search_word - returned %d records", userIdentifier, len(words))

	return words, nil
}

// UpdateUserSearchByWord updates a user's search record from old word to new word
func (db *MockPostgresDBV2) UpdateUserSearchByWord(userIdentifier, oldWord, newWord string, lastUpdated time.Time) error {
	db.mutex.Lock()
	defer db.mutex.Unlock()

	// Find the record with the old word
	var oldRecord *UserSearchRecord
	var oldKey string
	for key, record := range db.userSearches {
		if record.UserIdentifier == userIdentifier && record.SearchWord == oldWord {
			rec := record // Create a copy
			oldRecord = &rec
			oldKey = key
			break
		}
	}

	if oldRecord == nil {
		return fmt.Errorf("record not found for user %s with word %s", userIdentifier, oldWord)
	}

	// Check if there's already a record with the new word
	var existingRecord *UserSearchRecord
	var existingKey string
	for key, record := range db.userSearches {
		if record.UserIdentifier == userIdentifier && record.SearchWord == newWord {
			rec := record // Create a copy
			existingRecord = &rec
			existingKey = key
			break
		}
	}

	// Remove the old record
	delete(db.userSearches, oldKey)

	if existingRecord != nil {
		// Merge with existing record
		mergedRecord := UserSearchRecord{
			ID:              existingRecord.ID, // Keep existing record's ID
			UserIdentifier:  userIdentifier,
			SearchWord:      newWord,
			FirstSearchedAt: existingRecord.FirstSearchedAt, // Keep earlier timestamp
			LastUpdatedAt:   lastUpdated,
			SearchCount:     existingRecord.SearchCount + oldRecord.SearchCount,
		}

		if oldRecord.FirstSearchedAt.Before(existingRecord.FirstSearchedAt) {
			mergedRecord.FirstSearchedAt = oldRecord.FirstSearchedAt
		}

		db.userSearches[existingKey] = mergedRecord
	} else {
		// No existing record with new word, just update the old record
		db.userSearches[oldKey] = UserSearchRecord{
			ID:              oldRecord.ID,
			UserIdentifier:  userIdentifier,
			SearchWord:      newWord,
			FirstSearchedAt: oldRecord.FirstSearchedAt,
			LastUpdatedAt:   lastUpdated,
			SearchCount:     oldRecord.SearchCount + 1,
		}
	}

	// log.Printf("UPDATE user_searches SET search_word='%s', last_updated_at='%s', search_count=%d WHERE user_identifier='%s' AND search_word='%s'",
	//	newWord, lastUpdated.Format(time.RFC3339), oldRecord.SearchCount+1, userIdentifier, oldWord)

	return nil
}

// Close simulates closing the database connection
func (db *MockPostgresDBV2) Close() error {
	// log.Println("Database connection closed")
	return nil
}
