package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	"github.com/bakkerme/ai-news-processor/internal/bench"
	"github.com/bakkerme/ai-news-processor/internal/customerrors"
	"github.com/bakkerme/ai-news-processor/internal/llm"
	"github.com/bakkerme/ai-news-processor/internal/models"
	"github.com/bakkerme/ai-news-processor/internal/openai"
	"github.com/bakkerme/ai-news-processor/internal/persona"
	"github.com/bakkerme/ai-news-processor/internal/specification"
)

const evaluationPrompt = `You are an expert in evaluating AI-generated content. Your task is to evaluate the quality of the following news summary, focusing purely on how well it summarizes and analyzes the content.

The persona is {{.PersonaIdentity}}

The persona's focus areas are:
{{range .FocusAreas}}* {{.}}
{{end}}

The summary should be marked as irrelevant if it matches:
{{range .ExclusionCriteria}}* {{.}}
{{end}}

For each summary, evaluate:

1. Summary Quality (choose one):
   - Excellent: Comprehensive summary with exceptional technical depth, clear analysis, and insightful commentary
   - Good: Clear summary with solid technical details and meaningful analysis
   - Fair: Basic summary with some technical details but lacks depth or clarity
   - Poor: Incomplete or unclear summary lacking essential details

2. Evaluation Criteria:
   - Comprehensiveness: Does it capture all key details?
   - Technical Accuracy: Are technical concepts and specifications explained correctly?
   - Clarity: Is the information presented in a clear, well-structured manner?
   - Comment Integration: Are community discussions and feedback well-analyzed?

3. Relevance Assessment (separate from quality rating):
   - Check if the original content matches any exclusion criteria. If it does, the IsRelevant flag should be false.
   - Evaluate if the IsRelevant flag is set appropriately
   - Assess if the relevance explanation is clear and justified

Respond with a JSON object containing:
{
  "quality_rating": string,  // One of: "Excellent", "Good", "Fair", "Poor"
  "quality_explanation": string,  // Detailed explanation of the summary quality
  "relevance_correct": boolean,  // Whether IsRelevant flag was set correctly based on exclusion criteria
  "relevance_explanation": string // Explanation of relevance assessment
}`

// EvaluationResult represents the structure of the benchmark evaluation response
// (Benchmark-specific, not shared with internal packages)
type EvaluationResult struct {
	QualityRating        string `json:"quality_rating" jsonschema_description:"Descriptive rating for summary quality (Excellent, Good, Fair, Poor)" jsonschema:"required"`
	QualityExplanation   string `json:"quality_explanation" jsonschema_description:"Detailed explanation of the rating" jsonschema:"required"`
	RelevanceExplanation string `json:"relevance_explanation" jsonschema_description:"Explanation of relevance assessment" jsonschema:"required"`
	RelevanceCorrect     bool   `json:"relevance_correct" jsonschema_description:"Whether IsRelevant flag was set correctly" jsonschema:"required"`
}

// Generate the JSON schema for EvaluationResult
var EvaluationResultSchema = llm.GenerateSchema[EvaluationResult]()

