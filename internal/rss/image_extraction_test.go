package rss

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractImageURLs(t *testing.T) {
	tests := []struct {
		name           string
		entryID        string
		entryContent   string
		expectedImages []string
	}{
		{
			name:    "Direct image link in [link] element",
			entryID: "test_direct_image_link",
			entryContent: `<table> <tr><td> <a href="https://www.reddit.com/r/LocalLLaMA/comments/1k05wpt/bytedance_releases_liquid_model_family_of/">
				<img src="https://preview.redd.it/393vjiodz2ve1.jpeg?width=640&amp;crop=smart&amp;auto=webp&amp;s=afb315c5ae73bc479aead0533e99e06cf2db069a" alt="ByteDance releases" />
				</a> </td><td> <!-- SC_OFF --><div class="md"></div><!-- SC_ON --> &amp;#32; submitted by &amp;#32; 
				<a href="https://www.reddit.com/user/ResearchCrafty1804"> /u/ResearchCrafty1804 </a> <br/> 
				<span><a href="https://i.redd.it/393vjiodz2ve1.jpeg">[link]</a></span> &amp;#32; 
				<span><a href="https://www.reddit.com/r/LocalLLaMA/comments/1k05wpt/bytedance_releases_liquid_model_family_of/">[comments]</a></span> </td></tr></table>`,
			expectedImages: []string{
				"https://i.redd.it/393vjiodz2ve1.jpeg",
			},
		},
		{
			name:    "Image in img tag",
			entryID: "test_img_tag",
			entryContent: `<table> <tr><td> <a href="https://www.reddit.com/r/LocalLLaMA/comments/1k0tkca/massive_5000_tokens_per_second_on_2x3090/">
				<img src="https://b.thumbs.redditmedia.com/Kqc4r4j1pvS-lOt8Ugi0fd-cS_ZlQgpSRkB-O5FUESc.jpg" alt="Massive 5000 tokens per second" />
				</a> </td><td> <!-- SC_OFF --><div class="md"><p>For research purposes I need to process huge amounts of data.</p></div><!-- SC_ON --> &amp;#32;</td></tr></table>`,
			expectedImages: []string{}, // Should be empty since it contains "thumbs" in the URL
		},
		{
			name:    "Multiple images",
			entryID: "test_multiple_images",
			entryContent: `<div><p><a href="https://preview.redd.it/66ifgifkr5ve1.png?width=2756&format=png&auto=webp&s=77650cfe31229f9bde35da3e569cef3d5caa885f">
				https://preview.redd.it/66ifgifkr5ve1.png?width=2756&format=png&auto=webp&s=77650cfe31229f9bde35da3e569cef3d5caa885f</a></p>
				<p><img src="https://example.com/image.jpg" /></p>
				<span><a href="https://i.imgur.com/py5Tvae.png">[link]</a></span></div>`,
			expectedImages: []string{
				"https://example.com/image.jpg",
				"https://i.imgur.com/py5Tvae.png",
			},
		},
		{
			name:         "i.redd.it link without image extension",
			entryID:      "test_iredd_it",
			entryContent: `<div><span><a href="https://i.redd.it/someimage">[link]</a></span></div>`,
			expectedImages: []string{
				"https://i.redd.it/someimage",
			},
		},
		{
			name:    "Duplicate image URLs are deduplicated",
			entryID: "test_duplicate_urls",
			entryContent: `<div>
				<img src="https://example.com/image.jpg" />
				<span><a href="https://example.com/image.jpg">[link]</a></span>
				<a href="https://example.com/image.jpg">Same image</a>
			</div>`,
			expectedImages: []string{
				"https://example.com/image.jpg",
			},
		},
		{
			name:           "Empty content",
			entryID:        "test_empty_content",
			entryContent:   "",
			expectedImages: []string{},
		},
		{
			name:           "No images in content",
			entryID:        "test_no_images",
			entryContent:   `<div><p>This is a text-only post with no images.</p></div>`,
			expectedImages: []string{},
		},
		{
			name:         "Image in anchor tag",
			entryID:      "test_anchor_tag",
			entryContent: `<div><a href="https://example.com/image.png">Click to see image</a></div>`,
			expectedImages: []string{
				"https://example.com/image.png",
			},
		},
		{
			name:    "URLs with different protocols",
			entryID: "test_protocols",
			entryContent: `<div>
				<img src="http://example.com/image1.jpg" />
				<img src="https://example.com/image2.jpg" />
				<img src="data:image/jpeg;base64,xyz" />
			</div>`,
			expectedImages: []string{
				"http://example.com/image1.jpg",
				"https://example.com/image2.jpg",
				// data: URLs should be excluded as they're not external URLs
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			entry := Entry{
				ID:      tc.entryID,
				Content: tc.entryContent,
			}

			err := entry.ExtractImageURLs()
			assert.NoError(t, err)

			// Convert the URLs back to strings for easy comparison
			var actualImages []string
			for _, imgURL := range entry.ImageURLs {
				actualImages = append(actualImages, imgURL.String())
			}

			assert.ElementsMatch(t, tc.expectedImages, actualImages)
		})
	}
}

