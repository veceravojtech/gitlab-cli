# Project Context

## Purpose
GitLab CLI tool for automating GitLab merge request operations including rebase, merge with retry logic, and developer activity tracking across projects.

## Tech Stack
- Go 1.25.4
- Cobra (CLI framework)
- Viper (configuration management)
- Standard library `net/http` (GitLab API client)

## Project Conventions

### Code Style
- Standard Go formatting (`gofmt`)
- Error wrapping with `fmt.Errorf("context: %w", err)`
- Lowercase error messages without trailing punctuation
- Receiver names: single letter abbreviation (e.g., `c` for `Client`)
- Package-level variables for CLI flags

### Architecture Patterns
- **Standard Go Layout**: `cmd/` for entrypoints, `internal/` for private packages
- **Layer Separation**:
  - `internal/cli/` - Command definitions and user interaction
  - `internal/gitlab/` - API client and data types
  - `internal/config/` - Configuration loading and validation
  - `internal/progress/` - Progress display utilities
- **API Client Pattern**: Single `Client` struct with typed methods per resource
- **Caching**: In-memory caches for frequently accessed data (projects, default branches, MRs)

### Testing Strategy
- Table-driven tests with `t.Run()` subtests
- Test files colocated with source (`*_test.go`)
- Run tests: `go test ./...`
- Focus on unit tests for business logic (task extraction, transformations)

### Git Workflow
- **Conventional Commits**: `type(scope): message`
  - Types: `feat`, `fix`, `test`, `docs`, `refactor`
  - Scopes: `cli`, `gitlab`, `activity`, `config`
- **Branch Naming**: `feature/TASKID-description` or `TASKID-description`
- Task IDs are 5-digit numbers (e.g., `50607`)

## Domain Context
- **GitLab API v4**: All API calls go through `/api/v4/` endpoints
- **Task IDs**: 5-digit Redmine task numbers extracted from:
  - MR titles: `#50607: Fix login bug`
  - Branch names: `feature/50607-fix-bug`
- **Activity Events**: Push, merge request, comment events from GitLab
- **MR Operations**: Rebase, merge with configurable retry and polling

## Important Constraints
- Requires `GITLAB_URL` and `GITLAB_TOKEN` environment variables or config file
- Config file location: `~/.gitlab-cli.yaml`
- API timeout default: 30 seconds per request
- No third-party GitLab client library (uses hand-rolled client)

## External Dependencies
- **GitLab API v4**: Primary external service
  - Authentication: Private token via `PRIVATE-TOKEN` header
  - Rate limiting: Handled by retry logic with configurable max retries
