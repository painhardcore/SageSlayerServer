package quotes

import (
	_ "embed"
	"strings"

	"pgregory.net/rand"
)

//go:embed quotes.txt
var quotesText string

var quotes []string

var r *rand.Rand

func init() {
	r = rand.New()
	rawQuotes := strings.Split(quotesText, "\n")
	for _, quote := range rawQuotes {
		quote = strings.TrimSpace(quote)
		if quote != "" {
			quotes = append(quotes, quote)
		}
	}
}

// GetRandomQuote returns a random quote from the list
func GetRandomQuote() string {
	if len(quotes) == 0 {
		return "No quotes available."
	}
	index := r.Intn(len(quotes))
	return quotes[index]
}
