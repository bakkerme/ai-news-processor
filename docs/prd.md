# Overview  
The AI News Processor is a specialized tool designed to automatically collect, filter, and summarize posts from a variety of subreddits. It addresses the challenge of information overload by filtering out low-quality content and delivering only the most valuable and engaging discussions directly to users via email. This tool is primarily targeted at AI enthusiasts, developers, and researchers who want to stay updated on the latest developments in local LLM deployments without spending hours scrolling through Reddit.

# Core Features  
## Automated RSS Feed Processing
- **What it does**: Automatically fetches and processes posts from the `/r/localllama` subreddit
- **Why it's important**: Eliminates manual monitoring and ensures no important updates are missed
- **How it works**: Uses Go's RSS parsing capabilities to fetch feed data at scheduled intervals

## Quality Filtering
- **What it does**: Filters out low-engagement posts based on configurable thresholds
- **Why it's important**: Ensures only high-value content is included in summaries
- **How it works**: Applies filtering algorithms based on metrics like comment count (configurable via `ANP_QUALITY_FILTER_THRESHOLD`)

## LLM-Powered Content Summarization
- **What it does**: Uses language models to generate concise summaries of posts and their comments
- **Why it's important**: Distills lengthy discussions into actionable insights
- **How it works**: Sends post content to an OpenAI-compatible API and processes the response using carefully crafted prompts

## Email Delivery
- **What it does**: Delivers formatted summaries directly to user's inbox
- **Why it's important**: Provides a convenient push-based delivery mechanism
- **How it works**: Uses configured SMTP settings to send HTML emails with summarized content

## Persona-Based Processing
- **What it does**: Enables customized content processing based on different interest profiles
- **Why it's important**: Allows for targeted summarization based on specific areas of interest
- **How it works**: Uses YAML-defined personas to customize content selection and processing

# User Experience  
## Key User Flows
- Persona design and selection: Write personas for targeted content processing

# Technical Architecture  
## System Components
- **RSS Feed Processor**: Go-based service for fetching and parsing subreddit feeds
- **Content Analyzer**: Interface with LLM services for content summarization
- **Email Service**: SMTP-based delivery system for sending formatted summaries
- **Persona Manager**: System for loading and applying persona-specific processing rules
- **Scheduler**: Cron-based scheduling system for automated execution

## Data Models
- **RSS Entry**: Representation of a subreddit post with metadata
- **Processed Entry**: Filtered and enriched post data with summary
- **Persona**: Configuration object defining processing parameters and interests
- **Email Template**: Structure for formatting processed content for delivery

## APIs and Integrations
- **OpenAI-compatible API**: Interface for LLM services (works with various providers)
- **SMTP Service**: For email delivery
- **Reddit RSS**: For content acquisition

## Infrastructure Requirements
- Docker environment
- Access to OpenAI-compatible LLM service
- SMTP server access for email delivery

# Development Roadmap  
## MVP Requirements (COMPLETED)
- Basic RSS feed processing from `/r/localllama`
- Simple quality filtering based on comment counts
- Integration with OpenAI-compatible API for summarization
- Email delivery of summarized content
- Configurable scheduled execution
- Docker containerization
- Persona-based processing system
- Improved prompt engineering for better summaries
- Support for image processing with dedicated models
- Performance benchmarking for LLM outputs

## Future Enhancements
- Summarise linked web pages
- Filter out already sent IDs
- Use Wake On LAN to switch on LLM server before run
- Advanced filtering options beyond comment counts
- User feedback mechanism to improve summarization quality

# Logical Dependency Chain
## Foundation Components (COMPLETED)
1. Persona system implementation
1. RSS feed acquisition and parsing
1. Basic content filtering
1. LLM integration for summarization
1. Image processing capabilities
1. Email delivery system
1. Benchmarking system for quality assessment
1. Scheduled execution

## Future Development Path
- Summarise linked web pages
- Filter out already sent IDs
- Use Wake On LAN to switch on LLM server before run
- Advanced filtering options beyond comment counts
- User feedback mechanism to improve summarization quality

# Risks and Mitigations  
## Technical Challenges
- **Open Weights LLM Quality and Prompt Design**: Implement benchmarking to allow for a feedback loop
- **LLM Structure Reliability**: Implement retry mechanisms and fallback options
- **Email Delivery Issues**: Add logging and failure notifications
- **Accessing External Sites without Defined APIs**: Careful use of HTTP requests, appropriate retries and timesout

## MVP Scope Management
- **Feature Creep**: Strictly prioritize features based on core functionality
- **Performance Concerns**: Include benchmarking tools to identify bottlenecks
- **Resource Utilization**: Optimize LLM usage to minimize processing time and costs

## Resource Constraints
- **API Cost Management**: Implement efficient batching and caching
- **Testing Complexity**: Mock systems for RSS and LLM to facilitate fast development without external dependecies

# Appendix  
## Research Findings
- Local LLMs have shown sufficient capability for content summarization
- Qwen2.5 7B Instruct Q8, Qwen2.5 32B Instruct Q4, and Qwen3 models perform well with the current prompt design
- Comment count is an effective initial quality filter but could be enhanced with additional metrics

## Technical Specifications
- **Language**: Go
- **Containerization**: Docker
- **Configuration**: Environment variables
- **Scheduling**: Cron-based (configurable)
- **LLM Requirements**: OpenAI-compatible API
- **Minimum Comment Threshold**: Default 10 (configurable) 