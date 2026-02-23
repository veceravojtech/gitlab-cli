---
project_name: 'gitlab-cli'
user_name: 'Vojta'
date: '2025-12-18'
sections_completed: ['technology_stack', 'language_rules', 'framework_rules', 'testing_rules', 'code_quality', 'workflow_rules', 'critical_rules']
status: 'complete'
rule_count: 54
optimized_for_llm: true
---

# Project Context for AI Agents

_This file contains critical rules and patterns that AI agents must follow when implementing code in this project. Focus on unobvious details that agents might otherwise miss._

---

## Technology Stack & Versions

- **Go 1.25.4** - Use standard library where possible (net/http, encoding/json)
- **Cobra v1.10.1** - CLI framework, define commands as `var xxxCmd = &cobra.Command{}`
- **Viper v1.21.0** - Config hierarchy: flags > env vars > config file > defaults
- **No external GitLab SDK** - Custom HTTP client in `internal/gitlab/client.go`

## Critical Implementation Rules

### Go Language Rules

- **Error wrapping**: Always use `fmt.Errorf("context: %w", err)` - never return bare errors
- **Defer for cleanup**: Use `defer resp.Body.Close()` immediately after checking response error
- **Nil checks before access**: Check `if mr.HeadPipeline != nil` before accessing pipeline fields
- **Time parsing**: Use `time.RFC3339` for GitLab API timestamps, `time.Parse("2006-01-02", date)` for dates
- **HTTP client timeout**: Always set timeout on http.Client (default: 30s in this project)
- **Struct tags**: Use `json:"snake_case"` matching GitLab API field names exactly
- **Package-level vars for flags**: Define flag variables at package level, bind in `init()`

### Cobra CLI Framework Rules

- **Command naming**: Parent commands use noun (`mr`, `activity`), subcommands use verb (`list`, `show`, `create`)
- **Command variables**: Define as `var xxxCmd = &cobra.Command{}` at package level
- **Handler pattern**: Use `RunE` (not `Run`) with `func runXxxYyy(cmd *cobra.Command, args []string) error`
- **Flag variables**: Package-level variables (e.g., `var listProject int`), not local to functions
- **Flag registration**: All in `init()` function, use `Flags()` for command-specific, `PersistentFlags()` for inherited
- **Required flags**: Use `cmd.MarkFlagRequired("flagname")` after flag definition
- **Args validation**: Use `Args: cobra.ExactArgs(1)` for commands requiring arguments
- **Subcommand registration**: Register in `init()` with `parentCmd.AddCommand(childCmd)`

### Testing Rules

- **ALWAYS use TDD**: Write tests first, then implement - red-green-refactor cycle
- **Test file location**: Same directory as source file (`xxx_test.go` alongside `xxx.go`)
- **Package name**: Use same package (e.g., `package config`), not `package config_test`
- **Test function naming**: `TestXxxYyy(t *testing.T)` - describe what's being tested
- **Environment setup/teardown**: Use `os.Setenv()` with `defer os.Unsetenv()` for cleanup
- **Error checking**: Use `t.Fatalf()` for fatal errors, `t.Errorf()` for non-fatal assertions
- **No external test frameworks**: Use standard `testing` package only
- **Run tests**: `go test ./...` for all, `go test -v ./internal/...` for verbose

### Code Quality & Style Rules

- **Package structure**: Layered architecture - `cli/` (presentation), `gitlab/` (business), `config/` + `progress/` (infrastructure)
- **File naming**: `snake_case.go` for multi-word files (e.g., `mr_auto_merge.go`)
- **One command per file**: Each CLI command gets its own file in `internal/cli/`
- **Types in types.go**: All GitLab API structs go in `internal/gitlab/types.go`
- **No comments on obvious code**: Only add comments for non-obvious logic
- **Output formatting**: Use `tabwriter` for tables, `encoding/json` with `SetIndent("", "  ")` for JSON
- **String truncation**: Use helper like `truncate(s, maxLen)` for table column display
- **Constants for magic numbers**: Define constants like `commitDateFetchThreshold = 10`

### Development Workflow Rules

- **Commit message format**: `type(scope): description` - types: `feat`, `fix`, `refactor`, `docs`, `test`
- **Scope**: Use package name (`cli`, `gitlab`, `config`, `progress`)
- **Branch naming**: `{task-id}-description` or `feature/{task-id}-description` (5-digit Redmine IDs)
- **Task ID extraction**: Code expects `#12345` in titles or `12345-` prefix in branch names
- **Build command**: `go build -o gitlab-cli ./cmd/gitlab-cli`
- **Install**: `install.sh` builds and copies to `~/.local/bin`
- **No generated files in git**: Don't commit build artifacts

### Critical Don't-Miss Rules

**API Client Rules:**
- **Never use go-gitlab SDK**: This project uses a custom minimal client - don't add external GitLab libraries
- **API path format**: Always prefix with `/api/v4` in client methods (handled by `doRequest`)
- **Authentication header**: Use `PRIVATE-TOKEN`, not `Authorization: Bearer`
- **URL trailing slash**: Client trims trailing slashes - don't add them to paths

**GitLab API Gotchas:**
- **MR IID vs ID**: Use `IID` (project-scoped) for API calls, `ID` (global) only for cross-project lookups
- **detailed_merge_status can lag**: Check `HeadPipeline.Status` directly when waiting for CI
- **Rebase is async**: Must poll `RebaseInProgress` field until false after triggering rebase

**CLI Output Rules:**
- **Progress writer is mutex-protected**: Always use `prog.StopWait()` before `prog.Action()` or `prog.Success()`
- **Don't mix stdout formats**: A command outputs either table OR JSON, never both
- **Exit codes**: Return error from `RunE` to get non-zero exit - don't call `os.Exit()` directly

**Configuration Rules:**
- **Environment variables**: `GITLAB_URL` and `GITLAB_TOKEN` (no prefix), but `GITLAB_CLI_*` for other settings
- **Config validation**: Always call `cfg.Validate()` after `config.Load()` before using config values

---

## Usage Guidelines

**For AI Agents:**
- Read this file before implementing any code
- Follow ALL rules exactly as documented
- When in doubt, prefer the more restrictive option
- Update this file if new patterns emerge

**For Humans:**
- Keep this file lean and focused on agent needs
- Update when technology stack changes
- Review quarterly for outdated rules
- Remove rules that become obvious over time

---

_Last Updated: 2025-12-18_
