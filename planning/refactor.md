# AI News Processor - Module Dependencies Analysis

This document analyzes the dependencies between modules to help plan refactoring efforts.

## Core Module Dependencies

### Main (`main.go`)
- Depends on: `specification`, `common`, `rss`, `llm`, `openai`, `persona`, `prompts`, `summary`, `email`
- Role: Orchestrates the entire application flow

### Common (`internal/common/`)
- Depended on by: Most other modules
- Contains: 
  - Core data structures (`item.go`)
  - Retry mechanisms (`retry.go`)
  - Persona definitions (`persona.go`)
  - Benchmarking utilities (`benchmark.go`)
- Note: Acts as a central dependency for most modules

### RSS (`internal/rss/`)
- Depends on: `common`
- Depended on by: `main`, `llm`
- Role: Fetches and processes RSS feeds, enriches entries with comments

### LLM (`internal/llm/`)
- Depends on: `common`, `openai`, `rss`
- Depended on by: `main`
- Role: Processes entries and generates structured outputs using LLM

### OpenAI (`internal/openai/`)
- Depends on: `common`
- Depended on by: `llm`, `main`
- Role: Handles communication with OpenAI-compatible APIs

### Summary (`internal/summary/`)
- Depends on: `common`, `openai`, `rss`
- Depended on by: `main`
- Role: Generates summaries for content using LLM

### Email (`internal/email/`)
- Depends on: `common`
- Depended on by: `main`
- Role: Renders and sends email newsletters

### Persona (`internal/persona/`)
- Depends on: `common`
- Depended on by: `main`
- Role: Manages persona selection and loading

### Specification (`internal/specification/`)
- Depended on by: `main`
- Role: Handles application configuration loading and validation

### Prompts (`internal/prompts/`)
- Depends on: `common` (likely)
- Depended on by: `main`
- Role: Manages system prompts for LLM interactions

## Dependency Graph

```
                                  +-------------+
                                  |    main     |
                                  +------+------+
                                         |
                                         v
              +-------------+------+-----+------+---------+---------+
              |             |      |            |         |         |
              v             v      v            v         v         v
       +------+----+ +------+--+ ++-----+ +-----+--+ +----+---+ +---+----+
       |specification| |persona | | rss  | |  llm   | |summary | | email  |
       +-------------+ +----+---+ +--+---+ +---+----+ +----+---+ +--------+
                           |         |         |           |
                           v         v         v           v
                        +--+---------+---------+-----------+----+
                        |              common                   |
                        +----------------------------------------+
                                        ^
                                        |
                                  +-----+------+
                                  |   openai   |
                                  +------------+
```

## Potential Refactoring Strategies

1. **Reduce Common Module Coupling**:
   - Split `common` into more specific packages to reduce interdependencies
   - Move domain-specific types into their respective modules

User Notes: I agree with this assessment, common is currently a dumping ground for random types.

2. **Interface-Based Design**:
   - Define clear interfaces for module boundaries
   - Implement dependency injection for better testability

User Notes: Agreed.

3. **Context Propagation**:
   - Add consistent context handling for better control flow and cancellation

User Notes: Please provide more detail on what context handling does.

**Context Handling Details:**
- In Go, `context.Context` provides a standardized way to carry deadlines, cancellation signals, and request-scoped values across API boundaries
- Currently, context is used inconsistently in the codebase (e.g., in RSS's `fetchWithRetry` but not in other network operations)
- Benefits of consistent context usage:
  - Graceful cancellation of long-running operations (e.g., LLM calls)
  - Propagation of request-specific values (e.g., tracing IDs, timeouts)
  - Improved concurrency control for parallel processing
- Implementation would involve passing context from `main.go` through all operations that involve I/O or significant processing

4. **Error Handling Standardization**:
   - Implement consistent error handling patterns across modules
   - Add structured logging

User Notes: Please provide some technical details on what's wrong with error handling currently. Logging should be transitioned from fmt based to log based.

**Error Handling Technical Details:**
- Current issues:
  - Inconsistent error wrapping (sometimes using `fmt.Errorf(...: %w", err)`, sometimes not)
  - Direct `panic()` calls in `main.go` instead of returning errors and handling gracefully
  - Mix of returning errors and printing them with `fmt.Printf`
  - Lack of error categorization (e.g., distinguishing between temporary network failures vs. permanent configuration errors)
- Proposed improvements:
  - Adopt a consistent error handling package like `pkg/errors` or `golang.org/x/xerrors`
  - Create domain-specific error types for better error categorization and handling
  - Replace `fmt.Printf` with a structured logging library (e.g., `zap`, `logrus`, or standard `log`)
  - Implement centralized error handling in `main.go` instead of in-line panic calls

5. **Configuration Management**:
   - Refactor the `specification` pattern to support more flexible configuration sources

User Notes: What other configuration sources would be useful here? I would like other programatic options for running the software, so I'll consider other options like this.

**Additional Configuration Sources:**
- Command-line flags (beyond just `--persona`)
- Configuration files (YAML/JSON/TOML) for better organization of complex settings
- Secrets management integration (e.g., Hashicorp Vault, AWS Secrets Manager) for sensitive credentials (User Note: this is an overkill for this project)
- Service discovery for dynamic configuration in containerized environments (User Note: Like what?)
- Programmatic API to allow embedding the processor in other applications (User Note: Intesesting idea but invalid for our usecase)
- Hybrid approach that prioritizes sources (e.g., CLI flags override env vars, which override config files) (User Note: Seems reasonable)
- Hot-reloading of certain configuration parameters without full restart (User Note: Overkill)

6. **Module Independence**:
   - Reduce tight coupling between modules
   - Ensure each module has a clear, single responsibility 

User Notes: Agreed. I think the split between OpenAI and LLM is a useful model. OpenAI is just the API, LLM unites it with internal data types and operates it.

## Implementation Priority

Suggested order for implementing these refactoring strategies:

1. Error Handling Standardization - Provides immediate benefits with minimal structural changes
2. Context Propagation - Builds foundation for better concurrency and cancellation
3. Interface-Based Design - Defines clear boundaries between modules
4. Module Independence - Reorganizes code along the newly defined boundaries
5. Reduce Common Module Coupling - Breaks down the central dependency
6. Configuration Management - Takes advantage of cleaner architecture for more flexible configuration