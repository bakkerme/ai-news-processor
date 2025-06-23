package models

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestItem_ToSummaryString(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		expected string
	}{
		{
			name: "Complete item with all fields",
			item: Item{
				ID:             "test-id-123",
				Title:          "Test Article Title",
				Summary:        "This is a test summary of the article content.",
				CommentSummary: "This is a summary of the comments on this article.",
			},
			expected: "ID: test-id-123\nTitle: Test Article Title\nSummary: This is a test summary of the article content.\nComment Summary: This is a summary of the comments on this article.\n",
		},
		{
			name: "Item without comment summary",
			item: Item{
				ID:      "test-id-456",
				Title:   "Another Test Article",
				Summary: "Another test summary with important information.",
			},
			expected: "ID: test-id-456\nTitle: Another Test Article\nSummary: Another test summary with important information.\n",
		},
		{
			name: "Item with empty comment summary",
			item: Item{
				ID:             "test-id-789",
				Title:          "Article with Empty Comment Summary",
				Summary:        "Summary of an article that has no comments.",
				CommentSummary: "",
			},
			expected: "ID: test-id-789\nTitle: Article with Empty Comment Summary\nSummary: Summary of an article that has no comments.\n",
		},
		{
			name: "Item with empty fields",
			item: Item{
				ID:      "",
				Title:   "",
				Summary: "",
			},
			expected: "ID: \nTitle: \nSummary: \n",
		},
		{
			name: "Item with special characters",
			item: Item{
				ID:             "special-chars-123",
				Title:          "Article with \"Quotes\" & Special Characters",
				Summary:        "Summary with newlines\nand\ttabs.",
				CommentSummary: "Comments with émojis 🚀 and unicode ñ characters.",
			},
			expected: "ID: special-chars-123\nTitle: Article with \"Quotes\" & Special Characters\nSummary: Summary with newlines\nand\ttabs.\nComment Summary: Comments with émojis 🚀 and unicode ñ characters.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.ToSummaryString()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestItem_ToSummaryString_StructuralValidation(t *testing.T) {
	item := Item{
		ID:             "struct-test",
		Title:          "Structural Test",
		Summary:        "Testing the structure",
		CommentSummary: "Comment content",
		// Other fields should be ignored
		Overview:          "This should not appear",
		ImageSummary:      "This should not appear",
		WebContentSummary: "This should not appear",
		Link:              "https://example.com",
		IsRelevant:        true,
		ThumbnailURL:      "https://example.com/thumb.jpg",
	}

	result := item.ToSummaryString()

	// Check that only the expected fields are included
	assert.Contains(t, result, "ID: struct-test")
	assert.Contains(t, result, "Title: Structural Test")
	assert.Contains(t, result, "Summary: Testing the structure")
	assert.Contains(t, result, "Comment Summary: Comment content")

	// Check that other fields are NOT included
	assert.NotContains(t, result, "Overview")
	assert.NotContains(t, result, "This should not appear")
	assert.NotContains(t, result, "https://example.com")
	assert.NotContains(t, result, "IsRelevant")
	assert.NotContains(t, result, "ThumbnailURL")
}

func TestItem_ToSummaryString_LineFormat(t *testing.T) {
	item := Item{
		ID:             "format-test",
		Title:          "Format Test",
		Summary:        "Testing format",
		CommentSummary: "Comment format",
	}

	result := item.ToSummaryString()
	lines := strings.Split(result, "\n")

	// Should have 5 lines: ID, Title, Summary, Comment Summary, and final empty line
	assert.Len(t, lines, 5)
	assert.Equal(t, "ID: format-test", lines[0])
	assert.Equal(t, "Title: Format Test", lines[1])
	assert.Equal(t, "Summary: Testing format", lines[2])
	assert.Equal(t, "Comment Summary: Comment format", lines[3])
	assert.Equal(t, "", lines[4]) // Final empty line due to trailing \n
}

func TestItem_ToSummaryString_WithoutCommentSummary_LineFormat(t *testing.T) {
	item := Item{
		ID:      "no-comment-test",
		Title:   "No Comment Test",
		Summary: "Testing without comments",
	}

	result := item.ToSummaryString()
	lines := strings.Split(result, "\n")

	// Should have 4 lines: ID, Title, Summary, and final empty line (no Comment Summary line)
	assert.Len(t, lines, 4)
	assert.Equal(t, "ID: no-comment-test", lines[0])
	assert.Equal(t, "Title: No Comment Test", lines[1])
	assert.Equal(t, "Summary: Testing without comments", lines[2])
	assert.Equal(t, "", lines[3]) // Final empty line due to trailing \n

	// Should not contain "Comment Summary" anywhere
	assert.NotContains(t, result, "Comment Summary")
}