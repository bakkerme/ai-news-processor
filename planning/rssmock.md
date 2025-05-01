# Plan for RSS Feed Debug Dump

## Overview
Add a runtime debug flag (via an environment variable) that dumps RSS feed XML to disk under `rss/mocks/{persona}` for debugging and testing purposes. The file name will be derived from the feed URL.

## Goals
- Introduce environment variable `ANP_DEBUG_RSS_DUMP` through the Specification feature, following existing debug flag patterns
- Keep RSS dumping logic close to existing RSS processing in main.go
- Write raw XML to disk in `rss/mocks/{persona}/{feedName}.rss` before processing
- Use the `os` package for file operations (avoid deprecated `ioutil`)

## Implementation Steps

1. **Update Specification**
   - Add `DebugRssDump bool` to the Specification struct in `internal/specification/specification.go`
   - Add `env:"ANP_DEBUG_RSS_DUMP"` tag following existing patterns
   - Update GetConfig() validation if needed

2. **Integrate with RSS Processing**
   - Add dump logic after `rss.FetchAndProcessFeed` call in main.go
   - Use existing persona context from the persona loop
   - Leverage existing error handling patterns

3. **Hook into the feed loop**
   ```go
   // After fetching feed content, before processing
   if s.DebugRssDump {
       if err := dumpRSS(persona.FeedURL, rawContent, persona.Name); err != nil {
           fmt.Printf("Warning: Failed to dump RSS feed: %v\n", err)
       }
   }
   ```

4. **Error handling**
   - Follow existing error pattern of using fmt.Printf for non-critical errors
   - Continue processing even if dump fails
   - Use existing logging approach

## File Changes

- `internal/specification/specification.go`:
   ```go
   type Specification struct {
       // ... existing fields ...
       DebugRssDump bool `env:"ANP_DEBUG_RSS_DUMP" default:"false"`
   }
   ```

- `main.go`:
   ```go
   // Add after existing debug flag checks
   if s.DebugRssDump {
       if err := dumpRSS(persona.FeedURL, rawContent, persona.Name); err != nil {
           fmt.Printf("Warning: Failed to dump RSS feed: %v\n", err)
       }
   }

   func dumpRSS(feedURL, content, personaName string) error {
       feedName := sanitizeURL(feedURL)
       dir := filepath.Join("rss", "mocks", personaName)
       if err := os.MkdirAll(dir, 0755); err != nil {
           return fmt.Errorf("failed to create directory: %w", err)
       }
       path := filepath.Join(dir, feedName+".rss")
       return os.WriteFile(path, []byte(content), 0644)
   }
   ```

## Integration Notes

- Works with existing `DebugMockRss` flag
- Preserves current error handling patterns
- Uses existing persona context
- Follows established debug flag naming convention

## Persona-specific Mock RSS Loading

To enable loading mock RSS feeds per persona, update the mock code and processor to use `feed_mocks/rss/{personaName}`:

1. Modify `internal/rss/mocks.go`:
   - Change `func ReturnFakeRSS() string` to `func ReturnFakeRSS(personaName string) string` that reads from `feed_mocks/rss/{personaName}/{personaName}.rss`.
   - Change `func ReturnFakeCommentRSS(id string) string` to `func ReturnFakeCommentRSS(personaName, id string) string` that reads from `feed_mocks/rss/{personaName}/{id}.rss`.

2. Update `internal/rss/processor.go`:
   - Replace `ReturnFakeRSS()` with `ReturnFakeRSS(personaName)` in the mock branch.
   - Replace `ReturnFakeCommentRSS(id)` with `ReturnFakeCommentRSS(personaName, id)` when loading comment mocks.

3. Directory Structure:
   - Ensure mock files are stored under `feed_mocks/rss/{personaName}` with filenames `{personaName}.rss` and `{postID}.rss` for comments.

---

This plan aligns with the codebase's:
1. Use of Specification for environment variables
2. Debug flag patterns
3. Error handling approach
4. Persona management
5. File operation patterns