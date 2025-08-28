package main

import (
	"fmt"
	"log"
	"strings"
	"time"
)

// SearchLoggerV2 handles per-user search deduplication using database
// This version removes in-memory trie cache and relies on database for deduplication
type SearchLoggerV2 struct {
	db *MockPostgresDBV2
}

func NewSearchLoggerV2() (*SearchLoggerV2, error) {
	db := NewMockPostgresDBV2()
	return NewSearchLoggerV2WithDB(db)
}

func NewSearchLoggerV2WithDB(db *MockPostgresDBV2) (*SearchLoggerV2, error) {
	// Create table using MockPostgresDBV2
	if err := db.CreateTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	logger := &SearchLoggerV2{
		db: db,
	}

	return logger, nil
}

// LogSearchV2 processes a search term for a specific user
func (sl *SearchLoggerV2) LogSearchV2(userIdentifier, word string) error {
	if word == "" || userIdentifier == "" {
		return fmt.Errorf("word and userIdentifier cannot be empty")
	}

	word = strings.ToLower(strings.TrimSpace(word))
	now := time.Now()

	// Handle word extension and storage in a single operation
	if err := sl.storeOrExtendUserSearch(userIdentifier, word, now); err != nil {
		return fmt.Errorf("failed to store user search: %w", err)
	}

	return nil
}

// storeOrExtendUserSearch handles both word extension and storage in a single operation
func (sl *SearchLoggerV2) storeOrExtendUserSearch(userIdentifier, word string, timestamp time.Time) error {
	// Get all existing searches for this user
	existingWords, err := sl.db.GetUserSearches(userIdentifier)
	if err != nil {
		return err
	}

	// Check if the new word extends any existing shorter word (forward extension)
	for _, existingWord := range existingWords {
		if len(existingWord) < len(word) && strings.HasPrefix(word, existingWord) {
			fmt.Printf(" (extending '%s' to '%s')", existingWord, word)

			// Update the shorter word to the new longer word
			if err := sl.db.UpdateUserSearchByWord(userIdentifier, existingWord, word, timestamp); err != nil {
				log.Printf("Error updating user search from '%s' to '%s': %v", existingWord, word, err)
				return err
			}

			return nil
		}
	}

	// Check if the new word is a prefix of any existing longer word (out of order case)
	for _, existingWord := range existingWords {
		if len(word) < len(existingWord) && strings.HasPrefix(existingWord, word) {
			fmt.Printf(" (ignoring prefix of '%s')", existingWord)
			return nil
		}
	}

	// No extension found, store as new search or update existing
	_, err = sl.db.InsertOrUpdateUserSearch(userIdentifier, word, timestamp, timestamp)
	if err != nil {
		return err
	}

	fmt.Printf(" (new)")
	return nil
}

func (sl *SearchLoggerV2) GetUserSearches(userIdentifier string) ([]string, error) {
	return sl.db.GetUserSearches(userIdentifier)
}

func (sl *SearchLoggerV2) Close() error {
	return sl.db.Close()
}

type UserIdentifierGenerator struct {
	counter int
}

// NewUserIdentifierGenerator creates a new generator
func NewUserIdentifierGenerator() *UserIdentifierGenerator {
	return &UserIdentifierGenerator{counter: 0}
}

// GenerateAnonID generates an anonymous ID for guest users
func (gen *UserIdentifierGenerator) GenerateAnonID() string {
	gen.counter++
	return fmt.Sprintf("guest_%d", gen.counter)
}

// GenerateUserID generates a user ID for logged-in users
func (gen *UserIdentifierGenerator) GenerateUserID() string {
	gen.counter++
	userID := fmt.Sprintf("%d", gen.counter)
	return fmt.Sprintf("user_%s", userID)
}
