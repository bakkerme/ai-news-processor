# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Running the Application
```bash
# Run with specific persona
go run main.go --persona=LocalLLaMA

# Run with all personas  
go run main.go --persona=all

# Run directly (bypasses cron scheduling)
go run ./internal
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run specific package tests
go test ./internal/llm
go test ./internal/rss
```

### Build and Dependencies
```bash
# Build the application
go build -o ai-news-processor main.go

# Download and verify dependencies
go mod download
go mod verify

# Update dependencies
go mod tidy
```

## Architecture Overview

This is a Go-based AI news processor that:
1. Fetches RSS feeds from Reddit subreddits
2. Processes content through LLMs for summarization
3. Filters content based on quality metrics
4. Sends processed summaries via email

### Core Components

**Entry Point**: `main.go` → `internal/run.go`
- Single entry point that delegates to `internal.Run()`
- Configuration loaded via environment variables with `ANP_` prefix

**Configuration**: `internal/specification/`
- All config via environment variables using `envconfig` library
- Validation in `specification.go:43-95`
- Debug modes available for RSS, LLM, and email

**Processing Pipeline**:
1. **RSS Processing** (`internal/rss/`): Fetch feeds and comments from Reddit
2. **Quality Filtering** (`internal/qualityfilter/`): Filter by comment count threshold  
3. **LLM Processing** (`internal/llm/`): Summarize content via OpenAI-compatible API
4. **Content Extraction** (`internal/contentextractor/`): Extract article content from URLs
5. **Email Generation** (`internal/email/`): Render and send summary emails

**Persona System** (`internal/persona/`):
- YAML-based persona configurations in `personas/` directory
- Each persona defines feed URL, prompts, and processing preferences
- Supports processing single persona or all personas

**Mock/Debug System**:
- Mock RSS data in `feed_mocks/` 
- Mock LLM responses via `DebugMockLLM` flag
- Benchmarking output via `DebugOutputBenchmark` flag

### Key Data Flow

1. Load persona configs from YAML files
2. Fetch RSS feed using `FeedProvider` interface (real or mock)
3. Extract URLs and fetch additional content
4. Process through LLM with persona-specific prompts (creates `models.Item` objects)
5. Generate overall summary using `Item.Summary` and `Item.CommentSummary` fields (not raw RSS data)
6. Render email template and send via SMTP

### Summary Generation Optimization

The final summary generation uses processed `Item` summaries instead of raw RSS data to reduce token usage:
- **Input**: `[]models.Item` with `Summary` and `CommentSummary` fields
- **Format**: Structured text with ID, Title, Summary, and Comment Summary via `Item.ToSummaryString()`
- **Benefits**: Significantly reduced token count for large jobs
- **Location**: `internal/llm/summary.go:GenerateSummary()` and `internal/llm/llm.go:generateSummaryWithRetry()`

**Item String Formatting**:
- `models.Item.ToSummaryString()`: Method that generates structured summary text
- **Output Format**: 
  ```
  ID: [item_id]
  Title: [item_title]
  Summary: [item_summary]
  Comment Summary: [comment_summary] (only if present)
  ```
- **Tests**: Comprehensive test coverage in `models/item_test.go`

### Testing Patterns

- Test files use `*_test.go` naming convention
- Mocking via interfaces (e.g., `FeedProvider`, `OpenAIClient`)
- Test data in `internal/*/testdata/` directories
- Mock implementations in dedicated files (e.g., `mockprovider.go`)

### Important Interfaces

**FeedProvider** (`internal/rss/feedprovider.go`): RSS feed fetching
**OpenAIClient** (`internal/openai/`): LLM interaction  
**ArticleExtractor** (`internal/contentextractor/`): Content extraction
**Fetcher** (`internal/fetcher/`): HTTP fetching with retry logic

### Common Issues & Fixes

**Channel Deadlock in LLM Calls**: 
- **Issue**: `chatCompletionImageSummary` and `chatCompletionForWebSummary` were calling `client.ChatCompletion` synchronously but expecting async channel behavior
- **Fix**: Added `go` keyword before `client.ChatCompletion` calls to run them in goroutines
- **Location**: `internal/llm/chatcomplete.go:74` and `internal/llm/chatcomplete.go:99`
- **Symptoms**: "fatal error: all goroutines are asleep - deadlock!" during image processing

### Dependency Injection

The main processor (`internal/llm/processor_types.go`) receives all dependencies via constructor injection, enabling easy testing and mocking.

### Environment Configuration

All configuration via `ANP_` prefixed environment variables. Key settings:
- `ANP_LLM_URL`, `ANP_LLM_API_KEY`, `ANP_LLM_MODEL`: LLM configuration
- `ANP_EMAIL_*`: SMTP email settings  
- `ANP_DEBUG_*`: Debug and testing flags
- `ANP_PERSONAS_PATH`: Path to persona YAML files

### Error Tracking and Auditability

The system includes comprehensive error tracking in the audit data:

**Error Types Tracked**:
- `json_parsing_error`: LLM response parsing failures
- `llm_timeout`: LLM request timeouts
- `network_error`: Network connectivity issues
- `rate_limit_error`: API rate limiting
- `auth_error`: Authentication failures
- `summary_generation_error`: Overall summary generation failures

**Audit Data Structure** (`models/run_data.go`):
- `ProcessingError`: Detailed error information with type, message, retry count, and raw response
- Success/failure tracking for entries, images, and web content
- Error breakdown statistics by error type
- Success rates for different processing phases

**Error Capture Points**:
- Entry text processing (`internal/llm/llm.go:144-180`)
- Image processing (`internal/llm/llm.go:92-125`) 
- Web content processing (`internal/llm/llm.go:284-396`)
- Overall summary generation (`internal/run.go:188-206`)

Working directory expectation: Project root (not `./internal`)