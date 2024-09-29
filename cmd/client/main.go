package main

import (
	"flag"
	"log"
	"time"

	"github.com/painhardcore/SageSlayerServer/internal/client"
)

func main() {
	serverAddr := flag.String("server-addr", "localhost:8000", "Server address")
	attack := flag.Bool("attack", false, "Enable attack mode to simulate constant request")
	interval := flag.Duration("interval", 0, "Interval between requests in attack mode (e.g., 10s, 500ms)")
	flag.Parse()

	c := client.NewClient(*serverAddr)

	for {
		err := c.RequestQuote()
		if err != nil {
			log.Printf("Error: %v", err)
		}

		if !*attack {
			break
		}

		if *interval > 0 {
			time.Sleep(*interval)
		}
	}
}
