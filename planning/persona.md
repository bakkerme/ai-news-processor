# Persona System Architectural Plan

## Overview

The Persona system allows the AI News Processor to process any Reddit RSS feed (or other RSS feeds) by defining configurable personas. Each persona specifies the feed URL, prompt details, and other custom logic. The system supports selecting one or more personas at runtime.

---

## 1. Persona Struct

A Persona represents a configuration for a specific feed and its processing logic.

```go
// internal/common/persona.go

type Persona struct {
    Name              string   `yaml:"name"`                // Unique name for the persona (e.g., "LocalLLaMA")
    FeedURL           string   `yaml:"feed_url"`            // URL of the RSS feed (e.g., "https://reddit.com/r/localllama.rss")
    Topic             string   `yaml:"topic"`               // Main subject area (e.g., "AI Technology", "Gardening")
    PromptIntro       string   `yaml:"prompt_intro"`        // Persona's "voice" and role description
    FocusAreas        []string `yaml:"focus_areas"`         // List of topics/keywords to prioritize
    RelevanceCriteria []string `yaml:"relevance_criteria"`  // List of criteria for relevance analysis
    // Add more fields as needed (filters, etc.)
}
```

---

## 2. Persona Configuration (File per Persona)

Personas are defined in individual YAML files within the `personas/` directory (see Section 10). Each file represents one persona.

**Example (`personas/localllama.yaml`):**

```yaml
name: "LocalLLaMA"
feed_url: "https://reddit.com/r/localllama.rss"
topic: "AI Technology and Large Language Models"
prompt_intro: "You are an AI researcher and enthusiast who loves diving deep into technical details. Your job is to process data feeds and identify interesting developments in AI technology, providing detailed and engaging summaries."
focus_areas:
  - "New LLM models, runners or other infrastructure being released or open sourced"
  - "Big AI lab news (OpenAI, Anthropic, etc.)"
  - "Security news"
relevance_criteria:
  - "Describes the technical details and specifications"
  - "Explains the significance of the development"
  - "Highlights any novel approaches or techniques"
```

**Example (`personas/gardening.yaml`):**

```yaml
name: "GardeningGuru"
feed_url: "https://reddit.com/r/gardening.rss"
topic: "Gardening and Horticulture"
prompt_intro: "You are a passionate gardening expert who loves sharing practical advice and plant science."
focus_areas:
  - "Innovative gardening techniques"
  - "Plant care and pest management"
  - "Community gardening projects"
relevance_criteria:
  - "Provides actionable gardening tips"
  - "Highlights new plant varieties or tools"
```

---

## 3. Persona Loader

- Implement a loader to read all YAML files from the directory specified by `ANP_PERSONAS_PATH`.
- Parse each file into a `Persona` struct.
- Aggregate the loaded personas into a list or map.
- Loader can live in `internal/common` or a new `internal/persona` package.

---

## 4. Runtime Persona Selection

- Allow the user to specify which persona(s) to use at runtime (via CLI flag or ENV variable):
  - `--persona=LocalLLaMA` or `--persona=all`
- If "all", iterate over all loaded personas.

---

## 5. Refactor RSS Fetching and Processing

- Use the `FeedURL` from the selected persona instead of a hardcoded URL.
- Use the persona's prompt and settings for processing and output.

---

## 6. Output Handling

- Output files/emails should be named or tagged with the persona name for clarity and to avoid collisions.

---

## 7. Flow Summary

1. **Startup**: Load all personas from config.
2. **Persona Selection**: Determine which persona(s) to run (via CLI/ENV).
3. **For each persona**:
    - Fetch the RSS feed using `FeedURL`.
    - Process entries using persona-specific prompt/settings.
    - Output results (e.g., email, summary) using persona's configuration.

---

## 8. CLI Example

```sh
go run main.go --persona=LocalLLaMA
go run main.go --persona=all
```

---

## 9. Required Changes Checklist

- [ ] Define `Persona` struct and config file.
- [ ] Implement config loader.
- [ ] Refactor RSS fetching and processing to use persona data.
- [ ] Add CLI/ENV support for persona selection.
- [ ] Update output logic to handle multiple personas.

---

## 10. Docker & Deployment Strategy

### Personas Folder and Docker Image

