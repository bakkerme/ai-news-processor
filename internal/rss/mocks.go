package rss

import (
	"fmt"
	"os"
	"path/filepath"
)

func ReturnFakeRSS(personaName string) *Feed {
	path := filepath.Join("..", "feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", personaName))
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	rssFeed := &Feed{}
	err = ProcessRSSFeed(string(b), rssFeed)
	if err != nil {
		panic(err)
	}

	return rssFeed
}

func ReturnFakeCommentRSS(personaName, id string) *CommentFeed {
	path := filepath.Join("..", "feed_mocks", "rss", personaName, fmt.Sprintf("%s.rss", id))
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}

	commentFeed := &CommentFeed{}
	err = ProcessCommentsRSSFeed(string(b), commentFeed)
	if err != nil {
		panic(err)
	}

	return commentFeed
}
