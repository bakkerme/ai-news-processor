package main

import (
	"os"
)

func returnFakeRSS() string {
	// Assuming localllama.rss is located in the same directory as this file
	b, err := os.ReadFile("localllama.rss")
	if err != nil {
		panic(err)
	}
	return string(b)
}
