# Architecture Document - gitlab-cli

## Executive Summary

gitlab-cli is a command-line tool for managing GitLab merge requests with automated rebase and merge workflows. It provides a streamlined interface for common MR operations, activity tracking, and CI-aware merging.

## Technology Stack

| Category | Technology | Version | Justification |
|----------|------------|---------|---------------|
| Language | Go | 1.25.4 | Strong typing, single binary distribution, excellent CLI ecosystem |
| CLI Framework | Cobra | v1.10.1 | Industry-standard, supports subcommands, auto-generated help |
| Configuration | Viper | v1.21.0 | Unified config from files, env vars, defaults |
| HTTP | net/http | stdlib | Native HTTP client, no external dependencies |
| JSON | encoding/json | stdlib | API serialization |

## Architecture Pattern

**Pattern**: Command Pattern + Layered Architecture

The application follows a clean separation of concerns with four distinct layers:

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                        │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                  cmd/gitlab-cli                      │    │
│  │                    main.go                           │    │
│  │              (Application Bootstrap)                 │    │
│  └─────────────────────────┬───────────────────────────┘    │
│                            │                                 │
│  ┌─────────────────────────▼───────────────────────────┐    │
│  │                  internal/cli                        │    │
│  │           (Cobra Commands & Flag Handling)           │    │
│  │                                                      │    │
│  │  root.go ─── mr.go ─── activity.go ─── project.go   │    │
│  │              config.go ─── label.go ─── user.go     │    │
│  └─────────────────────────┬───────────────────────────┘    │
└─────────────────────────────┼───────────────────────────────┘
                              │
┌─────────────────────────────▼───────────────────────────────┐
│                    BUSINESS LAYER                            │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐    │
│  │                 internal/gitlab                      │    │
│  │              (API Client & Business Logic)           │    │
│  │                                                      │    │
│  │  client.go ─── types.go                             │    │
│  │  mr.go ─── activity.go ─── project.go               │    │
│  │  label.go ─── user.go                               │    │
│  └─────────────────────────────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                   INFRASTRUCTURE LAYER                       │
│                                                              │
│  ┌───────────────────┐    ┌────────────────────────────┐    │
│  │  internal/config  │    │    internal/progress       │    │
│  │                   │    │                            │    │
│  │  - Viper config   │    │  - Animated spinners       │    │
│  │  - Env vars       │    │  - Elapsed time            │    │
│  │  - File loading   │    │  - Status updates          │    │
│  └───────────────────┘    └────────────────────────────┘    │
└─────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────┐
│                    EXTERNAL SYSTEMS                          │
│                                                              │
│            GitLab API v4  ◄─── HTTPS + JSON                 │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Single Binary Distribution
- Go compiles to a single static binary
- No runtime dependencies
- Cross-platform builds (Linux, macOS, Windows)

### 2. GitLab API v4 Client
- Custom HTTP client (not using gitlab.com/go-gitlab SDK)
- Minimal dependencies
- Only implements required endpoints
- Authentication via `PRIVATE-TOKEN` header

### 3. Configuration Hierarchy
```
Priority (highest to lowest):
1. Command-line flags
2. Environment variables (GITLAB_URL, GITLAB_TOKEN)
3. Config file (~/.gitlab-cli.yaml)
4. Built-in defaults
```

### 4. Polling-Based Operations
- Rebase and merge operations poll for completion
- Configurable poll interval (default: 5s)
- Configurable timeout (default: 5m)
- Animated progress feedback during waits

## Component Details

### CLI Layer (`internal/cli/`)

| File | Lines | Responsibility |
|------|-------|----------------|
| mr.go | ~850 | MR commands: list, show, rebase, merge, create, label, reviewer |
| activity.go | ~530 | Activity log with task extraction, grouping, multiple output formats |
| root.go | ~30 | Root command, global flags |
| project.go | ~90 | Project listing |
| user.go | ~90 | User listing and search |
| label.go | ~85 | Label listing |
| config.go | ~85 | Config init/show |
| version.go | ~25 | Version display |

### GitLab API Layer (`internal/gitlab/`)

| File | Responsibility |
|------|----------------|
| client.go | HTTP client, authentication, request helpers |
| types.go | All API type definitions (~200 lines) |
| mr.go | MR operations: list, get, rebase, merge, update |
| activity.go | Events API, project/commit fetching |
| project.go | Project listing and lookup |
| user.go | User search and resolution |
| label.go | Project labels |

### Key Types

```go
type MergeRequest struct {
    ID, IID, ProjectID    int
    Title, State          string
    SourceBranch          string
    TargetBranch          string
    DetailedMergeStatus   string
    HeadPipeline          *Pipeline
    // ... more fields
}

type Config struct {
    GitLabURL    string
    GitLabToken  string
    MaxRetries   int
    Timeout      time.Duration
    PollInterval time.Duration
}
```

## Data Flow

### Merge Request with Auto-Rebase

```
User: gitlab-cli mr merge 123 --auto-rebase
        │
        ▼
┌─────────────────┐
│ Load Config     │ ◄── ~/.gitlab-cli.yaml + env vars
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Create Client   │ ◄── gitlab.NewClient(url, token)
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Get MR Details  │ ◄── GET /api/v4/merge_requests/{id}
└────────┬────────┘
         │
         ▼
┌─────────────────┐     ┌──────────────────┐
│ Check Status    │────►│ need_rebase?     │
└────────┬────────┘     └────────┬─────────┘
         │                       │ yes
         │                       ▼
         │              ┌──────────────────┐
         │              │ Trigger Rebase   │
         │              │ PUT .../rebase   │
         │              └────────┬─────────┘
         │                       │
         │                       ▼
         │              ┌──────────────────┐
         │              │ Poll until done  │◄──┐
         │              └────────┬─────────┘   │
         │                       │             │ sleep 5s
         │                       └─────────────┘
         ▼
┌─────────────────┐
│ Check Pipeline  │ ◄── GET .../pipelines/{id}/jobs
└────────┬────────┘
         │ success
         ▼
┌─────────────────┐
│ Merge           │ ◄── PUT .../merge
└────────┬────────┘
         │
         ▼
    ✓ Success
```

## Testing Strategy

### Unit Tests
- `config_test.go` - Environment variable loading
- `client_test.go` - Client initialization
- `activity_test.go` - Task extraction from branches/titles

### Test Commands
```bash
go test ./...                    # Run all tests
go test -v ./internal/cli/...    # Verbose CLI tests
go test -cover ./...             # With coverage
```

## Security Considerations

1. **Token Storage**: Token stored in config file (0600 permissions recommended)
2. **Token Display**: `config show` truncates token display
3. **HTTPS**: All API calls use HTTPS
4. **No Secrets in Code**: Configuration externalized

## Build & Distribution

### Build Command
```bash
go build -ldflags "-X .../cli.Version=1.0.0" -o gitlab-cli ./cmd/gitlab-cli
```

### Installation Options
1. `install.sh` - Builds and installs to `~/.local/bin`
2. Manual download from releases
3. Build from source

## Extension Points

### Adding New Commands
1. Create file in `internal/cli/`
2. Define Cobra command
3. Register in `init()` with `rootCmd.AddCommand()`
4. Implement API calls in `internal/gitlab/` if needed

### Adding New API Endpoints
1. Add types to `internal/gitlab/types.go`
2. Create method on `Client` struct
3. Use existing `get`/`post`/`put` helpers

---

*Generated by document-project workflow - 2025-12-16*
