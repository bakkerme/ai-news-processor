# Benchmark Tool Documentation

This tool enables quantitative evaluation of AI-generated news summaries by measuring both summary quality and relevance against defined personas. It automates end-to-end benchmarking to support continuous improvement.

## Design Principles
- **High-Capacity Evaluation Model**: We leverage `qwen2.5-72b-instruct`, a large language model with extensive reasoning and comprehension capabilities, to provide objective and nuanced assessments of summary quality and relevance.
- **Decoupled Evaluation**: By using a separate LLM for evaluation distinct from the generation model, we mitigate generation-specific biases and ensure that assessment results reflect general content quality.

## How to Run a Benchmark

1. **Generate benchmark data**:
   Enable debug output and run the main application to dump raw inputs and summaries:
   ```bash
   export ANP_DEBUG_OUTPUT_BENCHMARK=true
   go run .
   ```
   This writes `bench/results/benchmark.json` containing the raw feed data, generated summaries, and selected persona.

2. **Run the benchmark evaluation**:
   ```bash
   cd bench
   go run .
   ```

3. **Review results**:
   Console logs trace each loading and evaluation stage. The final JSON file in `bench/results/` includes detailed item evaluations and aggregate metrics.

## Output

Upon completion, the tool emits:

- **Console output**: Progress logs for configuration, data loading, per-item evaluation, and metric computation.
- **Results file**: `bench/results/benchmark_results_<persona>_<timestamp>.json` containing:
  - `total_items` (int)
  - `relevance_accuracy` (float, 0.0–1.0)
  - `quality_score` (float, 0.0–100.0)
  - `detailed_evaluations` (map of item IDs to LLM-driven evaluation objects)
  - `persona_name` and `persona_focus_areas`
  - `missing_items` (any IDs present in raw input but not processed)

These results enable you to gauge model performance at a glance:
- A high **relevance_accuracy** indicates accurate identification of important content, while lower values highlight potential blind spots.
- The **quality_score** reflects the average strength of summaries in clarity, depth, and completeness; use it to track improvements over time.
- The **detailed_evaluations** section pinpoints specific articles where the model excels or struggles, guiding prompt refinement or persona adjustments.
- **Missing_items** reveal mismatches between raw and processed data, useful for diagnosing pipeline or parsing issues.

### Summary Quality (Descriptive Rubric -> Score)
The LLM assigns one of the following ratings, which are then converted to a numerical score for the aggregate `quality_score`:
- Excellent (100): Meets all criteria with high quality and insight.
- Good (75): Meets most criteria, minor issues.
- Fair (50): Some important criteria are missing or weak.
- Poor (0): Fails to meet most criteria.

## Input JSON Schema (`bench/results/benchmark.json`)
```json
{
  "raw_input": ["string..."],        // Original content sent to the processor
  "results": [                        // Processed output from the main tool
    {
      "ID": "string",
      "Title": "string",
      "Summary": "string",
      "CommentSummary": "string",
      "Relevance": "string",
      "IsRelevant": true
    }
  ],
  "persona": "persona_name"         // Must match a persona in the `personas/` directory
}
```