- **Personas Directory:**  
  Store each persona configuration as a **separate YAML file** in a `personas/` directory at the project root (e.g., `personas/localllama.yaml`, `personas/gardening.yaml`).

- **Docker Image:**  
  The entire `personas/` directory is copied into the Docker image at build time (e.g., to `/app/personas/`).

- **Environment Variable for Config Path:**  
  At runtime, the service uses an environment variable (e.g., `ANP_PERSONAS_PATH`) to determine where to load persona files from.  
  Example:  
  ```sh
  ANP_PERSONAS_PATH=/app/personas/
  ```

- **How it works:**  
  - By default, the service loads all persona YAML files from the directory specified by `ANP_PERSONAS_PATH`.
  - You can override the built-in personas by mounting a different folder containing YAML files at runtime if needed.

- **Sample Dockerfile snippet:**
  ```dockerfile
  COPY personas/ /app/personas/
  ```

- **Sample Docker Compose/Portainer environment variable:**
  ```yaml
  environment:
    - ANP_PERSONAS_PATH=/app/personas/
  ```

- **Benefits:**  
  - All persona configs are versioned with the codebase.
  - Easy to add, remove, or update personas.
  - Works well with Docker, Docker Compose, and Portainer UI (just set the environment variable).

---

## 11. Updated Required Changes Checklist

- [ ] Define `Persona` struct (including prompt fields) and YAML format.
- [ ] Implement config loader to read all YAML files from the `ANP_PERSONAS_PATH` directory.
- [ ] Refactor RSS fetching and processing to use persona data (FeedURL, Topic, etc.).
- [ ] Implement prompt composition using Go templates and persona data.
- [ ] Add CLI/ENV support for selecting which persona(s) to run (`--persona=NAME` or `--persona=all`).
- [ ] Update output logic to handle multiple personas (e.g., tag output with persona name).
- [ ] Update Dockerfile to copy the `personas/` directory into the image.
- [ ] Document the use of the `ANP_PERSONAS_PATH` environment variable and persona selection flags.

---

## 12. Prompt System Specification

### Overview

The prompt system is responsible for generating the instructions given to the language model for each persona. It supports both generic and persona-specific prompt components, allowing for flexible, maintainable, and extensible prompt composition.

---

### 12.1. Prompt Composition Model

- **Base Prompt Template:**  
  A Go template (or similar) that defines the generic structure, formatting rules, and output requirements for all personas. It uses placeholders for persona-specific data.
- **Persona-Specific Fields (from YAML):**  
  Each persona YAML file defines fields that customize the prompt:
  - `topic`: Main subject area (inserted into template).
  - `prompt_intro`: The persona's "voice" and role description (inserted into template).
  - `focus_areas`: List of topics or types of content to prioritize (iterated in template).
  - `relevance_criteria`: List of criteria for analysis (iterated in template).

---

### 12.2. Runtime Prompt Generation

1. **Load persona YAML** from the `ANP_PERSONAS_PATH` directory and parse fields into the `Persona` struct.
2. **Select the desired persona(s)** based on runtime configuration (e.g., `--persona` flag).
3. **For each selected persona**, fill the base prompt template with its specific values (`Topic`, `PromptIntro`, etc.).
4. **Use the composed prompt** for all LLM interactions for that persona.

---

### 12.3. Extensibility

- New personas can be added by creating new YAML files in the `personas/` directory.
- The base template can be extended to support new fields if needed.
- Supports different base templates for different tasks (e.g., single-item analysis vs. overall summary).

---

### 12.4. Implementation Checklist

- [ ] Update `Persona` struct in Go code to include all YAML fields (`Name`, `FeedURL`, `Topic`, `PromptIntro`, `FocusAreas`, `RelevanceCriteria`).
- [ ] Update loader to parse these fields from individual YAML files in the specified directory.
- [ ] Implement prompt composition logic using Go templates and the loaded `Persona` data.
- [ ] Document how to add new personas (create YAML file) and customize prompts.

---

### 12.5. Benefits

- **Flexible:** Each persona can have a unique "voice", topic, and focus.
- **Maintainable:** Generic prompt logic is centralized in the base template.
- **Extensible:** Easy to add new personas or prompt fields without code changes (just YAML).
- **Topic-Agnostic:** Supports any subject matter defined in the persona.

--- 