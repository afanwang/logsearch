package main

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestBasicFunctionality tests the core function
func TestBasicFunctionality(t *testing.T) {
	logger, err := NewSearchLogger(200 * time.Millisecond)
	assert.NoError(t, err, "Failed to create search logger")
	defer logger.Close()

	// Log a search
	err = logger.LogSearch("test")
	assert.NoError(t, err, "Failed to log search")

	// Wait for timeout and force flush
	time.Sleep(250 * time.Millisecond)
	assert.NoError(t, err, "Failed to force flush")

	// Verify storage
	stored, err := logger.GetStoredSearches()
	assert.NoError(t, err, "Failed to get stored searches")
	assert.Equal(t, 1, len(stored), "Expected 1 stored search, got %d", len(stored))
	assert.Equal(t, "test", stored[0], "Expected stored search to be 'test', got: %v", stored[0])
}

// TestWordProgression tests incremental word building
func TestWordProgression(t *testing.T) {
	logger, err := NewSearchLogger(150 * time.Millisecond)
	assert.NoError(t, err, "Failed to create search logger")
	defer logger.Close()

	// Log progressive searches quickly
	words := []string{"a", "ap", "app"}
	for _, word := range words {
		err := logger.LogSearch(word)
		assert.NoError(t, err, "Failed to log '%s'", word)
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for timeout
	time.Sleep(200 * time.Millisecond)

	// Check results
	stored, err := logger.GetStoredSearches()
	assert.NoError(t, err, "Failed to get stored searches")

	assert.Equal(t, 1, len(stored), "Expected 1 stored searches, got %d", len(stored))
	assert.Equal(t, "app", stored[0], "Expected stored search to be 'app'")
	t.Logf("Stored searches: %v", stored)
}
