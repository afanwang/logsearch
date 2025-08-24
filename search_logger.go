package main

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

// TrieNode represents a node in the trie structure
type TrieNode struct {
	children    map[rune]*TrieNode
	isEndOfWord bool
	lastSeen    time.Time
	// ID of the record in DB if stored
	dbID *int64
}

// SearchLogger handles search deduplication and storage
type SearchLogger struct {
	prefixTree *TrieNode
	db         *MockPostgresDB
	mutex      sync.RWMutex
	// timeout is how long to wait before considering a word "complete"
	timeout time.Duration
	// stopChan to better control the flushing routine
	stopChan chan struct{}
}

// NewSearchLogger creates a new SearchLogger instance
// It will be called by the http server which hosts
// api /Query={word}&Limit={limit}&Verified={bool}
func NewSearchLogger(timeout time.Duration) (*SearchLogger, error) {
	db := NewMockPostgresDB()
	return NewSearchLoggerWithDB(timeout, db)
}

// NewSearchLoggerWithDB creates a new SearchLogger
func NewSearchLoggerWithDB(timeout time.Duration, db *MockPostgresDB) (*SearchLogger, error) {
	// Create table using MockPostgresDB
	if err := db.CreateTable(); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	logger := &SearchLogger{
		prefixTree: &TrieNode{children: make(map[rune]*TrieNode)},
		db:         db,
		timeout:    timeout,
		stopChan:   make(chan struct{}),
	}

	// Load existing words from database and build the prefix tree
	if err := logger.loadExistingWords(); err != nil {
		return nil, fmt.Errorf("failed to load existing words: %w", err)
	}

	// Start flushCompletedWordToDB goroutine
	go logger.flushCompletedWordToDBRoutine()

	return logger, nil
}

// LogSearch processes a search term and stores it
func (sl *SearchLogger) LogSearch(word string) error {
	if word == "" {
		return nil
	}

	word = strings.ToLower(strings.TrimSpace(word))
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	node := sl.prefixTree
	now := time.Now()

	// Traverse/build the trie
	for _, char := range word {
		if node.children[char] == nil {
			node.children[char] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[char]
	}

	// Update the last seen timestamp for this node
	node.lastSeen = now

	// Check if this word extends an existing stored word
	if err := sl.handleWordExtension(word, node); err != nil {
		return fmt.Errorf("failed to handle word extension: %w", err)
	}

	return nil
}

// handleWordExtension checks if this word extends a previously stored shorter word
func (sl *SearchLogger) handleWordExtension(word string, currentNode *TrieNode) error {
	// Look for shorter prefixes that might be stored in DB
	node := sl.prefixTree
	for i, char := range []rune(word) {
		if node.children[char] == nil {
			break
		}
		node = node.children[char]

		// If we find a shorter word that's stored in DB, need to update it
		if node.isEndOfWord && node.dbID != nil && i < len([]rune(word))-1 {
			prefix := word[:i+1]
			log.Printf("Found shorter stored word '%s', will replace with '%s'", prefix, word)

			// Update the existing record
			if err := sl.updateStoredWord(*node.dbID, word); err != nil {
				return fmt.Errorf("failed to update stored word: %w", err)
			}

			// Move the DB ID to the current (longer) word
			currentNode.dbID = node.dbID
			node.dbID = nil
		}
	}

	return nil
}

// updateStoredWord updates an existing record in the database
func (sl *SearchLogger) updateStoredWord(id int64, newWord string) error {
	return sl.db.Update(id, newWord, time.Now())
}

// storeWordToDB stores a word to the database
func (sl *SearchLogger) storeWordToDB(word string, node *TrieNode) error {
	now := time.Now()
	id, err := sl.db.InsertOrReplace(word, now, now)
	if err != nil {
		return err
	}

	node.dbID = &id
	log.Printf("Stored word '%s' to database with ID %d", word, id)
	return nil
}

// flushCompletedWordToDBRoutine runs periodically to store words that haven't been extended
func (sl *SearchLogger) flushCompletedWordToDBRoutine() {
	ticker := time.NewTicker(sl.timeout / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sl.processTimedOutWords()
		case <-sl.stopChan:
			return
		}
	}
}

