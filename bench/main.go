package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/bakkerme/ai-news-processor/internal/common"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/prompts"
	"github.com/bakkerme/ai-news-processor/internal/specification"
)

const evaluationPrompt = `You are an AI researcher and enthusiast who loves diving deep into technical details. Your task is to evaluate the quality of AI news summaries and their relevance assessment.

First, here is the original prompt used to generate the summaries:

` + "```prompt" + `
%s
` + "```" + `

Now, your task is to evaluate:

1. Summary Quality (choose one):
   - Excellent: Meets all criteria with high quality and insight.
   - Good: Meets most criteria, minor issues.
   - Fair: Some important criteria are missing or weak.
   - Poor: Fails to meet most criteria.

Consider accuracy of technical details, completeness, clarity, emphasis, technical depth, comment analysis, and relevance explanation.

2. Relevance Assessment:
   - Evaluate if the IsRelevant flag is correctly set based on the original criteria
   - Consider both the content and the quality of the explanation

Respond with a JSON object containing:
{
  "quality_rating": string,  // One of: "Excellent", "Good", "Fair", "Poor"
  "quality_explanation": string,  // Detailed explanation of the rating
  "relevance_correct": boolean,  // Whether IsRelevant flag was set correctly
  "relevance_explanation": string // Explanation of relevance assessment
}`

// EvaluationResult represents the structure of the benchmark evaluation response
// (Benchmark-specific, not shared with internal packages)
type EvaluationResult struct {
	QualityRating        string `json:"quality_rating" jsonschema_description:"Descriptive rating for summary quality (Excellent, Good, Fair, Poor)" jsonschema:"required"`
	QualityExplanation   string `json:"quality_explanation" jsonschema_description:"Detailed explanation of the rating" jsonschema:"required"`
	RelevanceCorrect     bool   `json:"relevance_correct" jsonschema_description:"Whether IsRelevant flag was set correctly" jsonschema:"required"`
	RelevanceExplanation string `json:"relevance_explanation" jsonschema_description:"Explanation of relevance assessment" jsonschema:"required"`
}

// Generate the JSON schema for EvaluationResult
var EvaluationResultSchema = openai.GenerateSchema[EvaluationResult]()

// QueryForBenchmarkEvaluation queries the LLM for a benchmark evaluation using the EvaluationResult schema
func QueryForBenchmarkEvaluation(llmClient *openai.Client, systemPrompt string, userPrompts []string, results chan common.ErrorString) {
	llmClient.QueryWithSchema(
		systemPrompt,
		userPrompts,
		EvaluationResultSchema,
		"benchmark_evaluation",
		"an object representing a benchmark evaluation result (quality and relevance)",
		results,
	)
}

type BenchmarkResults struct {
	TotalItems          int                         `json:"total_items"`
	RelevanceAccuracy   float64                     `json:"relevance_accuracy"`
	DetailedEvaluations map[string]EvaluationResult `json:"detailed_evaluations"`
}

