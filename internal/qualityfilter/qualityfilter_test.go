package qualityfilter

import (
	"testing"

	"github.com/bakkerme/ai-news-processor/internal/feeds"
)

func TestFilter(t *testing.T) {
	tests := []struct {
		name           string
		entries        []feeds.Entry
		threshold      int
		expectedLength int
		expectedTitles []string
	}{
		{
			name: "filters entries below threshold",
			entries: []feeds.Entry{
				{Title: "Entry1", Comments: make([]feeds.EntryComments, 5)},
				{Title: "Entry2", Comments: make([]feeds.EntryComments, 15)},
				{Title: "Entry3", Comments: make([]feeds.EntryComments, 8)},
			},
			threshold:      10,
			expectedLength: 1,
			expectedTitles: []string{"Entry2"},
		},
		{
			name: "custom threshold",
			entries: []feeds.Entry{
				{Title: "Entry1", Comments: make([]feeds.EntryComments, 5)},
				{Title: "Entry2", Comments: make([]feeds.EntryComments, 15)},
				{Title: "Entry3", Comments: make([]feeds.EntryComments, 8)},
			},
			threshold:      7,
			expectedLength: 2,
			expectedTitles: []string{"Entry2", "Entry3"},
		},
		{
			name: "no entries above threshold",
			entries: []feeds.Entry{
				{Title: "Entry1", Comments: make([]feeds.EntryComments, 5)},
				{Title: "Entry2", Comments: make([]feeds.EntryComments, 3)},
			},
			threshold:      10,
			expectedLength: 0,
			expectedTitles: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := Filter(tt.entries, tt.threshold)

			if len(filtered) != tt.expectedLength {
				t.Errorf("expected %d entries, got %d", tt.expectedLength, len(filtered))
			}

			for i, title := range tt.expectedTitles {
				if filtered[i].Title != title {
					t.Errorf("expected entry %d to have title %s, got %s", i, title, filtered[i].Title)
				}
			}
		})
	}
}