// ChatCompletionForBenchmarkEvaluation queries the LLM for a benchmark evaluation using the EvaluationResult schema
func ChatCompletionForBenchmarkEvaluation(llmClient openai.OpenAIClient, systemPrompt string, userPrompts []string, results chan customerrors.ErrorString) {
	llmClient.ChatCompletion(
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
	QualityScore        float64                     `json:"quality_score"`
	DetailedEvaluations map[string]EvaluationResult `json:"detailed_evaluations"`
	PersonaName         string                      `json:"persona_name"`
	PersonaFocusAreas   []string                    `json:"persona_focus_areas"`
	MissingItems        []string                    `json:"missing_items"`
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

	// Load benchmark data from benchmark.json
	log.Println("Loading benchmark data from benchmark.json...")
	benchmarkData, err := loadBenchmarkData("./results/benchmark.json")
	if err != nil {
		log.Printf("Error loading benchmark data: %v\n", err)
		os.Exit(1)
	}
	log.Printf("Loaded benchmark data with persona: %s\n", benchmarkData.Persona)

	// Set up persona handling
	personaPath := "../personas/" // default to Docker path

	// Load all personas and find the matching one
	personas, err := persona.LoadPersonas(personaPath)
	if err != nil {
		log.Printf("Error loading personas: %v\n", err)
		os.Exit(1)
	}

	var selectedPersona *persona.Persona
	for i, p := range personas {
		if p.Name == benchmarkData.Persona {
			selectedPersona = &personas[i]
			break
		}
	}

	if selectedPersona == nil {
		log.Printf("Error: Could not find persona %s in personas directory\n", benchmarkData.Persona)
		os.Exit(1)
	}

	log.Printf("Found matching persona: %s\n", selectedPersona.Name)

	// Generate evaluation prompt with persona-specific information
	tmpl, err := template.New("evaluation").Parse(evaluationPrompt)
	if err != nil {
		log.Printf("Error parsing evaluation prompt template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, selectedPersona)
	if err != nil {
		log.Printf("Error executing evaluation prompt template: %v\n", err)
		os.Exit(1)
	}

	fullPrompt := buf.String()

	// Build a map from ID to raw_input for matching
	rawInputByID := make(map[string]string)
	processedIDs := make(map[string]bool)
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
	results.PersonaName = selectedPersona.Name
	results.PersonaFocusAreas = selectedPersona.FocusAreas
	results.MissingItems = make([]string, 0)

	// Process each entry in the benchmark data
	for _, result := range benchmarkData.Results {
		if result.ID == "" {
			log.Printf("Warning: Empty ID for result\n")
			continue
		}

		processedIDs[result.ID] = true
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
		log.Printf("ChatCompletioning LLM for evaluation of entry ID: %s...\n", result.ID)
		resultChan := make(chan customerrors.ErrorString, 1)
		ChatCompletionForBenchmarkEvaluation(llmClient, fullPrompt, []string{evaluationInput}, resultChan)
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

	// Check for missing items
	for id := range rawInputByID {
		if !processedIDs[id] {
			log.Printf("Found missing item (ID: %s)...\n", id)
			results.MissingItems = append(results.MissingItems, id)

			// Add a Poor rating evaluation for the missing item
			results.DetailedEvaluations[id] = EvaluationResult{
				QualityRating:        "Poor",
				QualityExplanation:   "Item was present in raw input but missing from processed results",
				RelevanceCorrect:     false,
				RelevanceExplanation: "Unable to assess relevance as item was not processed",
			}
			results.TotalItems++
		}
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

		// Calculate quality score with Poor rated at 0%
		var totalQualityScore float64
		for _, eval := range results.DetailedEvaluations {
			switch eval.QualityRating {
			case "Excellent":
				totalQualityScore += 100.0
			case "Good":
				totalQualityScore += 75.0
			case "Fair":
				totalQualityScore += 50.0
			case "Poor":
				totalQualityScore += 0.0
			}
		}
		results.QualityScore = totalQualityScore / float64(results.TotalItems)
	}

	// Output results
	log.Println("Outputting results...")
	outputResults(results, benchmarkData.Results, selectedPersona)
}

func loadBenchmarkData(filename string) (*bench.BenchmarkData, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read benchmark data: %w", err)
	}

	var benchmarkData bench.BenchmarkData
	err = json.Unmarshal(data, &benchmarkData)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal benchmark data: %w", err)
	}

	return &benchmarkData, nil
}

func formatSummary(item models.Item) string {
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("Title: %s\n", item.Title))
	summary.WriteString(fmt.Sprintf("ID: %s\n", item.ID))
	summary.WriteString(fmt.Sprintf("Summary: %s\n", item.Summary))
	summary.WriteString(fmt.Sprintf("Comment Summary: %s\n", item.CommentSummary))
	// summary.WriteString(fmt.Sprintf("Relevance: %s\n", item.Relevance))
	summary.WriteString(fmt.Sprintf("IsRelevant: %v\n", item.IsRelevant))
	return summary.String()
}

func outputResults(results BenchmarkResults, items []models.Item, p *persona.Persona) {
	// Build a map from ID to Title
	titleMap := make(map[string]string)
	for _, item := range items {
		titleMap[item.ID] = item.Title
	}

	// Print summary
	fmt.Printf("\nBenchmark Results for Persona: %s\n", p.Name)
	fmt.Printf("Total Items Evaluated: %d\n", results.TotalItems)
	fmt.Printf("Relevance Accuracy: %.2f%%\n", results.RelevanceAccuracy*100)
	fmt.Printf("Quality Score: %.2f%%\n", results.QualityScore)
	fmt.Printf("Missing Items: %d\n", len(results.MissingItems))

	// Print missing items if any
	if len(results.MissingItems) > 0 {
		fmt.Printf("\nMissing Items:\n")
		for _, id := range results.MissingItems {
			fmt.Printf("- Item ID: %s\n", id)
		}
	}

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

	// Create filename with persona name
	personaUsed := strings.ToLower(strings.ReplaceAll(p.Name, " ", "_"))
	filename := fmt.Sprintf("./results/benchmark_results_%s_%s.json", personaUsed, time.Now().Format("2006-01-02_15-04-05"))

	err = os.WriteFile(filename, resultsJson, 0644)
	if err != nil {
		log.Printf("Error writing results file: %v\n", err)
	} else {
		log.Printf("Results written to %s\n", filename)
	}
}
