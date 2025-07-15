# Project Structure

## Root Level
- `main.go` - Entry point, delegates to `internal.Run()`
- `go.mod/go.sum` - Go module dependencies
- `Makefile` - Build automation and common commands
- `Dockerfile` - Multi-stage Docker build
- `docker-compose.yml` - Container orchestration
- `.env.example` - Environment variable template

## Core Directories

### `/internal` - Main Application Code
- `run.go` - Main application orchestration and pipeline
- `mocks.go` - Mock data for testing/development

#### `/internal/providers` - Data Source Providers
- `reddit.go` - Reddit API integration (primary provider)
- `mock.go` - Mock provider for testing
- `dump.go` - Data dumping utilities
- `funcs.go` - Shared provider utilities

#### `/internal/llm` - LLM Processing
- `llm.go` - Main LLM processor with retry logic
- `processor_types.go` - Type definitions for processor
- `chatcomplete.go` - Chat completion implementations
- `summary.go` - Summary generation logic
- `*_test.go` - Unit tests

#### `/internal/feeds` - Feed Processing
- `processing.go` - Feed processing pipeline
- `types.go` - Feed data structures

#### `/internal/email` - Email System
- `service.go` - Email service orchestration
- `email.go` - Email client implementation
- `render.go` - Email template rendering
- `/templates/` - Email HTML templates

#### `/internal/persona` - Persona Management
- `manager.go` - Persona loading and selection
- `persona.go` - Persona data structures
- `persona_test.go` - Persona tests

#### `/internal/prompts` - LLM Prompt Management
- `prompts.go` - Prompt templates and composition
- `schema_generator.go` - JSON schema generation
- `models_integration.go` - Model integration utilities

### `/personas` - Persona Configurations
- `*.yaml` - Individual persona definitions
- Each persona defines subreddit, prompts, and filtering criteria

### `/models` - Data Models
- `item.go` - Core item data structure
- `run_data.go` - Benchmark and run data models

### `/docs` - Documentation
- `prd.md` - Product requirements
- `persona.md` - Persona system documentation
- `email.md` - Email system documentation
- `rss.md` - RSS processing documentation

### `/feed_mocks` - Mock Data
- `/reddit/` - Mock Reddit feed data organized by subreddit
- Used for testing and development without API calls

## Code Organization Patterns

### Package Structure
- Each major feature has its own package under `/internal`
- Interfaces defined alongside implementations
- Test files co-located with source files
- Mock implementations separate from production code

### Configuration
- Environment variables centralized in `internal/specification`
- Persona configs in YAML format under `/personas`
- Debug flags prefixed with `ANP_DEBUG_`

### Error Handling
- Custom error types in `internal/customerrors`
- Retry logic with exponential backoff
- Graceful degradation for non-critical failures

### Testing Strategy
- Unit tests for core logic (`*_test.go`)
- Mock providers for external dependencies
- Benchmark data collection for LLM performance
- Test data organized under `/testdata` subdirectories