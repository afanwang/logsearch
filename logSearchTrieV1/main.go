package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	demonSearchLogger()
}

// Demo function for the search logger functionality
func demonSearchLogger() {
	fmt.Println("=== Search Logger Demo ===")
	fmt.Println("1. Load existing words from DB into trie at startup")
	fmt.Println("2. Build new tries for new words")
	fmt.Println("3. Store words to DB with timeout mechanism")

	sharedDB := NewMockPostgresDB()

	fmt.Println("\n1. Creating initial logger and adding test data:")
	initialLogger, err := NewSearchLoggerWithDB(200*time.Millisecond, sharedDB)
	if err != nil {
		log.Fatal("Failed to create initial logger:", err)
	}

	testWords := []string{"apple", "application", "banana", "band"}
	for _, word := range testWords {
		if err := initialLogger.LogSearch(word); err != nil {
			log.Printf("Error adding '%s': %v", word, err)
		}
	}
	time.Sleep(300 * time.Millisecond)
	initialLogger.Close()

	fmt.Println("\n\n2. Creating new logger - load existing words from DB into trie:")
	logger, err := NewSearchLoggerWithDB(200*time.Millisecond, sharedDB)
	if err != nil {
		log.Fatal("Failed to create logger:", err)
	}
	defer logger.Close()

	fmt.Println("\n\n3. Words loaded from database into tries:")
	stored, err := logger.GetStoredSearches()
	if err != nil {
		log.Printf("Error getting stored words: %v", err)
	} else {
		for _, word := range stored {
			fmt.Printf("   - %s\n", word)
		}
	}

	fmt.Println("\n\n4. Adding new search terms - these will create new trie nodes:")
	newSearches := []string{
		"B", "Bu", "Bus", "Busi", "Business",
		"c", "ca", "cat", "cats",
		"d", "do", "dog",
	}

	for _, search := range newSearches {
		err := logger.LogSearch(search)
		if err != nil {
			log.Printf("Error adding '%s': %v", search, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	storedAfter, err := logger.GetStoredSearches()
	if err != nil {
		log.Printf("Error getting stored searches: %v", err)
		return
	}
	fmt.Printf("\n\n5. Stored searches after timeout: %v", storedAfter)

	fmt.Println("\n\n6. Testing word extension:")
	fmt.Println("   Adding 'Businesses' (extends 'Business')")

	if err := logger.LogSearch("Businesses"); err != nil {
		log.Printf("Error getting stored searches: %v", err)
		return
	}

	time.Sleep(1 * time.Second)

	fmt.Println("\n\n7. Final stored searches after extension:")
	finalStored, _ := logger.GetStoredSearches()
	for _, word := range finalStored {
		fmt.Printf("   - %s\n", word)
	}

	fmt.Println("\n=== Demo Complete ===")
}
