package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/andres10976/SISAP-PoC/backend/internal/model"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/ctlog"
	"github.com/andres10976/SISAP-PoC/backend/internal/service/matcher"
)

func main() {
	// Expanded keywords: brands, cloud providers, CDN, common patterns
	testKeywords := []model.Keyword{
		// Major brands
		{ID: 1, Value: "amazon"},
		{ID: 2, Value: "google"},
		{ID: 3, Value: "microsoft"},
		{ID: 4, Value: "apple"},
		{ID: 5, Value: "facebook"},

		// Cloud providers & infrastructure
		{ID: 6, Value: "cloudflare"},
		{ID: 7, Value: "azure"},
		{ID: 8, Value: "digitalocean"},
		{ID: 9, Value: "heroku"},
		{ID: 10, Value: "vercel"},
		{ID: 11, Value: "netlify"},
		{ID: 12, Value: "cloudfront"},
		{ID: 13, Value: "akamai"},
		{ID: 14, Value: "fastly"},

		// Common domain patterns
		{ID: 15, Value: "api"},
		{ID: 16, Value: "app"},
		{ID: 17, Value: "web"},
		{ID: 18, Value: "www"},
		{ID: 19, Value: "mail"},
		{ID: 20, Value: "dev"},
		{ID: 21, Value: "test"},
		{ID: 22, Value: "staging"},
		{ID: 23, Value: "prod"},
		{ID: 24, Value: "admin"},

		// Generic cloud/hosting terms
		{ID: 25, Value: "cloud"},
		{ID: 26, Value: "cdn"},
		{ID: 27, Value: "server"},
		{ID: 28, Value: "host"},
		{ID: 29, Value: "vpn"},
		{ID: 30, Value: "ssl"},
	}

	ctx := context.Background()
	client := ctlog.NewClient("https://ct.cloudflare.com/logs/nimbus2027/")

	// Get current tree size
	sth, err := client.GetSTH(ctx)
	if err != nil {
		log.Fatalf("Failed to get STH: %v", err)
	}

	fmt.Printf("Tree size: %d\n", sth.TreeSize)

	// Fetch the most recent 100 entries
	start := sth.TreeSize - 100
	end := sth.TreeSize - 1

	fmt.Printf("Fetching entries %d to %d...\n", start, end)
	entries, err := client.GetEntries(ctx, start, end)
	if err != nil {
		log.Fatalf("Failed to fetch entries: %v", err)
	}

	fmt.Printf("Fetched %d entries\n\n", len(entries))

	// Track matches per keyword
	keywordMatches := make(map[string][]string)
	parseErrors := 0

	// Process each entry
	for i, entry := range entries {
		cert, err := ctlog.ParseLeafInput(entry.LeafInput, entry.ExtraData)
		if err != nil {
			parseErrors++
			continue
		}

		// Check against all keywords
		matches := matcher.Match(cert, testKeywords)
		for _, match := range matches {
			// Find keyword name
			var kwName string
			for _, kw := range testKeywords {
				if kw.ID == match.KeywordID {
					kwName = kw.Value
					break
				}
			}

			// Store the matched domain
			keywordMatches[kwName] = append(keywordMatches[kwName], match.MatchedDomain)
		}

		// Show progress
		if (i+1)%25 == 0 {
			fmt.Printf("Processed %d/%d entries...\n", i+1, len(entries))
		}
	}

	fmt.Printf("\n=== RESULTS ===\n")
	fmt.Printf("Parse errors: %d\n", parseErrors)
	fmt.Printf("Successfully parsed: %d\n\n", len(entries)-parseErrors)

	// Show keywords with matches
	matchedKeywords := make([]string, 0)
	for kw, domains := range keywordMatches {
		matchedKeywords = append(matchedKeywords, kw)
		fmt.Printf("✓ %s: %d matches\n", strings.ToUpper(kw), len(domains))
		// Show first 5 matched domains as examples
		for i, domain := range domains {
			if i >= 5 {
				fmt.Printf("  ... and %d more\n", len(domains)-5)
				break
			}
			fmt.Printf("  - %s\n", domain)
		}
		fmt.Println()
	}

	// Show keywords with NO matches
	fmt.Printf("Keywords with NO matches:\n")
	for _, kw := range testKeywords {
		if _, found := keywordMatches[kw.Value]; !found {
			fmt.Printf("✗ %s\n", kw.Value)
		}
	}
}
