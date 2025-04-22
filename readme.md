# AI News Processor

AI News Processor is a tool designed to process posts from the `/r/localllama` subreddit, their comments and email you a summary. You will need to provide an LLM accessible via an OpenAI-compatible API and SMTP details for an email service.

The prompt was built and tested with Qwen2.5 7B Instruct Q8 and Qwen2.5 32B Instruct Q4. 

## Repository Overview
- **Directory Structure**:
  - `internal/`: Go service
  - `build/`: Used as part of the Docker container for production.
  - `bench/`: LLM output benchmarking.
 
## Configuration

The following environment variables are used to configure the AI News Processor:

| Environment Variable          | Description                                  | Default Value      |
|-------------------------------|----------------------------------------------|--------------------|
| `ANP_LLM_URL`                 | The URL of the LLM (Language Model) service. Must be OpenAI-compatible. |                    |
| `ANP_LLM_API_KEY`             | The API key for authenticating with the LLM. |                    |
| `ANP_LLM_MODEL`               | The language model to use for analysis.      |                    |
| `ANP_EMAIL_TO`                | Email address to send email to.      |                    |
| `ANP_EMAIL_FROM`              | Email address to send email from.    |                    |
| `ANP_EMAIL_HOST`              | SMTP server host for emails.                 |                    |
| `ANP_EMAIL_PORT`              | SMTP server port for emails.                 |                    |
| `ANP_EMAIL_USERNAME`          | Username for the email account.              |                    |
| `ANP_EMAIL_PASSWORD`          | Password for the email account.              |                    |
| `ANP_CRON_SCHEDULE`           | Cron schedule for running the processor.     | `0 0 * * *` (Midnight) |

### Debug Configuration

The following environment variables are used for debugging purposes, The Mock RSS and LLM won't currently work in the docker container.

| Environment Variable             | Description                                               | Default Value |
|----------------------------------|-----------------------------------------------------------|---------------|
| `ANP_DEBUG_MOCK_RSS`             | Use mock RSS data instead of fetching real feeds.         | `false`       |
| `ANP_DEBUG_MOCK_LLM`             | Use mock LLM responses instead of querying the LLM.       | `false`       |
| `ANP_DEBUG_MOCK_SKIP_EMAIL`      | Skip sending email notifications during processing.       | `false`       |
| `ANP_DEBUG_OUTPUT_BENCHMARK`     | Output benchmark data for LLM performance benchmarking.          | `false`       |

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
