package main

import (
	"flag"
	"log"
	"math"

	"github.com/painhardcore/SageSlayerServer/internal/server"
)

func main() {
	addr := flag.String("addr", ":8000", "TCP address to listen on")
	flag.Parse()

	// Define the difficulty function
	difficultyFunc := func(requestCount float64) int {
		if requestCount <= 220 {
			// Increase 1 level every 10 requests up to difficulty level 22
			difficulty := int(math.Ceil(requestCount / 10))
			if difficulty < 1 {
				difficulty = 1
			}
			return difficulty
		} else {
			// For request counts above 220, increase 1 level per 100 requests
			additionalDifficulty := int((requestCount - 220) / 100)
			difficulty := 22 + additionalDifficulty
			return difficulty
		}
	}

	srv := server.NewServer(*addr, 60.0, difficultyFunc)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
