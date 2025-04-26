package rss

import (
	"fmt"
	"os"
)

func ReturnFakeRSS() string {
	b, err := os.ReadFile("./rss/mocks/localllama.rss")
	if err != nil {
		panic(err)
	}
	return string(b)
}

func ReturnFakeCommentRSS(id string) string {
	b, err := os.ReadFile(fmt.Sprintf("./rss/mocks/%s.rss", id))
	if err != nil {
		panic(err)
	}
	return string(b)
}
