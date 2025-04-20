package llamacpp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	// "time"
)

// CompletionRequest represents the request payload for the /completion endpoint.
type CompletionRequest struct {
	Prompt              string          `json:"prompt"`
	Temperature         float64         `json:"temperature,omitempty"`
	DynatempRange       float64         `json:"dynatemp_range,omitempty"`
	DynatempExponent    float64         `json:"dynatemp_exponent,omitempty"`
	TopK                int             `json:"top_k,omitempty"`
	TopP                float64         `json:"top_p,omitempty"`
	MinP                float64         `json:"min_p,omitempty"`
	NPredict            int             `json:"n_predict,omitempty"`
	NIndent             int             `json:"n_indent,omitempty"`
	NKeep               int             `json:"n_keep,omitempty"`
	Stream              bool            `json:"stream,omitempty"`
	Stop                []string        `json:"stop,omitempty"`
	TypicalP            float64         `json:"typical_p,omitempty"`
	RepeatPenalty       float64         `json:"repeat_penalty,omitempty"`
	RepeatLastN         int             `json:"repeat_last_n,omitempty"`
	PresencePenalty     float64         `json:"presence_penalty,omitempty"`
	FrequencyPenalty    float64         `json:"frequency_penalty,omitempty"`
	DryMultiplier       float64         `json:"dry_multiplier,omitempty"`
	DryBase             float64         `json:"dry_base,omitempty"`
	DryAllowedLength    int             `json:"dry_allowed_length,omitempty"`
	DryPenaltyLastN     int             `json:"dry_penalty_last_n,omitempty"`
	DrySequenceBreakers []string        `json:"dry_sequence_breakers,omitempty"`
	XtcProbability      float64         `json:"xtc_probability,omitempty"`
	XtcThreshold        float64         `json:"xtc_threshold,omitempty"`
	Mirostat            int             `json:"mirostat,omitempty"`
	MirostatTau         float64         `json:"mirostat_tau,omitempty"`
	MirostatEta         float64         `json:"mirostat_eta,omitempty"`
	Grammar             string          `json:"grammar,omitempty"`
	JsonSchema          interface{}     `json:"json_schema,omitempty"`
	Seed                int             `json:"seed,omitempty"`
	IgnoreEos           bool            `json:"ignore_eos,omitempty"`
	LogitBias           [][]interface{} `json:"logit_bias,omitempty"`
	NProbs              int             `json:"n_probs,omitempty"`
	MinKeep             int             `json:"min_keep,omitempty"`
	TMaxPredictMs       int             `json:"t_max_predict_ms,omitempty"`
	ImageData           []struct {
		Data string `json:"data"`
		ID   int    `json:"id"`
	} `json:"image_data,omitempty"`
	IdSlot            int      `json:"id_slot,omitempty"`
	CachePrompt       bool     `json:"cache_prompt,omitempty"`
	ReturnTokens      bool     `json:"return_tokens,omitempty"`
	Samplers          []string `json:"samplers,omitempty"`
	TimingsPerToken   bool     `json:"timings_per_token,omitempty"`
	PostSamplingProbs bool     `json:"post_sampling_probs,omitempty"`
	ResponseFields    []string `json:"response_fields,omitempty"`
	Lora              []struct {
		ID    int     `json:"id"`
		Scale float64 `json:"scale"`
	} `json:"lora,omitempty"`
}

type CompletionResponse struct {
	Content   string `json:"content"`
	Truncated bool   `json:"truncated"`
}

// Example function to create and send a request
func sendCompletionRequest(req *CompletionRequest, url string) (*CompletionResponse, error) {
	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	client := &http.Client{
		// Timeout: 5 * time.Second,
	}
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Read and print the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil, nil
	}

	// unmarshal body into CompletionRequest
	response := &CompletionResponse{}
	err = json.Unmarshal(body, response)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal api response: %w", err)
	}

	return response, nil
}
