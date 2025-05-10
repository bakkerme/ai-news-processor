package urlextraction

import (
	"reflect"
	"sort"
	"testing"

	"github.com/bakkerme/ai-news-processor/internal/rss"
)

// Helper function to compare slices regardless of order
func compareUnorderedStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	// Create copies to avoid modifying originals
	aCopy := make([]string, len(a))
	bCopy := make([]string, len(b))
	copy(aCopy, a)
	copy(bCopy, b)

	// Sort both slices
	sort.Strings(aCopy)
	sort.Strings(bCopy)

	return reflect.DeepEqual(aCopy, bCopy)
}

// Helper function to compare string slice maps regardless of slice order
func compareStringSliceMaps(a, b map[string][]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v1 := range a {
		v2, ok := b[k]
		if !ok {
			return false
		}
		if !compareUnorderedStringSlices(v1, v2) {
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
				t.Errorf("RedditExtractor.extractURLsFromHTML() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareUnorderedStringSlices(gotURLs, tt.wantURLs) {
				t.Errorf("RedditExtractor.extractURLsFromHTML() = %v, want %v", gotURLs, tt.wantURLs)
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
				t.Errorf("RedditExtractor.isRedditDomain() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RedditExtractor.isRedditDomain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedditExtractor_ExtractURLsFromEntry(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entry   rss.Entry
		want    []string
		wantErr bool
	}{
		{
			name: "no URLs in content",
			entry: rss.Entry{
				ID:      "1",
				Content: "Just some text without links",
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "only reddit URLs",
			entry: rss.Entry{
				ID:      "2",
				Content: `<a href="https://reddit.com/r/news">Reddit</a> <a href="https://redd.it/short">Short</a>`,
			},
			want:    []string{},
			wantErr: false,
		},
		{
			name: "mix of reddit and external URLs",
			entry: rss.Entry{
				ID:      "3",
				Content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/&quot;&gt; &lt;img src=&quot;placeholder.jpg&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Check this &lt;a href=&quot;http://external.example.com/page&quot;&gt;cool link&lt;/a&gt;!&lt;/p&gt; &lt;p&gt;And another one &lt;a href=&quot;https://another.example.org&quot;&gt;here&lt;/a&gt;.&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/Recurrents&quot;&gt; /u/Recurrents &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/gallery/1kexdgy&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
			},
			want:    []string{"http://external.example.com/page", "https://another.example.org"},
			wantErr: false,
		},
		{
			name: "multiple external URLs",
			entry: rss.Entry{
				ID:      "4",
				Content: `&lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;First link: &lt;a href=&quot;https://example.com/first&quot;&gt;1&lt;/a&gt;. Second link: &lt;a href=&quot;https://test.org/second?param=true&quot;&gt;2&lt;/a&gt;. A reddit one: &lt;a href=&quot;https://www.reddit.com/r/subreddit/&quot;&gt;reddit link&lt;/a&gt;.&lt;/p&gt;&lt;/div&gt;&lt;!-- SC_ON --&gt;`,
			},
			want:    []string{"https://example.com/first", "https://test.org/second?param=true"},
			wantErr: false,
		},
		{
			name: "escaped HTML content with links",
			entry: rss.Entry{
				ID:      "5",
				Content: `&lt;a href=&quot;http://example.com&quot;&gt;Example&lt;/a&gt;`,
			},
			want:    []string{"http://example.com"},
			wantErr: false,
		},
		{
			name: "real RSS entry with multiple huggingface and reddit links",
			entry: rss.Entry{
				ID:      "t3_1kf1yg9",
				Content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt; &lt;img src=&quot;https://external-preview.redd.it/jJ4wm0NIfgUy0MSOkw2YI6r-EjpVW_Y_SPR-xICfNk4.jpg?width=640&amp;amp;crop=smart&amp;amp;auto=webp&amp;amp;s=8bf4c693cb7ebd3ae7a7b3eb2dc65cfbfc6e1d6d&quot; alt=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; title=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Since IQ4_XS is my favorite quant for 32B models, I decided to run some benchmarks to compare IQ4_XS GGUFs from different sources.&lt;/p&gt; &lt;p&gt;&lt;strong&gt;MMLU-PRO 0.25 subset(3003 questions), 0 temp, No Think, IQ4_XS, Q8 KV Cache&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;The entire benchmark took &lt;strong&gt;&lt;em&gt;11 hours, 37 minutes, and 30 seconds.&lt;/em&gt;&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&quot;&gt;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&lt;/a&gt;&lt;/p&gt; &lt;p&gt;The difference is apparently minimum, so just keep using whatever iq4 quant you already downloaded. &lt;/p&gt; &lt;p&gt;&lt;em&gt;The official MMLU-PRO leaderboard is listing the score of Qwen3 base model instead of instruct, that&amp;#39;s why these iq4 quants score higher than the one on MMLU-PRO leaderboard.&lt;/em&gt;&lt;/p&gt; &lt;p&gt;gguf source:&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&quot;&gt;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/AaronFeng47&quot;&gt; /u/AaronFeng47 &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
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
			got, err := extractor.ExtractURLsFromEntry(tt.entry)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedditExtractor.ExtractURLsFromEntry() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareUnorderedStringSlices(got, tt.want) {
				t.Errorf("RedditExtractor.ExtractURLsFromEntry() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRedditExtractor_ExtractURLsFromEntries(t *testing.T) {
	extractor := NewRedditExtractor()
	tests := []struct {
		name    string
		entries []rss.Entry
		want    map[string][]string
		wantErr bool
	}{
		{
			name:    "empty entries",
			entries: []rss.Entry{},
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name: "entry without ID",
			entries: []rss.Entry{
				{
					Content: `<a href="https://example.com">Link</a>`,
				},
			},
			want:    map[string][]string{},
			wantErr: false,
		},
		{
			name: "multiple entries with mixed URLs",
			entries: []rss.Entry{
				{
					ID:      "1",
					Content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/&quot;&gt; &lt;img src=&quot;placeholder.jpg&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Check this &lt;a href=&quot;http://external.example.com/page&quot;&gt;cool link&lt;/a&gt;!&lt;/p&gt; &lt;p&gt;And another one &lt;a href=&quot;https://another.example.org&quot;&gt;here&lt;/a&gt;.&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/Recurrents&quot;&gt; /u/Recurrents &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/gallery/1kexdgy&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kexdgy/comments/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
				},
				{
					ID:      "2",
					Content: `&lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;First link: &lt;a href=&quot;https://example.com/first&quot;&gt;1&lt;/a&gt;. Second link: &lt;a href=&quot;https://test.org/second?param=true&quot;&gt;2&lt;/a&gt;. A reddit one: &lt;a href=&quot;https://www.reddit.com/r/subreddit/&quot;&gt;reddit link&lt;/a&gt;.&lt;/p&gt;&lt;/div&gt;&lt;!-- SC_ON --&gt;`,
				},
				{
					ID:      "t3_1kf1yg9",
					Content: `&lt;table&gt; &lt;tr&gt;&lt;td&gt; &lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt; &lt;img src=&quot;https://external-preview.redd.it/jJ4wm0NIfgUy0MSOkw2YI6r-EjpVW_Y_SPR-xICfNk4.jpg?width=640&amp;amp;crop=smart&amp;amp;auto=webp&amp;amp;s=8bf4c693cb7ebd3ae7a7b3eb2dc65cfbfc6e1d6d&quot; alt=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; title=&quot;Qwen3-32B-IQ4_XS GGUFs - MMLU-PRO benchmark comparison&quot; /&gt; &lt;/a&gt; &lt;/td&gt;&lt;td&gt; &lt;!-- SC_OFF --&gt;&lt;div class=&quot;md&quot;&gt;&lt;p&gt;Since IQ4_XS is my favorite quant for 32B models, I decided to run some benchmarks to compare IQ4_XS GGUFs from different sources.&lt;/p&gt; &lt;p&gt;&lt;strong&gt;MMLU-PRO 0.25 subset(3003 questions), 0 temp, No Think, IQ4_XS, Q8 KV Cache&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;The entire benchmark took &lt;strong&gt;&lt;em&gt;11 hours, 37 minutes, and 30 seconds.&lt;/em&gt;&lt;/strong&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&quot;&gt;https://preview.redd.it/9ptc0cl2svye1.png?width=2475&amp;amp;format=png&amp;amp;auto=webp&amp;amp;s=06a3b551fba60a33877f8e67af9932e381a15cc6&lt;/a&gt;&lt;/p&gt; &lt;p&gt;The difference is apparently minimum, so just keep using whatever iq4 quant you already downloaded. &lt;/p&gt; &lt;p&gt;&lt;em&gt;The official MMLU-PRO leaderboard is listing the score of Qwen3 base model instead of instruct, that&amp;#39;s why these iq4 quants score higher than the one on MMLU-PRO leaderboard.&lt;/em&gt;&lt;/p&gt; &lt;p&gt;gguf source:&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&quot;&gt;https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&quot;&gt;https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;p&gt;&lt;a href=&quot;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&quot;&gt;https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf&lt;/a&gt;&lt;/p&gt; &lt;/div&gt;&lt;!-- SC_ON --&gt; &amp;#32; submitted by &amp;#32; &lt;a href=&quot;https://www.reddit.com/user/AaronFeng47&quot;&gt; /u/AaronFeng47 &lt;/a&gt; &lt;br/&gt; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[link]&lt;/a&gt;&lt;/span&gt; &amp;#32; &lt;span&gt;&lt;a href=&quot;https://www.reddit.com/r/LocalLLaMA/comments/1kf1yg9/qwen332biq4_xs_ggufs_mmlupro_benchmark_comparison/&quot;&gt;[comments]&lt;/a&gt;&lt;/span&gt; &lt;/td&gt;&lt;/tr&gt;&lt;/table&gt;`,
				},
			},
			want: map[string][]string{
				"1": {"http://external.example.com/page", "https://another.example.org"},
				"2": {"https://example.com/first", "https://test.org/second?param=true"},
				"t3_1kf1yg9": {
					"https://huggingface.co/unsloth/Qwen3-32B-GGUF/blob/main/Qwen3-32B-IQ4_XS.gguf",
					"https://huggingface.co/unsloth/Qwen3-32B-128K-GGUF/blob/main/Qwen3-32B-128K-IQ4_XS.gguf",
					"https://huggingface.co/bartowski/Qwen_Qwen3-32B-GGUF/blob/main/Qwen_Qwen3-32B-IQ4_XS.gguf",
					"https://huggingface.co/mradermacher/Qwen3-32B-i1-GGUF/blob/main/Qwen3-32B.i1-IQ4_XS.gguf",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := extractor.ExtractURLsFromEntries(tt.entries)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedditExtractor.ExtractURLsFromEntries() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !compareStringSliceMaps(got, tt.want) {
				t.Errorf("RedditExtractor.ExtractURLsFromEntries() = %v, want %v", got, tt.want)
			}
		})
	}
}
