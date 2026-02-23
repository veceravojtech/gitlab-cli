# Development Guide - gitlab-cli

## Prerequisites

| Requirement | Version | Notes |
|------------|---------|-------|
| Go | 1.25+ | Required for building |
| Git | any | Source control |
| GitLab account | - | For testing |
| GitLab token | - | API scope required |

## Quick Start

### 1. Clone Repository
```bash
git clone https://github.com/veceravojtech/gitlab-cli.git
cd gitlab-cli
```

### 2. Install Dependencies
```bash
go mod download
```

### 3. Build
```bash
go build -o gitlab-cli ./cmd/gitlab-cli
```

### 4. Configure
```bash
# Option A: Environment variables
export GITLAB_URL="https://gitlab.example.com"
export GITLAB_TOKEN="your-token-here"

# Option B: Config file
cp .gitlab-cli.yaml.example ~/.gitlab-cli.yaml
# Edit ~/.gitlab-cli.yaml with your values
```

### 5. Verify
```bash
./gitlab-cli version
./gitlab-cli mr list --mine
```

## Development Workflow

### Building

```bash
# Simple build
go build -o gitlab-cli ./cmd/gitlab-cli

# Build with version info
VERSION="1.0.0"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
go build -ldflags "-X github.com/user/gitlab-cli/internal/cli.Version=${VERSION} -X github.com/user/gitlab-cli/internal/cli.BuildTime=${BUILD_TIME}" -o gitlab-cli ./cmd/gitlab-cli
```

### Running Tests

```bash
# All tests
go test ./...

# Verbose output
go test -v ./...

# Specific package
go test -v ./internal/cli/...

# With coverage
go test -cover ./...

# Coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Linting

```bash
# Install golangci-lint if not present
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Run linter
golangci-lint run
```

## Project Structure

```
gitlab-cli/
├── cmd/gitlab-cli/main.go    # Entry point
├── internal/
│   ├── cli/                  # Commands (add new commands here)
│   ├── config/               # Configuration loading
│   ├── gitlab/               # API client (add new API methods here)
│   └── progress/             # Console output utilities
├── go.mod                    # Dependencies
└── install.sh                # Installation script
```

## Adding New Features

### Adding a New Command

1. **Create command file** in `internal/cli/`:

```go
// internal/cli/mycommand.go
package cli

import (
    "github.com/spf13/cobra"
    "github.com/user/gitlab-cli/internal/config"
    "github.com/user/gitlab-cli/internal/gitlab"
)

var myCmd = &cobra.Command{
    Use:   "mycommand",
    Short: "Description of my command",
    RunE:  runMyCommand,
}

func init() {
    rootCmd.AddCommand(myCmd)
    // Add flags here
    myCmd.Flags().StringVar(&someFlag, "flag", "", "flag description")
}

func runMyCommand(cmd *cobra.Command, args []string) error {
    cfg, err := config.Load(cfgFile)
    if err != nil {
        return err
    }

    client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)

    // Implementation
    return nil
}
```

### Adding a New API Method

1. **Add types** to `internal/gitlab/types.go` if needed:

```go
type NewResource struct {
    ID   int    `json:"id"`
    Name string `json:"name"`
}
```

2. **Add method** to appropriate file in `internal/gitlab/`:

```go
// internal/gitlab/newresource.go
package gitlab

import "fmt"

func (c *Client) GetNewResource(id int) (*NewResource, error) {
    path := fmt.Sprintf("/new_resources/%d", id)

    var resource NewResource
    if err := c.get(path, &resource); err != nil {
        return nil, fmt.Errorf("getting resource: %w", err)
    }

    return &resource, nil
}
```

### Adding Subcommands

```go
var parentCmd = &cobra.Command{
    Use:   "parent",
    Short: "Parent command",
}

var childCmd = &cobra.Command{
    Use:   "child",
    Short: "Child command",
    RunE:  runChild,
}

func init() {
    rootCmd.AddCommand(parentCmd)
    parentCmd.AddCommand(childCmd)
}
```

## Configuration Options

### Config File (`~/.gitlab-cli.yaml`)

```yaml
gitlab_url: https://gitlab.example.com
gitlab_token: glpat-xxxxxxxxxxxx

defaults:
  max_retries: 3        # Max rebase retry attempts
  timeout: 5m           # Operation timeout
  poll_interval: 5s     # Status check interval
```

### Environment Variables

| Variable | Description |
|----------|-------------|
| `GITLAB_URL` | GitLab instance URL |
| `GITLAB_TOKEN` | Personal access token |
| `GITLAB_CLI_MAX_RETRIES` | Override max retries |
| `GITLAB_CLI_TIMEOUT` | Override timeout |

## Common Development Tasks

### Test a Specific MR Operation
```bash
# List your MRs
./gitlab-cli mr list --mine

# Show MR details
./gitlab-cli mr show 12345

# Test rebase (use --no-wait to skip waiting)
./gitlab-cli mr rebase 12345 --no-wait
```

### Debug API Calls
Add verbose logging by modifying `client.go`:
```go
func (c *Client) doRequest(method, path string, body io.Reader) (*http.Response, error) {
    url := fmt.Sprintf("%s/api/v4%s", c.baseURL, path)
    log.Printf("API: %s %s", method, url)  // Add this
    // ...
}
```

### Test Config Loading
```bash
# Show loaded config
./gitlab-cli config show

# Test with specific config file
./gitlab-cli --config /path/to/config.yaml config show
```

## Code Conventions

### Error Handling
```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}
```

### Flag Naming
- Use kebab-case: `--auto-rebase`, `--no-wait`
- Boolean flags: prefer positive names (`--json` not `--no-text`)

### Output Formatting
- Use `tabwriter` for tabular output
- Use `encoding/json` with indent for JSON output
- Use `progress.Writer` for long-running operations

## Troubleshooting

### Build Errors

**"package not found"**
```bash
go mod tidy
go mod download
```

**"undefined: ..."**
- Check imports in the file
- Ensure new files are in correct package

### Runtime Errors

**"config validation failed"**
- Check `GITLAB_URL` and `GITLAB_TOKEN` are set
- Verify token has API scope

**"API error (status 401)"**
- Token is invalid or expired
- Generate new token in GitLab

**"API error (status 404)"**
- Resource doesn't exist
- Check project/MR IDs

## Release Process

1. Update version in `install.sh`
2. Run tests: `go test ./...`
3. Build: `go build -ldflags "..." -o gitlab-cli ./cmd/gitlab-cli`
4. Tag release: `git tag v1.x.x`
5. Push: `git push origin v1.x.x`

---

*Generated by document-project workflow - 2025-12-16*
