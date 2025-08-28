package main

import (
	"fmt"
	"log"
	"time"
)

func main() {
	fmt.Println("=== Search Logger V2 Demo ===")

	// Create Version 2 logger
	logger, err := NewSearchLoggerV2()
	if err != nil {
		log.Fatal("Failed to create SearchLoggerV2:", err)
	}
	defer logger.Close()

	// Create user identifier generator for demo
	idGen := NewUserIdentifierGenerator()

	fmt.Println("\n=== Scenario 1: Logged-in User Progressive Typing in order ===")
	user1Identifier := idGen.GenerateUserID()
	fmt.Printf("Logged-in User 1 (%s) progressively typing 'Business':\n", user1Identifier)
	for i := 1; i <= len("Business"); i++ {
		partial := "Business"[:i]
		fmt.Printf("  Typing: '%s'", partial)
		if err := logger.LogSearchV2(user1Identifier, partial); err != nil {
			fmt.Printf(" - Error: %v\n", err)
		} else {
			fmt.Printf("\n")
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n=== Scenario 2: Anonymous User Progressive Typing in order ===")
	anon1Identifier := idGen.GenerateAnonID()
	fmt.Printf("Anonymous User 1 (%s) progressively typing 'business':\n", anon1Identifier)
	for i := 1; i <= len("business"); i++ {
		partial := "business"[:i]
		fmt.Printf("  Typing: '%s'", partial)
		if err := logger.LogSearchV2(anon1Identifier, partial); err != nil {
			fmt.Printf(" - Error: %v\n", err)
		} else {
			fmt.Printf("\n")
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n=== Scenario 3: A Third Users Same Words in order ===")
	user2Identifier := idGen.GenerateUserID()
	fmt.Printf("Logged-in User 2 (%s) progressively typing 'business':\n", user2Identifier)
	for i := 1; i <= len("business"); i++ {
		partial := "business"[:i]
		fmt.Printf("  Typing: '%s'", partial)
		if err := logger.LogSearchV2(user2Identifier, partial); err != nil {
			fmt.Printf(" - Error: %v\n", err)
		} else {
			fmt.Printf("\n")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Note, the user will always type in-order query;
	// But the query may arrive backend server out of order in a distributed system.
	fmt.Println("\n=== Scenario 4: User 3 Out-of-Order query ===")
	user3Identifier := idGen.GenerateUserID()
	fmt.Printf("Logged-in User 3 (%s): out of order 'business':\n", user3Identifier)
	for i := len("business"); i >= 1; i-- {
		partial := "business"[:i]
		fmt.Printf("  Typing: '%s'", partial)
		if err := logger.LogSearchV2(user3Identifier, partial); err != nil {
			fmt.Printf(" - Error: %v\n", err)
		} else {
			fmt.Printf("\n")
		}
		time.Sleep(50 * time.Millisecond)
	}

	fmt.Println("\n=== Final Results: Per-User Deduplication ===")
	displayFinalResults(logger, user1Identifier, anon1Identifier, user2Identifier, user3Identifier)
}

func displayFinalResults(logger *SearchLoggerV2, user1Identifier, anon1Identifier, user2Identifier, user3Identifier string) {
	users := []struct {
		identifier string
		name       string
	}{
		{user1Identifier, "Logged-in User 1"},
		{anon1Identifier, "Anonymous User 1"},
		{user2Identifier, "Logged-in User 2"},
		{user3Identifier, "Logged-in User 3"},
	}

	totalRecords := 0
	fmt.Println("Final search results:")
	for _, user := range users {
		searches, err := logger.GetUserSearches(user.identifier)
		if err != nil {
			log.Printf("Error getting searches for %s: %v", user.name, err)
			continue
		}
		if len(searches) > 0 {
			totalRecords += len(searches)
			fmt.Printf("  %s (%s): %v\n", user.name, user.identifier, searches)
		}
	}
	fmt.Printf("Total unique search terms stored: %d\n", totalRecords)
}
