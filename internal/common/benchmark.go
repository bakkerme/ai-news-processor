package common

// BenchmarkData represents the data collected during benchmarking
type BenchmarkData struct {
	RawInput []string `json:"raw_input"` // The raw input strings sent to the LLM
	Results  []Item   `json:"results"`   // The processed results from the LLM
}