func main() {
	// Load configuration using the specification system
	log.Println("Loading configuration...")
	spec, err := specification.GetConfig()
	if err != nil {
		log.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}
	log.Println("Configuration loaded.")

	model := "qwen2.5-72b-instruct"

	// Initialize OpenAI client using values from the specification
	log.Println("Initializing OpenAI client...")
	llmClient := openai.New(
		spec.LlmUrl,
		spec.LlmApiKey,
		model,
	)
	log.Println("OpenAI client initialized.")

	// Generate evaluation prompt with original system prompt
	fullPrompt := fmt.Sprintf(evaluationPrompt, prompts.GetSystemPrompt())

	// Load benchmark data from benchmark.json
	log.Println("Loading benchmark data from benchmark.json...")
	benchmarkData, err := loadBenchmarkData("./benchmark.json")
	if err != nil {
		log.Printf("Error loading benchmark data: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Loaded %d benchmark entries.\n", len(benchmarkData.RawInput))

	// Build a map from ID to raw_input for matching
	rawInputByID := make(map[string]string)
	for _, raw := range benchmarkData.RawInput {
		// Try to extract the ID from the raw input (assuming 'ID: <id>' is present)
		lines := strings.Split(raw, "\n")
		var id string
		for _, line := range lines {
			if strings.HasPrefix(line, "ID: ") {
				id = strings.TrimSpace(strings.TrimPrefix(line, "ID: "))
				break
			}
		}
		if id != "" {
			rawInputByID[id] = raw
		}
	}

	var results BenchmarkResults
	results.DetailedEvaluations = make(map[string]EvaluationResult)

	// Process each entry in the benchmark data
	for _, result := range benchmarkData.Results {
		if result.ID == "" {
			log.Printf("Warning: Empty ID for result\n")
			continue
		}

		log.Printf("Processing entry (ID: %s)...\n", result.ID)

		// Find the matching raw input by ID
		rawInput, ok := rawInputByID[result.ID]
		if !ok {
			log.Printf("Warning: No matching raw input for result ID: %s\n", result.ID)
			continue
		}

		// Create evaluation input
		evaluationInput := fmt.Sprintf("Source Material:\n%s\n\nGenerated Summary:\n%s\n",
			rawInput,
			formatSummary(result))

		// Call LLM for evaluation
		log.Printf("Querying LLM for evaluation of entry ID: %s...\n", result.ID)
		resultChan := make(chan common.ErrorString, 1)
		QueryForBenchmarkEvaluation(llmClient, fullPrompt, []string{evaluationInput}, resultChan)
		evalResponse := <-resultChan
		if evalResponse.Err != nil {
			log.Printf("Error evaluating entry %s: %v\n", result.ID, evalResponse.Err)
			continue
		}

		// Parse evaluation result
		var evalResult EvaluationResult
		jsonStr := llmClient.PreprocessJSON(evalResponse.Value)
		err = json.Unmarshal([]byte(jsonStr), &evalResult)
		if err != nil {
			log.Printf("Error parsing evaluation result for %s: %v\n", result.ID, err)
			continue
		}

		log.Printf("Evaluation for entry ID %s: Quality Rating = %s, Relevance Correct = %v\n", result.ID, evalResult.QualityRating, evalResult.RelevanceCorrect)
		results.DetailedEvaluations[result.ID] = evalResult
		results.TotalItems++
	}

	// Calculate aggregate metrics
	log.Println("Calculating aggregate metrics...")
	var correctRelevance int
	for _, eval := range results.DetailedEvaluations {
		if eval.RelevanceCorrect {
			correctRelevance++
		}
	}

	if results.TotalItems > 0 {
		results.RelevanceAccuracy = float64(correctRelevance) / float64(results.TotalItems)
	}

	// Output results
	log.Println("Outputting results...")
	outputResults(results, benchmarkData.Results)
}

func loadBenchmarkData(filename string) (*common.BenchmarkData, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading benchmark file: %w", err)
	}

	var benchmarkData common.BenchmarkData
	err = json.Unmarshal(data, &benchmarkData)
	if err != nil {
		return nil, fmt.Errorf("error parsing benchmark data: %w", err)
	}

	return &benchmarkData, nil
}

func formatSummary(item common.Item) string {
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Title: %s\n", item.Title))
	summary.WriteString(fmt.Sprintf("ID: %s\n", item.ID))
	summary.WriteString(fmt.Sprintf("Summary: %s\n", item.Summary))
	summary.WriteString(fmt.Sprintf("Comment Summary: %s\n", item.CommentSummary))
	summary.WriteString(fmt.Sprintf("Relevance: %s\n", item.Relevance))
	summary.WriteString(fmt.Sprintf("IsRelevant: %v\n", item.IsRelevant))
	return summary.String()
}

func outputResults(results BenchmarkResults, items []common.Item) {
	// Build a map from ID to Title
	titleMap := make(map[string]string)
	for _, item := range items {
		titleMap[item.ID] = item.Title
	}

	// Print summary
	fmt.Printf("\nBenchmark Results:\n")
	fmt.Printf("Total Items Evaluated: %d\n", results.TotalItems)
	fmt.Printf("Relevance Accuracy: %.2f%%\n", results.RelevanceAccuracy*100)

	// Print detailed evaluations
	fmt.Printf("\nDetailed Evaluations:\n")
	for id, eval := range results.DetailedEvaluations {
		title := titleMap[id]
		fmt.Printf("\nTitle: %s\n", title)
		fmt.Printf("Item ID: %s\n", id)
		fmt.Printf("Quality Rating: %s\n", eval.QualityRating)
		fmt.Printf("Quality Explanation: %s\n", eval.QualityExplanation)
		fmt.Printf("Relevance Correct: %v\n", eval.RelevanceCorrect)
		fmt.Printf("Relevance Explanation: %s\n", eval.RelevanceExplanation)
	}

	// Save results to file
	log.Println("Writing results to benchmark_results.json...")
	resultsJson, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		log.Printf("Error marshaling results: %v\n", err)
		return
	}

	err = os.WriteFile("benchmark_results.json", resultsJson, 0644)
	if err != nil {
		log.Printf("Error writing results file: %v\n", err)
	} else {
		log.Println("Results written to benchmark_results.json.")
	}
}
