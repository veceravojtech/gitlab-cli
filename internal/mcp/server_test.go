package mcp

import (
	"errors"
	"os"
	"testing"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name      string
		envURL    string
		envToken  string
		wantErr   bool
		errTarget error
	}{
		{
			name:      "missing both URL and token",
			envURL:    "",
			envToken:  "",
			wantErr:   true,
			errTarget: ErrConfigValidate,
		},
		{
			name:      "missing token",
			envURL:    "https://gitlab.example.com",
			envToken:  "",
			wantErr:   true,
			errTarget: ErrConfigValidate,
		},
		{
			name:      "missing URL",
			envURL:    "",
			envToken:  "glpat-test123",
			wantErr:   true,
			errTarget: ErrConfigValidate,
		},
		{
			name:     "valid config",
			envURL:   "https://gitlab.example.com",
			envToken: "glpat-test123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Setenv automatically saves/restores and calls t.Parallel()-safe cleanup
			t.Setenv("HOME", t.TempDir())

			if tt.envURL != "" {
				t.Setenv("GITLAB_URL", tt.envURL)
			} else {
				t.Setenv("GITLAB_URL", "")
				os.Unsetenv("GITLAB_URL")
			}
			if tt.envToken != "" {
				t.Setenv("GITLAB_TOKEN", tt.envToken)
			} else {
				t.Setenv("GITLAB_TOKEN", "")
				os.Unsetenv("GITLAB_TOKEN")
			}

			srv, err := NewServer()
			if (err != nil) != tt.wantErr {
				t.Fatalf("NewServer() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
					t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.errTarget)
				}
				return
			}

			if srv == nil {
				t.Fatal("server is nil")
			}
			if srv.client == nil {
				t.Error("client is nil")
			}
			if srv.config == nil {
				t.Error("config is nil")
			}
		})
	}
}

func TestNewServerWithClient(t *testing.T) {
	mock := &mockGitLabClient{}
	cfg := &config.Config{
		GitLabURL:    "https://test.example.com",
		GitLabToken:  "test-token",
		MaxRetries:   3,
		Timeout:      5 * time.Minute,
		PollInterval: 5 * time.Second,
	}

	srv := NewServerWithClient(mock, cfg)
	if srv == nil {
		t.Fatal("server is nil")
	}
	if srv.client != mock {
		t.Error("client is not the injected mock")
	}
	if srv.config != cfg {
		t.Error("config is not the injected config")
	}
}

func TestRegisterTools(t *testing.T) {
	mock := &mockGitLabClient{}
	cfg := &config.Config{
		GitLabURL:    "https://test.example.com",
		GitLabToken:  "test-token",
		MaxRetries:   3,
		Timeout:      5 * time.Minute,
		PollInterval: 5 * time.Second,
	}

	srv := NewServerWithClient(mock, cfg)
	sdkServer := sdkmcp.NewServer(&sdkmcp.Implementation{
		Name:    "test-server",
		Version: "1.0.0",
	}, nil)

	// Should not panic
	srv.RegisterTools(sdkServer)
}

// mockGitLabClient implements GitLabClient for testing
type mockGitLabClient struct {
	listMRsFunc            func(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error)
	getMRFunc              func(projectID, iid int) (*gitlab.MergeRequest, error)
	getMRByGlobalIDFunc    func(id int) (*gitlab.MergeRequest, error)
	rebaseMRFunc           func(projectID, iid int) error
	mergeMRFunc            func(projectID, iid int) error
	createMRFunc           func(projectID string, opts gitlab.CreateMROptions) (*gitlab.MergeRequest, error)
	updateMRLabelsFunc     func(projectID, iid int, labels []string) (*gitlab.MergeRequest, error)
	updateMRReviewersFunc  func(projectID, iid int, reviewerIDs []int) (*gitlab.MergeRequest, error)
	setAutoMergeFunc       func(projectID, iid int) error
	cancelAutoMergeFunc    func(projectID, iid int) error
	getMRDiscussionsFunc   func(projectID, iid int) ([]gitlab.Discussion, error)
	getMRApprovalsFunc     func(projectID, iid int) (*gitlab.ApprovalState, error)
	getMRPipelinesFunc     func(projectID, mrIID int) ([]gitlab.PipelineInfo, error)
	getEventsFunc          func(opts gitlab.ListEventsOptions) ([]gitlab.Event, error)
	listProjectsFunc       func(opts gitlab.ListProjectsOptions) ([]gitlab.Project, error)
	listUsersFunc          func(opts gitlab.ListUsersOptions) ([]gitlab.User, error)
	listProjectMembersFunc func(projectID string, search string) ([]gitlab.User, error)
	listProjectLabelsFunc  func(projectID string, search string) ([]gitlab.Label, error)
}

