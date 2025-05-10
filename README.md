# AI News Processor

AI News Processor is a tool designed to summarise posts from any subreddit (mainly tested with `/r/localllama`), attached images and links and the comments, filter out unworthy posts and email you a summary. You will need to provide an LLM accessible via an OpenAI-compatible API and SMTP details for an email service.

The prompt was built and optimised for Qwen3 30B MoE for text summarisation, and Gemma 3 27B Instruct IT QAT for image summarisation.

## Repository Overview
- **Directory Structure**:
  - `internal/`: Go service
  - `build/`: Used as part of the Docker container for production.
  - `bench/`: LLM output benchmarking.
  - `docs/`: Project documentation, including [CI workflows](docs/ci.md).
 
## Configuration

The following environment variables are used to configure the AI News Processor:

| Environment Variable          | Description                                  | Default Value      |
|-------------------------------|----------------------------------------------|--------------------|
| `ANP_LLM_URL`                 | The URL of the LLM (Language Model) service. Must be OpenAI-compatible. |                    |
| `ANP_LLM_API_KEY`             | The API key for authenticating with the LLM. |                    |
| `ANP_LLM_MODEL`               | The language model to use for analysis.      |                    |
| `ANP_LLM_URL_SUMMARY_ENABLED` | If true, enables summarizing content from external URLs found in feed items. | `true`             |
| `ANP_LLM_IMAGE_ENABLED`       | If true, enables separate image processing with a dedicated model. | false              |
| `ANP_LLM_IMAGE_MODEL`         | The dedicated model to use for image processing. Only used when ANP_LLM_IMAGE_ENABLED is true. |  |
| `ANP_EMAIL_TO`                | Email address to send email to.      |                    |
| `ANP_EMAIL_FROM`              | Email address to send email from.    |                    |
| `ANP_EMAIL_HOST`              | SMTP server host for emails.                 |                    |
| `ANP_EMAIL_PORT`              | SMTP server port for emails.                 |                    |
| `ANP_EMAIL_USERNAME`          | Username for the email account.              |                    |
| `ANP_EMAIL_PASSWORD`          | Password for the email account.              |                    |
| `ANP_CRON_SCHEDULE`           | Cron schedule for running the processor.     | `0 0 * * *` (Midnight) |
| `ANP_PERSONAS_PATH`           | Directory containing persona YAML files.     | `/app/personas/`   |
| `ANP_QUALITY_FILTER_THRESHOLD`| Minimum number of comments required for a post to be included. | `10` |

### Debug Configuration

The following environment variables are used for debugging purposes, The Mock RSS and LLM won't currently work in the docker container.

| Environment Variable             | Description                                               | Default Value |
|----------------------------------|-----------------------------------------------------------|---------------|
| `ANP_DEBUG_MOCK_RSS`             | Use mock RSS data instead of fetching real feeds.         | `false`       |
| `ANP_DEBUG_MOCK_LLM`             | Use mock LLM responses instead of using LLM completion.   | `false`       |
| `ANP_DEBUG_SKIP_EMAIL`           | Skip sending email notifications during processing.       | `false`       |
| `ANP_DEBUG_OUTPUT_BENCHMARK`     | Output benchmark data for LLM performance benchmarking.   | `false`       |
| `ANP_DEBUG_MAX_ENTRIES`          | Limit the number of entries processed (0 = no limit).     | `0`           |
| `ANP_DEBUG_SKIP_CRON`            | If true, skips cron setup and runs main directly.         | `false`       |

## Personas System

- Each persona is defined in a YAML file in the `personas/` directory at the project root.
- At runtime, set the environment variable `ANP_PERSONAS_PATH` to the directory containing persona YAML files (default: `/app/personas/` in Docker).
- To select a persona at runtime, use the CLI flag `--persona=NAME` or `--persona=all` to process all personas.
- To add a new persona, create a new YAML file in the `personas/` directory with the required fields (see examples in `planning/persona.md`).

Example CLI usage:

```sh
go run main.go --persona=LocalLLaMA
go run main.go --persona=all
```

## Getting Started

### Prerequisites

- [Go](https://golang.org/doc/install) installed
- [Docker](https://www.docker.com/) (optional, for containerization)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/bakkerme/ai-news-processor.git
   cd ai-news-processor
   ```

2. Build the project:
   ```bash
    touch .envrc.local
   ```
    Set environment variabls as per above.
   
   ```bash
   go run ./internal
   ```

There are also built docker images in the packages section.
