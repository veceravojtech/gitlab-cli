package mcp

import (
	"fmt"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

// GitLabClient defines all gitlab.Client methods used by MCP tool handlers.
type GitLabClient interface {
	ListMRs(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error)
	GetMR(projectID, iid int) (*gitlab.MergeRequest, error)
	GetMRByGlobalID(id int) (*gitlab.MergeRequest, error)
	RebaseMR(projectID, iid int) error
	MergeMR(projectID, iid int) error
	CreateMR(projectID string, opts gitlab.CreateMROptions) (*gitlab.MergeRequest, error)
	UpdateMRLabels(projectID, iid int, labels []string) (*gitlab.MergeRequest, error)
	UpdateMRReviewers(projectID, iid int, reviewerIDs []int) (*gitlab.MergeRequest, error)
	SetAutoMerge(projectID, iid int) error
	CancelAutoMerge(projectID, iid int) error
	GetMRDiscussions(projectID, iid int) ([]gitlab.Discussion, error)
	GetMRApprovals(projectID, iid int) (*gitlab.ApprovalState, error)
	GetMRPipelines(projectID, mrIID int) ([]gitlab.PipelineInfo, error)
	GetEvents(opts gitlab.ListEventsOptions) ([]gitlab.Event, error)
	ListProjects(opts gitlab.ListProjectsOptions) ([]gitlab.Project, error)
	ListUsers(opts gitlab.ListUsersOptions) ([]gitlab.User, error)
	ListProjectMembers(projectID string, search string) ([]gitlab.User, error)
	ListProjectLabels(projectID string, search string) ([]gitlab.Label, error)
}

// Server holds the MCP server state.
type Server struct {
	client GitLabClient
	config *config.Config
}

// NewServer creates a new MCP server with config loaded from environment/file.
func NewServer() (*Server, error) {
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigLoad, err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrConfigValidate, err)
	}

	client := gitlab.NewClient(cfg.GitLabURL, cfg.GitLabToken)
	return &Server{client: client, config: cfg}, nil
}

// NewServerWithClient creates a server with an injected client (for testing).
func NewServerWithClient(client GitLabClient, cfg *config.Config) *Server {
	if cfg == nil {
		cfg = &config.Config{}
	}
	return &Server{client: client, config: cfg}
}

// RegisterTools registers all 13 MCP tools on the SDK server.
func (s *Server) RegisterTools(sdkServer *sdkmcp.Server) {
	falseVal := false

	// Read-only tools
	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-list",
		Description: "List open merge requests with optional filters (mine, approved, project)",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRListHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-show",
		Description: "Show merge request details including discussions and approvals",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRShowHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "activity-list",
		Description: "List recent GitLab activity events for the authenticated user",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.ActivityListHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "project-list",
		Description: "List GitLab projects with optional search filter",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.ProjectListHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "user-list",
		Description: "List GitLab users globally or within a project",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.UserListHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "label-list",
		Description: "List labels for a project",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.LabelListHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "config-show",
		Description: "Show current gitlab-cli configuration (token masked)",
		Annotations: &sdkmcp.ToolAnnotations{
			ReadOnlyHint:    true,
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.ConfigShowHandler)

	// Idempotent write tools
	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-rebase",
		Description: "Trigger a rebase for a merge request and optionally wait for completion",
		Annotations: &sdkmcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRRebaseHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-label",
		Description: "Add or remove labels on a merge request",
		Annotations: &sdkmcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRLabelHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-reviewer",
		Description: "Add or remove reviewers on a merge request by user ID",
		Annotations: &sdkmcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRReviewerHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-auto-merge",
		Description: "Enable or cancel merge-when-pipeline-succeeds on a merge request",
		Annotations: &sdkmcp.ToolAnnotations{
			IdempotentHint:  true,
			DestructiveHint: &falseVal,
		},
	}, s.MRAutoMergeHandler)

	// Non-idempotent tools
	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-create",
		Description: "Create a new merge request with optional labels and reviewers",
	}, s.MRCreateHandler)

	sdkmcp.AddTool(sdkServer, &sdkmcp.Tool{
		Name:        "mr-merge",
		Description: "Merge a merge request with optional auto-rebase and retry logic",
	}, s.MRMergeHandler)
}
