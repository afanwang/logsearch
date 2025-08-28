package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSearchLoggerV2_BasicProgressiveTyping(t *testing.T) {
	logger, err := NewSearchLoggerV2()
	assert.NoError(t, err)
	defer logger.Close()

	// Test progressive typing for user - should consolidate to final word
	assert.NoError(t, logger.LogSearchV2("user_1", "b"))
	assert.NoError(t, logger.LogSearchV2("user_1", "bu"))
	assert.NoError(t, logger.LogSearchV2("user_1", "bus"))
	assert.NoError(t, logger.LogSearchV2("user_1", "business"))

	// User should only have "business" stored
	searches, err := logger.GetUserSearches("user_1")
	assert.NoError(t, err)
	assert.Equal(t, []string{"business"}, searches)
}

func TestSearchLoggerV2_MultipleUsers(t *testing.T) {
	logger, err := NewSearchLoggerV2()
	assert.NoError(t, err)
	defer logger.Close()

	// User1 progressive typing
	assert.NoError(t, logger.LogSearchV2("user_1", "c"))
	assert.NoError(t, logger.LogSearchV2("user_1", "ca"))
	assert.NoError(t, logger.LogSearchV2("user_1", "cat"))

	// User2 different progressive typing
	assert.NoError(t, logger.LogSearchV2("user_2", "d"))
	assert.NoError(t, logger.LogSearchV2("user_2", "do"))
	assert.NoError(t, logger.LogSearchV2("user_2", "dog"))

	// Both users typing same word separately
	assert.NoError(t, logger.LogSearchV2("user_1", "apple"))
	assert.NoError(t, logger.LogSearchV2("user_2", "apple"))

	// Check user1 searches
	user1Searches, err := logger.GetUserSearches("user_1")
	assert.NoError(t, err)
	assert.Contains(t, user1Searches, "cat")
	assert.Contains(t, user1Searches, "apple")
	assert.Len(t, user1Searches, 2)

	// Check user2 searches
	user2Searches, err := logger.GetUserSearches("user_2")
	assert.NoError(t, err)
	assert.Contains(t, user2Searches, "dog")
	assert.Contains(t, user2Searches, "apple")
	assert.Len(t, user2Searches, 2)
}

func TestSearchLoggerV2_InOrderVsOutOfOrder(t *testing.T) {
	logger, err := NewSearchLoggerV2()
	assert.NoError(t, err)
	defer logger.Close()

	// Test 1: In-order progressive typing (normal case)
	assert.NoError(t, logger.LogSearchV2("user_inorder", "b"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "bu"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "bus"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "busi"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "busin"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "busine"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "busines"))
	assert.NoError(t, logger.LogSearchV2("user_inorder", "business"))

	// Test 2: Out-of-order typing (full word first, then shorter)
	assert.NoError(t, logger.LogSearchV2("user_outorder", "business"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "busines"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "busine"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "busin"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "busi"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "bus"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "bu"))
	assert.NoError(t, logger.LogSearchV2("user_outorder", "b"))

	// Both users should end up with the same final result
	inOrderSearches, err := logger.GetUserSearches("user_inorder")
	assert.NoError(t, err)
	assert.Equal(t, []string{"business"}, inOrderSearches)

	outOfOrderSearches, err := logger.GetUserSearches("user_outorder")
	assert.NoError(t, err)
	assert.Equal(t, []string{"business"}, outOfOrderSearches)

	// Both should have exactly one record
	assert.Len(t, inOrderSearches, 1, "In-order user should have exactly one record")
	assert.Len(t, outOfOrderSearches, 1, "Out-of-order user should have exactly one record")
}
