package urlextraction

import (
	"net/url"
	"testing"
)

// mockContentProvider implements ContentProvider for testing
type mockContentProvider struct {
	id      string
	content string
}

func (m mockContentProvider) GetID() string {
	return m.id
}

func (m mockContentProvider) GetContent() string {
	return m.content
}

// Helper function to compare slices regardless of order
func compareUnorderedStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aMap := make(map[string]int)
	for _, s := range a {
		aMap[s]++
	}
	for _, s := range b {
		if count, ok := aMap[s]; !ok || count == 0 {
			return false
		}
		aMap[s]--
	}
	return true
}

// Helper function to compare URL slices with string slices (converts URLs to strings)
func compareUnorderedURLSlices(a []url.URL, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Convert URLs to strings
	aStrings := make([]string, len(a))
	for i, u := range a {
		aStrings[i] = u.String()
	}
	return compareUnorderedStringSlices(aStrings, b)
}

// Helper function to compare URL slice maps with string slice maps (converts URLs to strings)
func compareURLSliceMaps(a map[string][]url.URL, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v1 := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}
		// Convert URLs to strings
		v1Strings := make([]string, len(v1))
		for i, u := range v1 {
			v1Strings[i] = u.String()
		}
		if !compareUnorderedStringSlices(v1Strings, v2) {
			return false
		}
	}
	return true
}

