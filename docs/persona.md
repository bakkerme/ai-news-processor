# Feature: Persona Configuration

## Overview
The Persona feature enables the AI News Processor to handle multiple configurable personas, each of which defines its own RSS feed, prompt structure, and processing behavior. Personas are loaded at startup, selected via CLI flag or environment variable, and then used to drive the RSS fetching, LLM processing, and output generation steps in a unified pipeline.

---

## 1. Persona YAML Configuration
Each persona is represented by its own YAML file in the `personas/` directory at the project root. Example structure:
```
personas/
  ├─ localllama.yaml
  └─ gardening.yaml
```

Example (`personas/localllama.yaml`):
```yaml
name: "LocalLLaMA"
feed_url: "https://reddit.com/r/localllama.rss"
topic: "AI Technology and Large Language Models"
persona_identity: "You are an AI researcher..."
base_prompt_task: "Analyze each post for technical depth."
summary_prompt_task: "Summarize the key developments overall."
focus_areas:
  - "New LLM releases"
  - "Model infrastructure news"
relevance_criteria:
  - "Contains specific technical details"
  - "Explains significance and impact"
exclusion_criteria:
  - "Pure speculation without substance"
  - "Only discusses market valuations"
summary_analysis:
  - "Trends across posts"
  - "Overall impact"
```

---

## 2. Understanding YAML Fields and Prompt Composition

Each field in the Persona YAML file plays a specific role in constructing the prompts sent to the language model. There are two primary prompts generated:

1.  **Base Item Analysis Prompt**: Used for analyzing individual RSS feed items and their comments.
2.  **Summary Prompt**: Used for generating an overall summary of multiple relevant items.

Here's how the YAML fields map to these prompts:

| Field                  | Used In Prompt      | Role in Prompt Text                                                                                                       |
|------------------------|---------------------|---------------------------------------------------------------------------------------------------------------------------|
| `Name`                 | Neither             | Internal identifier for selecting the persona via CLI flag or config.                                                       |
| `FeedURL`              | Neither             | Specifies the RSS feed URL to fetch data from.                                                                            |
| `Topic`                | Base Item Analysis  | Contextualizes relevance, used in: "...why this development matters to {{.Topic}} researchers and practitioners."         |
| `PersonaIdentity`      | Both                | Sets the core identity: "You are {{.PersonaIdentity}}". Defines the LLM's voice and expertise.                           |
| `BasePromptTask`       | Base Item Analysis  | Describes the specific task for analyzing individual items, following the `PersonaIdentity`.                                |
| `SummaryPromptTask`    | Summary             | Describes the specific task for generating the overall summary, following the `PersonaIdentity`.                            |
| `FocusAreas`           | Base Item Analysis  | Populates a bulleted list under "Relevant items include:", guiding positive topic focus.                                |
| `RelevanceCriteria`    | Base Item Analysis  | Populates a bulleted list under "An item is not relevant if...", guiding positive filtering criteria.                       |
| `ExclusionCriteria`    | Base Item Analysis  | Populates a bulleted list under "Exclude items if they match:", explicitly filtering out unwanted items.                    |
| `SummaryAnalysis`      | Summary             | Populates a bulleted list under "Your analysis should focus on:", guiding the content of the final summary.             |

Refer to `internal/prompts/prompts.go` for the exact template structures (`basePromptTemplate` and `summaryPromptTemplate`). By carefully crafting the content of each YAML field, you can precisely control the instructions given to the LLM for each persona.

---

## 3. Tips for Crafting Effective Personas

Creating a well-defined persona leads to more accurate and useful results from the LLM. Here are some tips:

*   **Be Specific in `PersonaIdentity`**: Instead of just "an expert", describe *what kind* of expert (e.g., "an AI researcher focused on hardware acceleration", "a gardening enthusiast specializing in organic pest control"). This sets the tone and knowledge base.
*   **Align Tasks with Identity**: Ensure `BasePromptTask` and `SummaryPromptTask` naturally follow from the `PersonaIdentity`. What would this specific expert be *doing* when analyzing items or summarizing?
*   **Use Action Verbs in Tasks**: Start task descriptions with clear verbs (e.g., "Analyze...", "Identify...", "Summarize...", "Compare...").
*   **Keep `FocusAreas` Concise**: List specific keywords, concepts, or types of news you want the persona to prioritize. Avoid overly broad terms.
*   **Use `RelevanceCriteria` for Inclusion**: These define the *positive* attributes that make an item relevant. Phrase them as characteristics to look for (e.g., "Provides detailed specifications", "Explains significance to the field").
*   **Use `ExclusionCriteria` for Filtering**: Define clear rules for what *should be ignored*. Phrase them as conditions for exclusion (e.g., "Is purely promotional content", "Focuses only on stock price", "Lacks any technical detail").
*   **Guide `SummaryAnalysis` Towards Insight**: Think about the *kind* of overview you want. Should it focus on trends, major breakthroughs, common problems, practical tips? List these desired outputs.
*   **Iterate and Test**: Create a persona, run it against a feed, and examine the output (especially the raw LLM output if using debug flags). Refine the YAML fields based on whether the LLM understood the instructions and produced the desired analysis.
*   **Review Prompt Templates**: Occasionally review `internal/prompts/prompts.go` to fully understand how your YAML fields are being inserted into the final instructions for the LLM.

---

## 4. Loading and Selecting Personas
At runtime, the application loads and filters personas using:

```go
// internal/persona/manager.go
func LoadAndSelect(path, personaName string) ([]persona.Persona, error)
```
- **Load**: Scans `path` for `.yaml` files and returns all parsed personas.
- **Select**: If `personaName == "all"` or empty, returns all; otherwise filters by `Name`.

By default, the directory is determined by the `ANP_PERSONAS_PATH` environment variable (mapped to `Specification.PersonasPath`). If not set, it defaults to `/app/personas/` in Docker.

---

## 5. Runtime Configuration
- **Environment Variable**: `ANP_PERSONAS_PATH` points to the personas directory.
- **CLI Flag**: `--persona` allows selecting a single persona by name or `all`.

```bash
# Use LocalLLaMA persona
go run main.go --persona=LocalLLaMA

# Process all personas
go run main.go --persona=all
```

---

## 6. Integration in Main Pipeline
In `internal/main.go`, for each selected persona:
1. Fetch RSS entries using `persona.FeedURL`.
2. Enrich entries (e.g., comments).
3. Compose the system prompt via `prompts.ComposePrompt(persona)` using `PersonaIdentity`, `BasePromptTask`, and `FocusAreas`.
4. Invoke LLM to process items.
5. Filter items by `RelevanceCriteria`.
6. Generate overall summary with `SummaryPromptTask` and `SummaryAnalysis`.
7. Render and send the email tagged with `persona.Name`.

---

## 7. Docker & Deployment
- Copy the `personas/` directory into the container image:
  ```dockerfile
  COPY personas/ /app/personas/
  ```
- Set the environment variable in deployment:
  ```yaml
  environment:
    - ANP_PERSONAS_PATH=/app/personas/
  ```
- Mount a custom directory at `/app/personas/` to override built-in personas if needed.

---