func TestIsLikelyImageURL(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://i.redd.it/somefile.jpg", true},
		{"https://i.redd.it/noextension", true}, // should be true because i.redd.it is an image host
		{"https://i.imgur.com/abcdef", true},    // should be true because i.imgur.com is an image host
		{"https://example.com/image.png", true},
		{"https://example.com/image.jpg", true},
		{"https://example.com/image.jpeg", true},
		{"https://example.com/image.gif", true},
		{"https://example.com/image.webp", true},
		{"https://example.com/document.pdf", false},
		{"https://example.com/noextension", false},
		{"https://example.com/thumbnail.jpg", true}, // Still an image despite "thumbnail"
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := isLikelyImageURL(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestContainsExcludedTerms(t *testing.T) {
	tests := []struct {
		url      string
		expected bool
	}{
		{"https://example.com/thumbnail.jpg", true},
		{"https://example.com/preview-image.png", true},
		{"https://thumbs.example.com/image.jpg", true},
		{"https://example.com/preview/image.jpg", true},
		{"https://example.com/normal-image.jpg", false},
		{"https://example.com/image.png", false},
	}

	for _, tc := range tests {
		t.Run(tc.url, func(t *testing.T) {
			result := containsExcludedTerms(tc.url)
			assert.Equal(t, tc.expected, result)
		})
	}
}
func TestURLDeduplication(t *testing.T) {
	// Test the deduplication mechanism in ExtractImageURLs
	entry := Entry{
		ID: "deduplication_test",
		Content: `<div>
			<img src="https://example.com/image.jpg" />
			<img src="https://example.com/image.jpg" /> <!-- Same URL repeated -->
			<a href="https://example.com/image.jpg">Same URL in link</a>
			<img src="https://example.com/another-image.png" />
		</div>`,
	}

	err := entry.ExtractImageURLs()
	assert.NoError(t, err)

	// Should only have 2 distinct URLs despite having 3 image references
	assert.Len(t, entry.ImageURLs, 2)

	// Convert to strings for easier comparison
	urlStrings := make([]string, 0, len(entry.ImageURLs))
	for _, u := range entry.ImageURLs {
		urlStrings = append(urlStrings, u.String())
	}

	// Verify the expected URLs are present
	assert.Contains(t, urlStrings, "https://example.com/image.jpg")
	assert.Contains(t, urlStrings, "https://example.com/another-image.png")
}

func TestAddImageURLIfValid(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		expectedValid  bool
		expectedURLStr string
	}{
		{
			name:           "Valid image URL",
			url:            "https://example.com/image.jpg",
			expectedValid:  true,
			expectedURLStr: "https://example.com/image.jpg",
		},
		{
			name:          "URL with excluded term",
			url:           "https://example.com/thumbnail.jpg",
			expectedValid: false,
		},
		{
			name:          "Non-image URL",
			url:           "https://example.com/document.pdf",
			expectedValid: false,
		},
		{
			name:           "Image URL on known image host",
			url:            "https://i.imgur.com/abc123",
			expectedValid:  true,
			expectedURLStr: "https://i.imgur.com/abc123",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a map to store URLs
			urlMap := make(map[string]url.URL)

			// Call the function under test
			addImageURLIfValid(tc.url, urlMap)

			// Check if the URL was added as expected
			if tc.expectedValid {
				assert.Len(t, urlMap, 1)

				// Parse the expected URL for comparison
				expectedURL, err := url.Parse(tc.expectedURLStr)
				assert.NoError(t, err)

				// Check if the map contains our URL
				_, exists := urlMap[expectedURL.String()]
				assert.True(t, exists)
			} else {
				assert.Empty(t, urlMap)
			}
		})
	}
}

func TestEnsureValidImageURL(t *testing.T) {
	tests := []struct {
		name     string
		urlStr   string
		expected string
	}{
		{
			name:     "Already valid URL with HTTPS",
			urlStr:   "https://example.com/image.jpg",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "Already valid URL with HTTP",
			urlStr:   "http://example.com/image.jpg",
			expected: "http://example.com/image.jpg",
		},
		{
			name:     "URL without scheme",
			urlStr:   "example.com/image.jpg",
			expected: "https://example.com/image.jpg",
		},
		{
			name:     "URL with just domain",
			urlStr:   "i.imgur.com",
			expected: "https://i.imgur.com",
		},
		{
			name:     "URL with subdomain but no scheme",
			urlStr:   "imgs.example.com/path/to/image.png",
			expected: "https://imgs.example.com/path/to/image.png",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := ensureValidImageURL(tc.urlStr)
			assert.Equal(t, tc.expected, result)
		})
	}
}