func (m *mockGitLabClient) ListMRs(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error) {
	if m.listMRsFunc != nil {
		return m.listMRsFunc(opts)
	}
	return nil, nil
}

func (m *mockGitLabClient) GetMR(projectID, iid int) (*gitlab.MergeRequest, error) {
	if m.getMRFunc != nil {
		return m.getMRFunc(projectID, iid)
	}
	return nil, nil
}

func (m *mockGitLabClient) GetMRByGlobalID(id int) (*gitlab.MergeRequest, error) {
	if m.getMRByGlobalIDFunc != nil {
		return m.getMRByGlobalIDFunc(id)
	}
	return nil, nil
}

func (m *mockGitLabClient) RebaseMR(projectID, iid int) error {
	if m.rebaseMRFunc != nil {
		return m.rebaseMRFunc(projectID, iid)
	}
	return nil
}

func (m *mockGitLabClient) MergeMR(projectID, iid int) error {
	if m.mergeMRFunc != nil {
		return m.mergeMRFunc(projectID, iid)
	}
	return nil
}

func (m *mockGitLabClient) CreateMR(projectID string, opts gitlab.CreateMROptions) (*gitlab.MergeRequest, error) {
	if m.createMRFunc != nil {
		return m.createMRFunc(projectID, opts)
	}
	return &gitlab.MergeRequest{}, nil
}

func (m *mockGitLabClient) UpdateMRLabels(projectID, iid int, labels []string) (*gitlab.MergeRequest, error) {
	if m.updateMRLabelsFunc != nil {
		return m.updateMRLabelsFunc(projectID, iid, labels)
	}
	return &gitlab.MergeRequest{Labels: labels}, nil
}

func (m *mockGitLabClient) UpdateMRReviewers(projectID, iid int, reviewerIDs []int) (*gitlab.MergeRequest, error) {
	if m.updateMRReviewersFunc != nil {
		return m.updateMRReviewersFunc(projectID, iid, reviewerIDs)
	}
	return &gitlab.MergeRequest{}, nil
}

func (m *mockGitLabClient) SetAutoMerge(projectID, iid int) error {
	if m.setAutoMergeFunc != nil {
		return m.setAutoMergeFunc(projectID, iid)
	}
	return nil
}

func (m *mockGitLabClient) CancelAutoMerge(projectID, iid int) error {
	if m.cancelAutoMergeFunc != nil {
		return m.cancelAutoMergeFunc(projectID, iid)
	}
	return nil
}

func (m *mockGitLabClient) GetMRDiscussions(projectID, iid int) ([]gitlab.Discussion, error) {
	if m.getMRDiscussionsFunc != nil {
		return m.getMRDiscussionsFunc(projectID, iid)
	}
	return nil, nil
}

func (m *mockGitLabClient) GetMRApprovals(projectID, iid int) (*gitlab.ApprovalState, error) {
	if m.getMRApprovalsFunc != nil {
		return m.getMRApprovalsFunc(projectID, iid)
	}
	return &gitlab.ApprovalState{}, nil
}

func (m *mockGitLabClient) GetMRPipelines(projectID, mrIID int) ([]gitlab.PipelineInfo, error) {
	if m.getMRPipelinesFunc != nil {
		return m.getMRPipelinesFunc(projectID, mrIID)
	}
	return nil, nil
}

func (m *mockGitLabClient) GetEvents(opts gitlab.ListEventsOptions) ([]gitlab.Event, error) {
	if m.getEventsFunc != nil {
		return m.getEventsFunc(opts)
	}
	return nil, nil
}

func (m *mockGitLabClient) ListProjects(opts gitlab.ListProjectsOptions) ([]gitlab.Project, error) {
	if m.listProjectsFunc != nil {
		return m.listProjectsFunc(opts)
	}
	return nil, nil
}

func (m *mockGitLabClient) ListUsers(opts gitlab.ListUsersOptions) ([]gitlab.User, error) {
	if m.listUsersFunc != nil {
		return m.listUsersFunc(opts)
	}
	return nil, nil
}

func (m *mockGitLabClient) ListProjectMembers(projectID string, search string) ([]gitlab.User, error) {
	if m.listProjectMembersFunc != nil {
		return m.listProjectMembersFunc(projectID, search)
	}
	return nil, nil
}

func (m *mockGitLabClient) ListProjectLabels(projectID string, search string) ([]gitlab.Label, error) {
	if m.listProjectLabelsFunc != nil {
		return m.listProjectLabelsFunc(projectID, search)
	}
	return nil, nil
}
