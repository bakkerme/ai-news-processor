package rss

import (
	"os"
)

func ReturnFakeRSS() string {
	// Assuming localllama.rss is located in the same directory as this file
	b, err := os.ReadFile("./rss/localllama.rss")
	if err != nil {
		panic(err)
	}
	return string(b)
}
