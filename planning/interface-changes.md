# Interface-Based Design Changes for Improved Testability

This document outlines proposed interface-based design changes to the AI News Processor codebase, focusing on enhancing unit testability by decoupling modules. The core idea is to define interfaces for the dependencies of each module, allowing real dependencies to be replaced with mock implementations during testing.

## Proposed Interfaces

### 2. RSS Feed Fetching (`rss` module)

*   **Problem:** `main` (and potentially other modules like `llm` if it re-fetches data) depends directly on the `rss` module's concrete fetching and processing logic.
*   **Proposal:** Define a `FeedProvider` interface in the `rss` package.

    ```go
    package rss

    import (
        "context"
        // Assuming Item struct is defined in a common package
        "github.com/your_org/ai-news-processor/internal/common" // Adjust import path as needed
    )

    // FeedProvider defines the interface for fetching and potentially processing RSS feed data.
    type FeedProvider interface {
        // FetchFeed retrieves items from a given feed URL.
        // Consider whether this should return raw feed items or already enriched common.Item structs.
        FetchFeed(ctx context.Context, url string) ([]common.Item, error)
        // Potentially add methods for fetching comments if that's a distinct operation.
        // FetchComments(ctx context.Context, item common.Item) (string, error)
    }
    ```
*   **Usage:** Inject a `FeedProvider` into `main` (or other consumers). The existing `rss` logic (fetching, parsing, enrichment) would be encapsulated within a struct that implements this interface.
*   **Testing:** Mock `FeedProvider` in `main`'s tests to provide controlled feed data (slices of `common.Item`) without actual network calls or parsing complexities.

### 3. Email Sending (`email` module)

*   **Problem:** `main` calls the `email` module directly to render templates and send emails, making it hard to test the main flow without involving email infrastructure.
*   **Proposal:** Define an `EmailSender` interface in the `email` package.

    ```go
    package email

    import "context"

    // EmailSender defines the interface for sending email notifications.
    type EmailSender interface {
        SendNewsletter(ctx context.Context, recipient string, subject string, bodyHTML string) error
        // Add other email types if necessary
    }
    ```
*   **Usage:** Inject an `EmailSender` into `main`. The current email logic (template rendering, SMTP interaction) would be part of a struct implementing this interface.
*   **Testing:** Provide a mock `EmailSender` in `main`'s tests to verify that email sending is *attempted* with the correct parameters (recipient, subject, rendered body), without actually sending any emails.

### 4. Configuration Loading (`specification` module)

*   **Problem:** `main` depends on the specific mechanism within the `specification` package to load configuration (currently from environment variables).
*   **Proposal:** Define a `ConfigLoader` interface, perhaps within `specification` or a dedicated `config` package.

    ```go
    package specification // or config

    import "context"

    // AppConfig should represent the fully loaded and validated application configuration structure.
    // type AppConfig struct { ... }

    // ConfigLoader defines the interface for loading application configuration.
    type ConfigLoader interface {
        Load(ctx context.Context) (*AppConfig, error) // AppConfig is the target struct
    }
    ```
*   **Usage:** Inject `ConfigLoader` into `main`. The current logic reading environment variables would be wrapped in a struct implementing this interface. This also supports the potential refactoring mentioned in `refactor.md` to allow different config sources (files, flags).
*   **Testing:** Mock `ConfigLoader` to provide various `AppConfig` instances for testing different application behaviors driven by configuration, without manipulating environment variables.

### 5. Persona Management (`persona` module)

*   **Problem:** `main` depends on the concrete `persona` loading logic (reading files from the `personas/` directory).
*   **Proposal:** Define a `PersonaProvider` interface in the `persona` package.

    ```go
    package persona

    import (
        "context"
        // Assuming Persona struct is defined in a common package
        "github.com/your_org/ai-news-processor/internal/common" // Adjust import path as needed
    )

    // PersonaProvider defines the interface for accessing persona configurations.
    type PersonaProvider interface {
        GetPersona(ctx context.Context, name string) (*common.Persona, error)
        ListPersonas(ctx context.Context) ([]string, error) // Useful for validation or selection logic
    }
    ```
*   **Usage:** Inject `PersonaProvider` into `main`. The file-reading logic would be part of a struct implementing this interface.
*   **Testing:** Mock `PersonaProvider` to simulate different persona selections and configurations without needing actual persona files on disk.

## General Approach

*   **Dependency Injection (DI):** Use constructor injection as the primary method. Functions or structs that require dependencies (like `main`, `llm`, `summary`) should accept these dependencies as interface types in their constructor or factory function signatures.
*   **Instantiation / Wiring:** The `main` function (or a dedicated initialization function called by `main`) will be responsible for creating the concrete instances of each service (e.g., `openai.NewClient`, `rss.NewFeedFetcher`, `email.NewSMTPSender`, etc.) and injecting them into the components that require them.
*   **`common` Package Interaction:** Introducing interfaces helps decouple modules from the *behavior* of their dependencies. While refactoring the `common` package itself (as noted in `refactor.md`) is a separate task, using interfaces for services interacting *with* common types (like `FeedProvider` returning `[]common.Item`) is a key part of this strategy. Avoid interfaces that simply mirror complex structs unless there's a clear benefit.

By implementing these interfaces, we create well-defined boundaries, reduce tight coupling, and significantly improve the ability to write isolated, fast unit tests for each module. 