package main

import (
	"flag"
	"log"
	"sync"
	"time"

	"github.com/painhardcore/SageSlayerServer/internal/client"
)

func main() {
	serverAddr := flag.String("server-addr", "localhost:8000", "Server address")
	attack := flag.Bool("attack", false, "Enable attack mode to simulate constant requests")
	interval := flag.Duration("interval", 0, "Interval between requests in attack mode (e.g., 10s, 500ms)")
	concurrency := flag.Int("concurrency", 1, "Number of concurrent clients in attack mode")
	silent := flag.Bool("silent", false, "Disable output (printing quotes)")
	flag.Parse()

	c := client.NewClient(*serverAddr)
	var wg sync.WaitGroup

	if *attack {
		// Shared counter for statistics
		var quoteCount int64
		var mu sync.Mutex

		// Start time for calculating quotes per second
		startTime := time.Now()

		// Ticker for displaying statistics
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for i := 0; i < *concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for {
					err := c.RequestQuote(*silent)
					if err != nil {
						log.Printf("Client %d: Error: %v", id, err)
					} else {
						mu.Lock()
						quoteCount++
						mu.Unlock()
					}

					if *interval > 0 {
						time.Sleep(*interval)
					}
				}
			}(i)
		}

		// Display statistics
		go func() {
			for range ticker.C {
				mu.Lock()
				elapsed := time.Since(startTime).Seconds()
				qps := float64(quoteCount) / elapsed
				log.Printf("Quotes received: %d, Quotes per second: %.2f", quoteCount, qps)
				mu.Unlock()
			}
		}()

		wg.Wait()
	} else {
		// Single request mode
		err := c.RequestQuote(*silent)
		if err != nil {
			log.Printf("Error: %v", err)
		}
	}
}
