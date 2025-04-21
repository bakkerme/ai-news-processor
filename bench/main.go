package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Item struct {
	Title         string `json:"Title"`
	ID            string `json:"ID"`
	ShouldInclude bool   `json:"ShouldInclude"`
}

func main() {
	// Read benchmark.json
	benchmarkData, err := os.ReadFile("benchmark.json")
	if err != nil {
		fmt.Printf("Error reading benchmark.json: %v\n", err)
		os.Exit(1)
	}

	var benchmarkItems []Item
	err = json.Unmarshal(benchmarkData, &benchmarkItems)
	if err != nil {
		fmt.Printf("Error unmarshaling benchmark.json: %v\n", err)
		os.Exit(1)
	}

	// Read compare.json
	compareData, err := os.ReadFile("compare.json")
	if err != nil {
		fmt.Printf("Error reading compare.json: %v\n", err)
		os.Exit(1)
	}

	var compareItems []Item
	err = json.Unmarshal(compareData, &compareItems)
	if err != nil {
		fmt.Printf("Error unmarshaling compare.json: %v\n", err)
		os.Exit(1)
	}

	// Create a map of benchmark items for quick lookup
	benchmarkMap := make(map[string]Item)
	for _, item := range benchmarkItems {
		benchmarkMap[item.ID] = item
	}

	// Initialize counters and violation lists
	var (
		rule1Count int
		rule2Count int
		rule3Count int
		rule4Count int
		totalScore int

		rule3Violations []Item
		rule4Violations []Item
	)

	// Check each item in compare.json
	for _, compareItem := range compareItems {
		benchmarkItem, exists := benchmarkMap[compareItem.ID]

		if !compareItem.ShouldInclude {
			if !exists {
				// Rule 1: +2 points
				rule1Count++
				totalScore += 2
			} else if !benchmarkItem.ShouldInclude {
				// Rule 2: +1 point
				rule2Count++
				totalScore += 1
			} else {
				// Rule 3: -1 point
				rule3Count++
				totalScore -= 1
				rule3Violations = append(rule3Violations, compareItem)
			}
		} else if !exists {
			// Rule 4: -2 points
			rule4Count++
			totalScore -= 2
			rule4Violations = append(rule4Violations, compareItem)
		}
	}

	// Print results
	fmt.Printf("Number of items in benchmark.json: %d\n", len(benchmarkItems))
	fmt.Printf("Number of items in compare.json: %d\n", len(compareItems))
	fmt.Println("\nScoring Rules Applied:")
	fmt.Printf("Rule 1 (+2): compare=false and not in benchmark: %d\n", rule1Count)
	fmt.Printf("Rule 2 (+1): compare=false and benchmark=false: %d\n", rule2Count)
	fmt.Printf("Rule 3 (-1): compare=false and benchmark=true: %d\n", rule3Count)
	fmt.Printf("Rule 4 (-2): compare=true and not in benchmark: %d\n", rule4Count)
	fmt.Printf("\nFinal Tally Score: %d\n", totalScore)

	// Print Rule 3 violations if any
	if len(rule3Violations) > 0 {
		fmt.Println("\nRule 3 Violations (compare=false but benchmark=true):")
		for _, item := range rule3Violations {
			fmt.Printf("- ID: %s, Title: %q\n", item.ID, item.Title)
		}
	}

	// Print Rule 4 violations if any
	if len(rule4Violations) > 0 {
		fmt.Println("\nRule 4 Violations (compare=true but not in benchmark):")
		for _, item := range rule4Violations {
			fmt.Printf("- ID: %s, Title: %q\n", item.ID, item.Title)
		}
	}
}
