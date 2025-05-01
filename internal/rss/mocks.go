package rss

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReturnFakeRSS(personaName string) string {
	path := filepath.Join("..", "feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", personaName))
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}

func ReturnFakeCommentRSS(personaName, id string) string {
	path := filepath.Join("..", "feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", id))
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	return string(b)
}
