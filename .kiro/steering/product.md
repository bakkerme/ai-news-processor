# AI News Processor

AI News Processor is a specialized tool that automatically collects, filters, and summarizes posts from subreddits (primarily `/r/localllama`) using LLM-powered analysis. It addresses information overload by filtering low-quality content and delivering valuable discussions via email.

## Core Purpose
- **Automated Content Curation**: Fetches and processes Reddit posts at scheduled intervals
- **Quality Filtering**: Filters posts based on engagement metrics (comment count thresholds)
- **LLM-Powered Summarization**: Uses OpenAI-compatible APIs to generate concise summaries
- **Persona-Based Processing**: Customizable content processing based on different interest profiles
- **Email Delivery**: Sends formatted HTML summaries directly to users' inboxes

## Key Features
- Multi-persona support with YAML configuration
- Image processing with dedicated models
- External URL content summarization
- Quality filtering and relevance scoring
- Benchmarking system for LLM performance
- Docker containerization with cron scheduling
- Debug modes for development and testing

## Target Users
AI enthusiasts, developers, and researchers who want curated updates on local LLM developments without manual Reddit monitoring.