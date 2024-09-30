package main

import (
	"flag"
	"log"
	"math"
	"os"
	"os/signal"
	"runtime/pprof"
	"syscall"

	"github.com/painhardcore/SageSlayerServer/internal/server"
)

func main() {
	addr := flag.String("addr", ":8000", "TCP address to listen on")
	cpuProfile := flag.String("cpuProfile", "cpu.prof", "File to store CPU profile")
	memProfile := flag.String("memProfile", "mem.prof", "File to store memory profile")
	goroutineProfile := flag.String("goroutineProfile", "goroutine.prof", "File to store goroutine profile")
	flag.Parse()

	// Start CPU profiling
	cpuProfFile, err := os.Create(*cpuProfile)
	if err != nil {
		log.Fatalf("could not create CPU profile: %v", err)
	}
	defer cpuProfFile.Close()
	if err := pprof.StartCPUProfile(cpuProfFile); err != nil {
		log.Fatalf("could not start CPU profile: %v", err)
	}
	defer pprof.StopCPUProfile()

	// Handle termination signals to stop profiling gracefully
	exitChan := make(chan os.Signal, 1)
	signal.Notify(exitChan, syscall.SIGINT, syscall.SIGTERM)

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

	go func() {
		if err := srv.Start(); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for termination signal
	<-exitChan
	log.Println("Shutting down server...")

	// Stop CPU profiling (deferred)

	// Write memory profile
	memProfFile, err := os.Create(*memProfile)
	if err != nil {
		log.Fatalf("could not create memory profile: %v", err)
	}
	defer memProfFile.Close()
	if err := pprof.WriteHeapProfile(memProfFile); err != nil {
		log.Fatalf("could not write memory profile: %v", err)
	}

	// Write goroutine profile
	goroutineProfFile, err := os.Create(*goroutineProfile)
	if err != nil {
		log.Fatalf("could not create goroutine profile: %v", err)
	}
	defer goroutineProfFile.Close()
	if err := pprof.Lookup("goroutine").WriteTo(goroutineProfFile, 0); err != nil {
		log.Fatalf("could not write goroutine profile: %v", err)
	}
}
