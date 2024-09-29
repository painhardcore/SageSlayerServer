package main

import (
	"fmt"
	"log"
	"time"

	"github.com/painhardcore/SageSlayerServer/pkg/network"
	"github.com/painhardcore/SageSlayerServer/pkg/pow"
)

func main() {
	startDifficulty := 1
	maxDifficulty := 100
	incrementStep := 1

	for difficulty := startDifficulty; difficulty <= maxDifficulty; difficulty += incrementStep {
		fmt.Printf("Testing difficulty level: %d\n", difficulty)
		for range 5 {
			// Generate a new PoW challenge with the current difficulty
			challenge, err := pow.GenerateChallenge(difficulty)
			if err != nil {
				log.Fatalf("Error generating challenge at difficulty %d: %v", difficulty, err)
			}
			challengeProto := &network.Challenge{
				Qx:         challenge.Qx.Bytes(),
				Qy:         challenge.Qy.Bytes(),
				Curve:      "P-256",
				Difficulty: int32(challenge.Difficulty),
			}

			// Measure time to solve the challenge
			startTime := time.Now()
			nonce, err := pow.SolveChallenge(challengeProto)
			if err != nil {
				log.Fatal(err)
			}
			duration := time.Since(startTime)

			// Print the time it took to solve the challenge
			fmt.Printf("Solved challenge at difficulty %d in %s\n", difficulty, duration)

			// Verify the solution
			verificationErr := pow.VerifySolution(challenge, nonce)
			if verificationErr != nil {
				log.Fatalf("Failed to verify solution at difficulty %d: %v", difficulty, verificationErr)
			}
		}
		// Print a message indicating successful verification
		fmt.Printf("Solution verified successfully at difficulty %d\n\n", difficulty)
	}
}
