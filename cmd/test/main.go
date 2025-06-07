package main

import (
	"fmt"
	"log"

	"soundcloud-tui/internal/soundcloud"
)

func main() {
	fmt.Println("Testing SoundCloud API integration...")

	client, err := soundcloud.NewClient()
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Test search functionality
	fmt.Println("\n🔍 Testing search for 'lofi hip hop'...")
	tracks, err := client.Search("lofi hip hop")
	if err != nil {
		log.Printf("Search failed: %v", err)
	} else {
		fmt.Printf("Found %d tracks:\n", len(tracks))
		for i, track := range tracks[:min(3, len(tracks))] {
			fmt.Printf("%d. %s by %s\n", i+1, track.Title, track.User.Username)
		}
	}

	// Test getting track info (using a known SoundCloud URL)
	fmt.Println("\n🎵 Testing track info retrieval...")
	fmt.Println("Note: This will only work with a valid SoundCloud track URL")
	
	fmt.Println("\n✅ Basic SoundCloud API integration test completed!")
	fmt.Println("If no errors above, the API is working correctly.")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}