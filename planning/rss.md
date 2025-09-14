 # RSS Re-introduction Planning
 
 ## Overview
 Re-introduce RSS support as a first-class citizen alongside the Reddit API. This will require extending the persona + system to support different processor types and allow personas to specify their data source (RSS feeds vs Reddit + subreddits).

## Current State Analysis

### Existing RSS Implementation (Removed)
- Previous RSS module documentation exists in `docs/rss.md`
- RSS functionality was fully implemented but has been removed from codebase
- RSS directory structure: `internal/rss/` (rss.go, processor.go, internal_funcs.go, mocks.go)
- RSS mock data remains in `feed_mocks/` directory but appears unused
 
 ### Current Reddit API Implementation
 - **Provider**: `internal/providers/reddit.go` - implements `feeds.FeedProvider` interface
 - **Interface**: `internal/feeds/processing.go` - defines `FeedProvider` interface with `FetchFeed()` and + `FetchComments()` methods
 - **Data Flow**: Reddit API → `feeds.Entry` objects → LLM processing → email generation
 
 ### Persona System Current State
 - **Configuration**: YAML-based personas in `personas/` directory
 - **Current Fields**: `name`, `subreddit`, `topic`, `persona_identity`, prompts, criteria, `comment_threshold`
 - **Processing**: Single processor type assumed (Reddit API via subreddit field)
 
 ## Required Changes
 
 ### 1. Persona System Extension
 - **Add `processor_type` field**: Specify "reddit" or "rss" as data source type
 - **Add `feed_url` field**: RSS feed URL for RSS-type personas
 - **Maintain `subreddit` field**: For Reddit-type personas
 - **Validation**: Ensure correct fields are present based on processor type
 - **Backward Compatibility**: Default to "reddit" processor type if not specified
 
### 2. RSS Provider Implementation
- **Create RSS Provider**: `internal/providers/rss.go` implementing `feeds.FeedProvider` interface
- **RSS Parsing**: Reimplement RSS XML parsing functionality
- **Mock Support**: RSS mock data support for testing
- **Comment Handling**: RSS comment feeds (where available) or fallback to no comments

### 3. Provider Factory/Selection
- **Provider Factory**: Dynamic provider selection based on persona processor type
- **Interface Consistency**: Ensure both providers implement same `feeds.FeedProvider` interface
- **Configuration**: Provider-specific configuration (Reddit API keys vs RSS HTTP settings)

### 4. Processing Pipeline Updates
- **Entry Point**: `internal/run.go` - select appropriate provider based on persona
- **Feed Processing**: `internal/feeds/processing.go` - ensure compatibility with both data sources
- **URL Extraction**: Ensure RSS entries properly extract external URLs and images
- **Error Handling**: Provider-specific error handling and logging

## 5. Testing and Validation
- **Unit Tests**: RSS provider implementation tests
- **Integration Tests**: End-to-end testing with RSS personas
- **Mock Data**: RSS mock feeds for consistent testing
- **Persona Examples**: Example RSS personas for different use cases

# Implementation Phases

## Phase 1: Core Infrastructure
 [ ] Extend persona struct with `processor_type` and `feed_url` fields
 [ ] Create provider factory/selector mechanism
 [ ] Implement basic RSS provider structure

## Phase 2: RSS Provider Implementation
 [ ] Implement RSS XML parsing functionality
 [ ] Implement RSS comment handling (where available)
 [ ] Add RSS mock data support for testing
 [ ] Create RSS provider tests

## Phase 3: Integration and Testing
 [ ] Update `internal/run.go` to use provider factory
 [ ] Create example RSS personas
 [ ] End-to-end testing with RSS feeds
 [ ] Documentation updates

## Phase 4: Validation and Polish
 [ ] Comprehensive testing of both provider types
 [ ] Error handling improvements
 [ ] Performance optimization
 [ ] Migration guide for existing setups

# Data Source Considerations

# Persona Configuration Examples

## RSS Persona
``yaml
ame: "TechNews"
rocessor_type: "rss"
eed_url: "https://feeds.ycombinator.com/frontpage.xml"
topic: "Technology News"
# ... rest of persona config
```

### Reddit Persona (Backward Compatible)
```yaml
name: "LocalLLaMA"
processor_type: "reddit"  # optional, defaults to reddit
subreddit: "localllama"
topic: "AI Technology"
# ... rest of persona config
```