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

func TestExtractLinkURLsFromContent(t *testing.T) {
	content := `<div>
		<span><a href="https://i.redd.it/image1.jpg">[link]</a></span>
		<span><a href="https://i.imgur.com/image2.png">[link]</a></span>
		Some other content
		<span><a href="https://example.com/notlink">Not a link</a></span>
	</div>`

	expected := []string{
		"https://i.redd.it/image1.jpg",
		"https://i.imgur.com/image2.png",
	}

	result := extractLinkURLsFromContent(content)
	assert.ElementsMatch(t, expected, result)
}

func TestDeduplicateURLs(t *testing.T) {
	// Create duplicate URLs
	url1, _ := url.Parse("https://example.com/image1.jpg")
	url2, _ := url.Parse("https://example.com/image2.jpg")
	url3, _ := url.Parse("https://example.com/image1.jpg") // Duplicate of url1

	urls := []url.URL{*url1, *url2, *url3}

	deduplicated := deduplicateURLs(urls)

	// Should only have 2 unique URLs
	assert.Len(t, deduplicated, 2)

	// Convert back to strings for easier comparison
	var deduplicatedStrings []string
	for _, u := range deduplicated {
		deduplicatedStrings = append(deduplicatedStrings, u.String())
	}

	assert.Contains(t, deduplicatedStrings, "https://example.com/image1.jpg")
	assert.Contains(t, deduplicatedStrings, "https://example.com/image2.jpg")
}
