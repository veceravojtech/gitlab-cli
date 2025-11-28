# GitLab CLI

A command-line tool for managing GitLab merge requests with automated rebase and merge workflows.

## Features

- **List merge requests** - Filter by project, assignee, or approval status
- **View MR details** - Quick summary or full JSON output
- **Rebase MRs** - Trigger and wait for rebase completion
- **Merge with auto-rebase** - Automatically rebase and retry when needed
- **CI-aware merging** - Waits for pipelines to complete before merging
- **Progress feedback** - Animated status updates during long operations

## Installation

Download the latest binary for your platform from [Releases](https://github.com/user/gitlab-cli/releases).

### Linux

```bash
curl -L https://github.com/user/gitlab-cli/releases/latest/download/gitlab-cli-linux-amd64 -o gitlab-cli
chmod +x gitlab-cli
sudo mv gitlab-cli /usr/local/bin/
```

### macOS

```bash
curl -L https://github.com/user/gitlab-cli/releases/latest/download/gitlab-cli-darwin-amd64 -o gitlab-cli
chmod +x gitlab-cli
sudo mv gitlab-cli /usr/local/bin/
```

## Configuration

Create a configuration file at `~/.gitlab-cli.yaml`:

```yaml
gitlab_url: https://gitlab.example.com
gitlab_token: your-personal-access-token

# Optional defaults
defaults:
  max_retries: 3
  timeout: 5m
  poll_interval: 5s
```

### Getting a GitLab Token

1. Go to GitLab → User Settings → Access Tokens
2. Create a token with `api` scope
3. Copy the token to your config file

You can also specify a config file location with the `--config` flag.

## Quick Reference

| Command | Description | Key Flags |
|---------|-------------|-----------|
| `mr list` | List open merge requests | `--project`, `--mine`, `--approved` |
| `mr show <id>` | Show MR details | `--json` |
| `mr rebase <id>` | Rebase a merge request | `--no-wait` |
| `mr merge <id>` | Merge a merge request | `--auto-rebase`, `--max-retries`, `--timeout` |

### Flag Details

| Flag | Command | Description |
|------|---------|-------------|
| `--project <id>` | list | Filter by project ID |
| `--mine` | list | Only MRs assigned to me |
| `--approved` | list | Only approved MRs |
| `--json` | show | Output as JSON |
| `--no-wait` | rebase | Don't wait for rebase completion |
| `--auto-rebase` | merge | Automatically rebase if needed |
| `--max-retries <n>` | merge | Max rebase attempts (default: 3) |
| `--timeout <duration>` | merge | Overall timeout (default: 5m) |

## Examples

### List all open MRs assigned to me

```bash
gitlab-cli mr list --mine
```

### List approved MRs in a specific project

```bash
gitlab-cli mr list --project 123 --approved
```

### View MR details

```bash
gitlab-cli mr show 456

# Output as JSON for scripting
gitlab-cli mr show 456 --json
```

### Rebase an MR

```bash
gitlab-cli mr rebase 456

# Trigger rebase without waiting
gitlab-cli mr rebase 456 --no-wait
```

### Merge an MR with automatic rebase

```bash
# Simple merge (fails if rebase needed)
gitlab-cli mr merge 456

# Auto-rebase and retry up to 3 times
gitlab-cli mr merge 456 --auto-rebase

# Custom retry limit and timeout
gitlab-cli mr merge 456 --auto-rebase --max-retries 5 --timeout 10m
```

The merge command automatically waits for CI pipelines to complete and shows live progress updates.

## Development

### Building from Source

```bash
git clone https://github.com/user/gitlab-cli.git
cd gitlab-cli
go build -o gitlab-cli ./cmd/gitlab-cli
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
├── cmd/gitlab-cli/     # Application entrypoint
├── internal/
│   ├── cli/            # Cobra commands and flag handling
│   ├── config/         # Configuration loading and validation
│   ├── gitlab/         # GitLab API client
│   └── progress/       # Animated progress output
├── .gitlab-cli.yaml.example
└── go.mod
```

### Adding New Commands

1. Add command definition in `internal/cli/`
2. Implement API methods in `internal/gitlab/`
3. Register the command in `init()`
4. Update the README quick reference table

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/my-feature`)
3. Make your changes
4. Run tests (`go test ./...`)
5. Commit with a descriptive message
6. Open a pull request

For bug reports and feature requests, please open an issue.

## License

MIT License - see [LICENSE](LICENSE) for details.
