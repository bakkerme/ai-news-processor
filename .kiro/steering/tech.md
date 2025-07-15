# Technology Stack

## Core Technologies
- **Language**: Go 1.23.7
- **Build System**: Go modules with Makefile
- **Containerization**: Docker with multi-stage builds
- **Deployment**: Docker Compose
- **Scheduling**: Cron-based execution

## Key Dependencies
- **LLM Integration**: OpenAI Go SDK (`github.com/openai/openai-go`)
- **Web Scraping**: go-readability for content extraction
- **Reddit API**: go-reddit/v2 for Reddit API integration
- **Configuration**: godotenv for environment variables
- **YAML Processing**: gopkg.in/yaml.v3 for persona configuration
- **JSON Schema**: invopop/jsonschema for structured data validation
- **HTTP Client**: Built-in net/http with custom retry logic
- **Testing**: testify for unit tests

## Architecture Patterns
- **Dependency Injection**: Services are initialized with their dependencies
- **Interface-Based Design**: Core components use interfaces (FeedProvider, ArticleExtractor, etc.)
- **Retry Pattern**: Built-in retry mechanisms with exponential backoff
- **Pipeline Processing**: Multi-phase processing (image → URL → text summarization)
- **Configuration-Driven**: Environment variables and YAML personas drive behavior

## Common Commands

### Development
```bash
# Run with specific persona
make run                    # Uses LocalLLaMa persona
go run main.go --persona=LocalLLaMa

# Run all personas
make dev-all               # Process all personas
go run main.go --persona=all

# Run directly (bypasses persona selection)
make dev-direct
go run ./internal

# Run benchmarking
make run-benchmark
```

### Building & Testing
```bash
# Build applications
make build

# Run tests
make test
make test-verbose

# Clean build artifacts
make clean

# Dependency management
make deps
make deps-update
```

### Docker
```bash
# Build and run with Docker Compose
docker-compose up --build

# Build Docker image
docker build -t ai-news-processor .
```

## Environment Configuration
- Use `.env` file for local development
- Copy `.env.example` to `.env` and configure
- All config prefixed with `ANP_` (AI News Processor)
- Debug flags available for development (`ANP_DEBUG_*`)
- Persona path configurable via `ANP_PERSONAS_PATH`