// processTimedOutWords finds words that haven't been updated recently and stores them
func (sl *SearchLogger) processTimedOutWords() {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	cutoffTime := time.Now().Add(-sl.timeout)

	// Find all timed-out words
	timedOutWords := make(map[string]*TrieNode)
	sl.findAllTimedOutWords(sl.prefixTree, "", cutoffTime, timedOutWords)

	if len(timedOutWords) == 0 {
		return
	}

	// Store words that are not prefixes of any other word
	for word, node := range timedOutWords {
		if node.dbID != nil {
			continue
		}

		// Check if this word is a prefix of any other word
		isPrefixOfOther := sl.isPrefixOfAnyWord(word)

		// Only store if this word is not a prefix of any other word
		if !isPrefixOfOther {
			node.isEndOfWord = true
			if err := sl.storeWordToDB(word, node); err != nil {
				log.Printf("Error storing word '%s': %v", word, err)
			}
		}
	}
}

// Close closes the database connection and stops background routines
func (sl *SearchLogger) Close() error {
	close(sl.stopChan)
	return sl.db.Close()
}

// findAllTimedOutWords recursively finds all words that have timed out
func (sl *SearchLogger) findAllTimedOutWords(node *TrieNode, currentWord string, cutoffTime time.Time, result map[string]*TrieNode) {
	// Check if this node represents a timed-out word
	if !node.lastSeen.IsZero() && node.lastSeen.Before(cutoffTime) {
		result[currentWord] = node
	}

	// Process children
	for char, child := range node.children {
		sl.findAllTimedOutWords(child, currentWord+string(char), cutoffTime, result)
	}
}

// isPrefixOfAnyWord checks if a word is a prefix of any other word in the trie
func (sl *SearchLogger) isPrefixOfAnyWord(word string) bool {
	node := sl.prefixTree

	// Navigate to the word's node
	for _, char := range word {
		if node.children[char] == nil {
			return false // Word doesn't exist in trie
		}
		node = node.children[char]
	}

	// Check if this node has any children with lastSeen set
	return sl.hasAnySearchedDescendants(node)
}

// hasAnySearchedDescendants checks if any descendant nodes have been searched
func (sl *SearchLogger) hasAnySearchedDescendants(node *TrieNode) bool {
	for _, child := range node.children {
		if !child.lastSeen.IsZero() {
			return true
		}
		if sl.hasAnySearchedDescendants(child) {
			return true
		}
	}
	return false
}

// GetStoredSearches returns all stored searches
func (sl *SearchLogger) GetStoredSearches() ([]string, error) {
	sl.mutex.RLock()
	defer sl.mutex.RUnlock()

	return sl.db.GetAllSearchedWords()
}

// loadExistingWords loads all words from database and builds the trie
func (sl *SearchLogger) loadExistingWords() error {
	words, err := sl.db.GetAllSearchedWords()
	if err != nil {
		return fmt.Errorf("failed to get words from database: %w", err)
	}

	log.Printf("Loading %d words from database into trie", len(words))

	for _, word := range words {
		if err := sl.buildTrieFromWord(word); err != nil {
			log.Printf("Error building trie for word '%s': %v", word, err)
			continue
		}
	}

	return nil
}

// buildTrieFromWord builds trie path for a stored word
func (sl *SearchLogger) buildTrieFromWord(word string) error {
	node := sl.prefixTree

	for _, char := range word {
		if node.children[char] == nil {
			node.children[char] = &TrieNode{children: make(map[rune]*TrieNode)}
		}
		node = node.children[char]
	}

	node.isEndOfWord = true
	node.lastSeen = time.Now()
	return nil
}