func TestRedditExtractor_extractURLsFromHTML(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name        string
		htmlContent string
		wantURLs    []string
		wantErr     bool
	}{
		{
			name:        "empty content",
			htmlContent: "",
			wantURLs:    []string{},
			wantErr:     false,
		},
		{
			name:        "no anchor tags",
			htmlContent: "<p>Some text here</p><div></div>",
			wantURLs:    []string{},
			wantErr:     false,
		},
		{
			name:        "single anchor tag",
			htmlContent: `<p><a href="http://example.com">Example</a></p>`,
			wantURLs:    []string{"http://example.com"},
			wantErr:     false,
		},
		{
			name:        "multiple anchor tags",
			htmlContent: `<a href="http://one.com">1</a> Some text <a href="https://two.org/path?q=v">2</a>`,
			wantURLs:    []string{"http://one.com", "https://two.org/path?q=v"},
			wantErr:     false,
		},
		{
			name:        "escaped HTML content with links",
			htmlContent: `&lt;a href=&quot;http://example.com&quot;&gt;Example&lt;/a&gt;`,
			wantURLs:    []string{"http://example.com"},
			wantErr:     false,
		},
		{
			name:        "anchor without href",
			htmlContent: `<a>No href</a><a href="">Empty href</a>`,
			wantURLs:    []string{},
			wantErr:     false,
		},
		{
			name:        "complex reddit-style HTML with mixed links",
			htmlContent: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/&quot;&gt; &lt;img src=&quot;placeholder.jpg&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Check this &lt;a href=&quot;http://external.example.com/page&quot;&gt;cool link&lt;/a&gt;!&lt;/p&gt; &lt;p&gt;And another one &lt;a href=&quot;https://another.example.org&quot;&gt;here&lt;/a&gt;.&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/Recurrents&quot;&gt; /u/Recurrents &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/gallery/1kexdgy&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			wantURLs: []string{
				"https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/",
				"http://external.example.com/page",
				"https://another.example.org",
				"https://www.reddit.com/user/Recurrents",
				"https://www.reddit.com/gallery/1kexdgy",
				"https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURLs, err := extractor.extractURLsFromHTML(tt.htmlContent)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareUnorderedStringSlices(gotURLs, tt.wantURLs) {
				gotStr := ""
				for _, u := range gotURLs {
					gotStr += "\n    " + u
				}
				wantStr := ""
				for _, u := range tt.wantURLs {
					wantStr += "\n    " + u
				}
				t.Errorf("[%s]\n  Got URLs:%s\n  Want URLs:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}

func TestRedditExtractor_isRedditDomain(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		url     string
		want    bool
		wantErr bool
	}{
		{
			name:    "reddit.com domain",
			url:     "https://reddit.com/r/news",
			want:    true,
			wantErr: false,
		},
		{
			name:    "redd.it domain",
			url:     "https://redd.it/short",
			want:    true,
			wantErr: false,
		},
		{
			name:    "external domain",
			url:     "https://example.com",
			want:    false,
			wantErr: false,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			want:    false,
			wantErr: true,
		},
		{
			name:    "case insensitive reddit.com",
			url:     "https://ReDdIt.CoM/path",
			want:    true,
			wantErr: false,
		},
		{
			name:    "case insensitive redd.it",
			url:     "https://ReDd.iT/path",
			want:    true,
			wantErr: false,
		},
		{
			name:    "reddit.com in path",
			url:     "https://example.com/reddit.com",
			want:    false,
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.isRedditDomain(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("[%s]\n  Got:  %v\n  Want: %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestRedditExtractor_ExtractAllURLsFromEntry(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entry   mockContentProvider
		want    []string
		wantErr bool
	}{
		{
			name: "no URLs in content",
			entry: mockContentProvider{
				id:      "1",
				content: "Just some text without links",
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "only reddit URLs",
			entry: mockContentProvider{
				id:      "2",
				content: `<a href="https://reddit.com/r/news">Reddit</a> <a href="https://redd.it/short">Short</a>`,
			},
			want:    []string{"https://reddit.com/r/news", "https://redd.it/short"},
			wantErr: false,
		},
		{
			name: "mix of reddit and external URLs",
			entry: mockContentProvider{
				id:      "3",
				content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/&quot;&gt; &lt;img src=&quot;placeholder.jpg&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Check this &lt;a href=&quot;http://external.example.com/page&quot;&gt;cool link&lt;/a&gt;!&lt;/p&gt; &lt;p&gt;And another one &lt;a href=&quot;https://another.example.org&quot;&gt;here&lt;/a&gt;.&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/Recurrents&quot;&gt; /u/Recurrents &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/gallery/1kexdgy&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			},
			want: []string{
				"https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/",
				"http://external.example.com/page",
				"https://another.example.org",
				"https://www.reddit.com/user/Recurrents",
				"https://www.reddit.com/gallery/1kexdgy",
				"https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/",
			},
			wantErr: false,
		},
		{
			name: "multiple external URLs with some reddit",
			entry: mockContentProvider{
				id:      "4",
				content: `&lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;First link: &lt;a href=&quot;https://example.com/first&quot;&gt;1&lt;/a&gt;. Second link: &lt;a href=&quot;https://test.org/second?param=true&quot;&gt;2&lt;/a&gt;. A reddit one: &lt;a href=&quot;https://www.reddit.com/r/subreddit/&quot;&gt;reddit link&lt;/a&gt;.&lt;/p&gt;&lt;/div&gt;&lt;!-- SC_ON --&gt;`,
			},
			want:    []string{"https://example.com/first", "https://test.org/second?param=true", "https://www.reddit.com/r/subreddit/"},
			wantErr: false,
		},
		{
			name: "escaped HTML content with links",
			entry: mockContentProvider{
				id:      "5",
				content: `&lt;a href=&quot;http://example.com&quot;&gt;Example&lt;/a&gt;`,
			},
			want:    []string{"http://example.com"},
			wantErr: false,
		},
		{
			name: "real RSS entry with multiple huggingface and reddit links",
			entry: mockContentProvider{
				id:      "t3_1kf1yg9",
				content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt; &lt;img src=&quot;https://external-preview.redd.it/jJ4wm0NIfgUy0MSOkw2YI6r-EjpVW_Y_SPR-xICfNk4.jpg?width=640&amp;amp;crop=smart&amp;amp;auto=webp&amp;amp;s=8bf4c693cb7ebd3ae7a7b3eb2dc65cfbfc6e1d6d&quot; alt=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; title=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Since IQ4_XS is my favorite quant for 32B models, I decided to run some benchmarks to compare IQ4_XS GGUFs from different sources.&lt;/p&gt; &lt;p&gt;&lt;strong&gt;MMLU-PRO 0.25 subset(3003 questions), 0 temp, No Think, IQ4_XS, Q8 KV Cache&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;The entire benchmark took &lt;strong&gt;&lt;em&gt;11 hours, 37 minutes, and 30 seconds.&lt;/em&gt;&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&quot;&gt;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&lt;/a&gt;&lt;/p&gt; &lt;p&gt;The difference is apparently minimum, so just keep using whatever iq4 quant you already downloaded. &lt;/p&gt; &lt;p&gt;&lt;em&gt;The official MMLU-PRO leaderboard is listing the score of Qwen3 base model instead of instruct, that&amp;#39;s why these iq4 quants score higher than the one on MMLU-PRO leaderboard.&lt;/em&gt;&lt;/p&gt; &lt;p&gt;gguf source:&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&quot;&gt;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/AaronFeng47&quot;&gt; /u/AaronFeng47 &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			},
			want: []string{
				"https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/",
				"https://external-preview.redd.it/jJ4wm0NIfgUy0MSOkw2YI6r-EjpVW_Y_SPR-xICfNk4.jpg?width=640&crop=smart&auto=webp&s=8bf4c693cb7ebd3ae7a7b3eb2dc65cfbfc6e1d6d",
				"https://preview.redd.it/9ptc0cl2svye1.png?width=2475&format=png&auto=webp&s=06a3b551fba60a33877f8e67af9932e381a15cc6",
				"https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf",
				"https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf",
				"https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf",
				"https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf",
				"https://www.reddit.com/user/AaronFeng47",
				"https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/",
				"https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.extractURLsFromEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareUnorderedURLSlices(got, tt.want) {
				gotStr := ""
				for _, u := range got {
					gotStr += "\n    " + u.String()
				}
				wantStr := ""
				for _, u := range tt.want {
					wantStr += "\n    " + u
				}
				t.Errorf("[%s]\n  Got URLs:%s\n  Want URLs:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}

func TestRedditExtractor_ExtractExternalURLsFromEntry(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entry   mockContentProvider
		want    []string
		wantErr bool
	}{
		{
			name: "no URLs in content",
			entry: mockContentProvider{
				id:      "1",
				content: "Just some text without links",
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "only reddit URLs (should be filtered out)",
			entry: mockContentProvider{
				id:      "2",
				content: `<a href="https://reddit.com/r/news">Reddit</a> <a href="https://redd.it/short">Short</a>`,
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "mix of reddit and external URLs (only external returned)",
			entry: mockContentProvider{
				id:      "3",
				content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/&quot;&gt; &lt;img src=&quot;placeholder.jpg&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Check this &lt;a href=&quot;http://external.example.com/page&quot;&gt;cool link&lt;/a&gt;!&lt;/p&gt; &lt;p&gt;And another one &lt;a href=&quot;https://another.example.org&quot;&gt;here&lt;/a&gt;.&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/Recurrents&quot;&gt; /u/Recurrents &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/gallery/1kexdgy&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			},
			want: []string{
				"http://external.example.com/page",
				"https://another.example.org",
			},
			wantErr: false,
		},
		{
			name: "multiple external URLs with some reddit (only external returned)",
			entry: mockContentProvider{
				id:      "4",
				content: `&lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;First link: &lt;a href=&quot;https://example.com/first&quot;&gt;1&lt;/a&gt;. Second link: &lt;a href=&quot;https://test.org/second?param=true&quot;&gt;2&lt;/a&gt;. A reddit one: &lt;a href=&quot;https://www.reddit.com/r/subreddit/&quot;&gt;reddit link&lt;/a&gt;.&lt;/p&gt;&lt;/div&gt;&lt;!-- SC_ON --&gt;`,
			},
			want:    []string{"https://example.com/first", "https://test.org/second?param=true"},
			wantErr: false,
		},
		{
			name: "escaped HTML content with links",
			entry: mockContentProvider{
				id:      "5",
				content: `&lt;a href=&quot;http://example.com&quot;&gt;Example&lt;/a&gt;`,
			},
			want:    []string{"http://example.com"},
			wantErr: false,
		},
		{
			name: "real RSS entry with multiple huggingface and reddit links (only external returned)",
			entry: mockContentProvider{
				id:      "t3_1kf1yg9",
				content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt; &lt;img src=&quot;https://external-preview.redd.it/jJ4wm0NIfgUy0MSOkw2YI6r-EjpVW_Y_SPR-xICfNk4.jpg?width=640&amp;amp;crop=smart&amp;amp;auto=webp&amp;amp;s=8bf4c693cb7ebd3ae7a7b3eb2dc65cfbfc6e1d6d&quot; alt=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; title=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Since IQ4_XS is my favorite quant for 32B models, I decided to run some benchmarks to compare IQ4_XS GGUFs from different sources.&lt;/p&gt; &lt;p&gt;&lt;strong&gt;MMLU-PRO 0.25 subset(3003 questions), 0 temp, No Think, IQ4_XS, Q8 KV Cache&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;The entire benchmark took &lt;strong&gt;&lt;em&gt;11 hours, 37 minutes, and 30 seconds.&lt;/em&gt;&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&quot;&gt;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&lt;/a&gt;&lt;/p&gt; &lt;p&gt;The difference is apparently minimum, so just keep using whatever iq4 quant you already downloaded. &lt;/p&gt; &lt;p&gt;&lt;em&gt;The official MMLU-PRO leaderboard is listing the score of Qwen3 base model instead of instruct, that&amp;#39;s why these iq4 quants score higher than the one on MMLU-PRO leaderboard.&lt;/em&gt;&lt;/p&gt; &lt;p&gt;gguf source:&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&quot;&gt;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/AaronFeng47&quot;&gt; /u/AaronFeng47 &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			},
			want: []string{
				"https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf",
				"https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf",
				"https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf",
				"https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.ExtractExternalURLsFromEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareUnorderedURLSlices(got, tt.want) {
				gotStr := ""
				for _, u := range got {
					gotStr += "\n    " + u.String()
				}
				wantStr := ""
				for _, u := range tt.want {
					wantStr += "\n    " + u
				}
				t.Errorf("[%s]\n  Got URLs:%s\n  Want URLs:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}

func TestRedditExtractor_ExtractExternalURLsFromEntries(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entries []mockContentProvider
		want    map[string][]string
		wantErr bool
	}{
		{
			name:    "no entries",
			entries: []mockContentProvider{},
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name: "single entry no URLs",
			entries: []mockContentProvider{
				{
					id:      "1",
					content: "Just some text without links",
				},
			},
			want:    map[string][]string{"1": {}},
			wantErr: false,
		},
		{
			name: "single entry with reddit URLs (filtered out)",
			entries: []mockContentProvider{
				{
					id:      "2",
					content: `<a href="https://reddit.com/r/news">Reddit</a> <a href="https://redd.it/short">Short</a>`,
				},
			},
			want:    map[string][]string{"2": {}},
			wantErr: false,
		},
		{
			name: "single entry with external URLs",
			entries: []mockContentProvider{
				{
					id:      "3",
					content: `Check this <a href="http://external.example.com/page">cool link</a>!`,
				},
			},
			want:    map[string][]string{"3": {"http://external.example.com/page"}},
			wantErr: false,
		},
		{
			name: "multiple entries with mixed URLs",
			entries: []mockContentProvider{
				{
					id:      "4",
					content: `<a href="https://reddit.com/r/news">Reddit</a>`,
				},
				{
					id:      "5",
					content: `Check this <a href="http://external.example.com/page">cool link</a>!`,
				},
				{
					id:      "6",
					content: `<a href="https://redd.it/short">Short</a>`,
				},
				{
					id:      "7",
					content: `Another link <a href="https://example.com/another">here</a>!`,
				},
			},
			want: map[string][]string{
				"4": {},
				"5": {"http://external.example.com/page"},
				"6": {},
				"7": {"https://example.com/another"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			entries := make([]ContentProvider, len(tt.entries))
			for i, entry := range tt.entries {
				entries[i] = entry
			}
			got, err := extractor.ExtractExternalURLsFromEntries(entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareURLSliceMaps(got, tt.want) {
				gotStr := ""
				for k, urls := range got {
					gotStr += "\n  " + k + ":"
					for _, u := range urls {
						gotStr += "\n    " + u.String()
					}
				}
				wantStr := ""
				for k, urls := range tt.want {
					wantStr += "\n  " + k + ":"
					for _, u := range urls {
						wantStr += "\n    " + u
					}
				}
				t.Errorf("[%s]\n  Got URLs by entry:%s\n  Want URLs by entry:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}

func TestRedditExtractor_ExtractImageURLsFromEntry(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entry   mockContentProvider
		want    []string
		wantErr bool
	}{
		{
			name: "no image URLs in content",
			entry: mockContentProvider{
				id:      "1",
				content: "Just some text without image links",
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "valid image URLs",
			entry: mockContentProvider{
				id:      "2",
				content: `<img src="https://example.com/image.jpg"> <img src="https://example.com/photo.png">`,
			},
			want:    []string{"https://example.com/image.jpg", "https://example.com/photo.png"},
			wantErr: false,
		},
		{
			name: "invalid image URLs",
			entry: mockContentProvider{
				id:      "3",
				content: `<img src="not-a-url">`,
			},
			want:    []string{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.ExtractImageURLsFromEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareUnorderedURLSlices(got, tt.want) {
				gotStr := ""
				for _, u := range got {
					gotStr += "\n    " + u.String()
				}
				wantStr := ""
				for _, u := range tt.want {
					wantStr += "\n    " + u
				}
				t.Errorf("[%s]\n  Got URLs:%s\n  Want URLs:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}

func TestRedditExtractor_ExtractImageURLsFromEntries(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entries []mockContentProvider
		want    map[string][]string
		wantErr bool
	}{
		{
			name:    "no entries",
			entries: []mockContentProvider{},
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name: "single entry with image URLs",
			entries: []mockContentProvider{
				{
					id:      "1",
					content: `<img src="https://example.com/image.jpg">`,
				},
			},
			want: map[string][]string{
				"1": {"https://example.com/image.jpg"},
			},
			wantErr: false,
		},
		{
			name: "multiple entries with mixed content",
			entries: []mockContentProvider{
				{
					id:      "2",
					content: `<img src="https://example.com/photo.png">`,
				},
				{
					id:      "3",
					content: "No images here",
				},
			},
			want: map[string][]string{
				"2": {"https://example.com/photo.png"},
				"3": {},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert mockContentProvider slice to ContentProvider slice
			entries := make([]ContentProvider, len(tt.entries))
			for i, e := range tt.entries {
				entries[i] = e
			}

			got, err := extractor.ExtractImageURLsFromEntries(entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("[%s] error = %v, wantErr %v", tt.name, err, tt.wantErr)
				return
			}
			if !compareURLSliceMaps(got, tt.want) {
				t.Errorf("[%s]\n  Got:  %v\n  Want: %v", tt.name, got, tt.want)
			}
		})
	}
}

func TestFilterNonHTTPProtocols(t *testing.T) {
	tests := []struct {
		name      string
		inputURLs []string
		wantURLs  []string
	}{
		{
			name:      "empty slice",
			inputURLs: []string{},
			wantURLs:  []string{},
		},
		{
			name:      "only http/https",
			inputURLs: []string{"http://example.com", "https://another.org/path"},
			wantURLs:  []string{"http://example.com", "https://another.org/path"},
		},
		{
			name:      "only non-http/https",
			inputURLs: []string{"mailto:test@example.com", "ftp://ftp.example.com", "irc://irc.example.com/channel"},
			wantURLs:  []string{},
		},
		{
			name:      "mixed protocols",
			inputURLs: []string{"http://example.com", "mailto:test@example.com", "https://another.org/path", "ftp://ftp.example.com"},
			wantURLs:  []string{"http://example.com", "https://another.org/path"},
		},
		{
			name:      "with relative paths and fragments (should be filtered out as they don't have http/https schemes)",
			inputURLs: []string{"/just/a/path", "#fragment", "http://example.com/path#frag", "https://another.com"},
			wantURLs:  []string{"http://example.com/path#frag", "https://another.com"},
		},
		{
			name:      "invalid and unparseable URLs",
			inputURLs: []string{"http://valid.com", "://invalid-url", "http://[::1]:namedport", "another valid https://url.com"},
			wantURLs:  []string{"http://valid.com"}, // "another valid https://url.com" is not a valid URL, url.Parse will make its scheme ""
		},
		{
			name:      "schemes with mixed case",
			inputURLs: []string{"HTTP://example.com", "Https://another.org/path", "FTP://ftp.example.com"},
			wantURLs:  []string{"HTTP://example.com", "Https://another.org/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotURLs := filterNonHTTPProtocols(tt.inputURLs)
			if !compareUnorderedStringSlices(gotURLs, tt.wantURLs) {
				gotStr := ""
				for _, u := range gotURLs {
					gotStr += "\n    " + u
				}
				wantStr := ""
				for _, u := range tt.wantURLs {
					wantStr += "\n    " + u
				}
				t.Errorf("[%s]\n  Got URLs:%s\n  Want URLs:%s", tt.name, gotStr, wantStr)
			}
		})
	}
}
