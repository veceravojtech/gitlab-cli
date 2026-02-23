package mcp

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/user/gitlab-cli/internal/config"
	"github.com/user/gitlab-cli/internal/gitlab"
)

func testServer(mock *mockGitLabClient) *Server {
	return &Server{
		client: mock,
		config: &config.Config{
			GitLabURL:    "https://test.example.com",
			GitLabToken:  "glpat-testtoken123",
			MaxRetries:   3,
			Timeout:      5 * time.Minute,
			PollInterval: 1 * time.Millisecond,
		},
	}
}

func TestMRListHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       MRListInput
		setupClient func() *mockGitLabClient
		wantCount   int
		wantErr     bool
		errContains string
	}{
		{
			name:  "list all open MRs",
			input: MRListInput{},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					listMRsFunc: func(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error) {
						return []gitlab.MergeRequest{
							{ID: 1, IID: 10, Title: "Test MR", State: "opened", Author: gitlab.User{Username: "dev"}},
							{ID: 2, IID: 11, Title: "Another MR", State: "opened", Author: gitlab.User{Username: "dev2"}},
						}, nil
					},
				}
			},
			wantCount: 2,
		},
		{
			name:  "filter mine",
			input: MRListInput{Mine: true},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					listMRsFunc: func(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error) {
						if opts.Scope != "assigned_to_me" {
							t.Errorf("expected scope assigned_to_me, got %s", opts.Scope)
						}
						return []gitlab.MergeRequest{{ID: 1, IID: 10, Title: "My MR"}}, nil
					},
				}
			},
			wantCount: 1,
		},
		{
			name:  "API error",
			input: MRListInput{},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					listMRsFunc: func(opts gitlab.ListMROptions) ([]gitlab.MergeRequest, error) {
						return nil, errors.New("network error")
					},
				}
			},
			wantErr:     true,
			errContains: "network error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setupClient())
			_, output, err := s.MRListHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if len(output.MergeRequests) != tt.wantCount {
				t.Errorf("got %d MRs, want %d", len(output.MergeRequests), tt.wantCount)
			}
		})
	}
}

func TestMRShowHandler(t *testing.T) {
	tests := []struct {
		name         string
		input        MRShowInput
		setupClient  func() *mockGitLabClient
		wantErr      bool
		errContains  string
		wantTitle    string
		wantWarnings int
	}{
		{
			name:  "show basic MR",
			input: MRShowInput{ProjectID: 1, MRIID: 10},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(projectID, iid int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							ID: 100, IID: 10, ProjectID: 1, Title: "Test MR",
							Labels: []string{"bug"}, Author: gitlab.User{Username: "dev"},
						}, nil
					},
				}
			},
			wantTitle: "Test MR",
		},
		{
			name:  "show with detail",
			input: MRShowInput{ProjectID: 1, MRIID: 10, Detail: true},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(projectID, iid int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							ID: 100, IID: 10, ProjectID: 1, Title: "Detail MR",
							Author: gitlab.User{Username: "dev"},
						}, nil
					},
					getMRDiscussionsFunc: func(projectID, iid int) ([]gitlab.Discussion, error) {
						return []gitlab.Discussion{{ID: "d1", Notes: []gitlab.Note{{ID: 1, Body: "comment"}}}}, nil
					},
					getMRApprovalsFunc: func(projectID, iid int) (*gitlab.ApprovalState, error) {
						return &gitlab.ApprovalState{Approved: true}, nil
					},
				}
			},
			wantTitle: "Detail MR",
		},
		{
			name:  "detail with partial failures populates warnings",
			input: MRShowInput{ProjectID: 1, MRIID: 10, Detail: true},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(projectID, iid int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							ID: 100, IID: 10, ProjectID: 1, Title: "Warning MR",
							Author: gitlab.User{Username: "dev"},
						}, nil
					},
					getMRDiscussionsFunc: func(projectID, iid int) ([]gitlab.Discussion, error) {
						return nil, errors.New("discussions forbidden")
					},
					getMRApprovalsFunc: func(projectID, iid int) (*gitlab.ApprovalState, error) {
						return nil, errors.New("approvals forbidden")
					},
				}
			},
			wantTitle:    "Warning MR",
			wantWarnings: 2,
		},
		{
			name:  "missing params",
			input: MRShowInput{},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantErr:     true,
			errContains: "required",
		},
		{
			name:  "API error",
			input: MRShowInput{ProjectID: 1, MRIID: 10},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(projectID, iid int) (*gitlab.MergeRequest, error) {
						return nil, errors.New("not found")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setupClient())
			_, output, err := s.MRShowHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if output.MergeRequest.Title != tt.wantTitle {
				t.Errorf("title = %q, want %q", output.MergeRequest.Title, tt.wantTitle)
			}
			if tt.wantWarnings > 0 && len(output.Warnings) != tt.wantWarnings {
				t.Errorf("warnings = %d, want %d", len(output.Warnings), tt.wantWarnings)
			}
		})
	}
}

func TestMRCreateHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       MRCreateInput
		setupClient func() *mockGitLabClient
		wantErr     bool
		errContains string
	}{
		{
			name: "create basic MR",
			input: MRCreateInput{
				Project:      "group/repo",
				SourceBranch: "feature",
				TargetBranch: "main",
				Title:        "New feature",
			},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					createMRFunc: func(projectID string, opts gitlab.CreateMROptions) (*gitlab.MergeRequest, error) {
						if opts.Title != "New feature" {
							t.Errorf("title = %q, want 'New feature'", opts.Title)
						}
						return &gitlab.MergeRequest{ID: 1, IID: 10, ProjectID: 42, Title: opts.Title}, nil
					},
				}
			},
		},
		{
			name: "create with labels and reviewers",
			input: MRCreateInput{
				Project:      "group/repo",
				SourceBranch: "feature",
				TargetBranch: "main",
				Title:        "With extras",
				Labels:       []string{"bug"},
				ReviewerIDs:  []int{42},
			},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					createMRFunc: func(projectID string, opts gitlab.CreateMROptions) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{ID: 1, IID: 10, ProjectID: 42, Title: opts.Title}, nil
					},
					updateMRLabelsFunc: func(projectID, iid int, labels []string) (*gitlab.MergeRequest, error) {
						if len(labels) != 1 || labels[0] != "bug" {
							t.Errorf("labels = %v, want [bug]", labels)
						}
						return &gitlab.MergeRequest{ID: 1, IID: 10, ProjectID: 42, Labels: labels}, nil
					},
					updateMRReviewersFunc: func(projectID, iid int, reviewerIDs []int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{ID: 1, IID: 10, ProjectID: 42}, nil
					},
				}
			},
		},
		{
			name:  "missing required params",
			input: MRCreateInput{Project: "group/repo"},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantErr:     true,
			errContains: "required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setupClient())
			_, _, err := s.MRCreateHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
			}
		})
	}
}

func TestMRLabelHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       MRLabelInput
		setupClient func() *mockGitLabClient
		wantLabels  int
		wantErr     bool
	}{
		{
			name:  "list current labels",
			input: MRLabelInput{ProjectID: 1, MRIID: 10},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{Labels: []string{"bug", "feature"}}, nil
					},
				}
			},
			wantLabels: 2,
		},
		{
			name:  "add and remove labels",
			input: MRLabelInput{ProjectID: 1, MRIID: 10, Add: []string{"urgent"}, Remove: []string{"bug"}},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{Labels: []string{"bug", "feature"}}, nil
					},
					updateMRLabelsFunc: func(_, _ int, labels []string) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{Labels: labels}, nil
					},
				}
			},
			wantLabels: 2, // feature + urgent
		},
		{
			name:  "missing params",
			input: MRLabelInput{},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setupClient())
			_, output, err := s.MRLabelHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}

			if len(output.Labels) != tt.wantLabels {
				t.Errorf("got %d labels, want %d", len(output.Labels), tt.wantLabels)
			}
		})
	}
}

func TestMRReviewerHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   MRReviewerInput
		setup   func() *mockGitLabClient
		wantErr bool
	}{
		{
			name:  "list reviewers",
			input: MRReviewerInput{ProjectID: 1, MRIID: 10},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							Reviewers: []gitlab.User{{ID: 1, Username: "dev"}},
						}, nil
					},
				}
			},
		},
		{
			name:  "add reviewer",
			input: MRReviewerInput{ProjectID: 1, MRIID: 10, Add: []int{42}},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							Reviewers: []gitlab.User{{ID: 1, Username: "dev"}},
						}, nil
					},
					updateMRReviewersFunc: func(_, _ int, ids []int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							Reviewers: []gitlab.User{{ID: 1}, {ID: 42}},
						}, nil
					},
				}
			},
		},
		{
			name:    "missing params",
			input:   MRReviewerInput{},
			setup:   func() *mockGitLabClient { return &mockGitLabClient{} },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, _, err := s.MRReviewerHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMRAutoMergeHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       MRAutoMergeInput
		setup       func() *mockGitLabClient
		wantEnabled bool
		wantErr     bool
	}{
		{
			name:  "enable auto-merge",
			input: MRAutoMergeInput{ProjectID: 1, MRIID: 10},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantEnabled: true,
		},
		{
			name:  "cancel auto-merge",
			input: MRAutoMergeInput{ProjectID: 1, MRIID: 10, Cancel: true},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantEnabled: false,
		},
		{
			name:    "missing params",
			input:   MRAutoMergeInput{},
			setup:   func() *mockGitLabClient { return &mockGitLabClient{} },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, output, err := s.MRAutoMergeHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if output.Enabled != tt.wantEnabled {
				t.Errorf("enabled = %v, want %v", output.Enabled, tt.wantEnabled)
			}
		})
	}
}

func TestActivityListHandler(t *testing.T) {
	tests := []struct {
		name      string
		input     ActivityListInput
		setup     func() *mockGitLabClient
		wantCount int
		wantErr   bool
	}{
		{
			name:  "default 30 days",
			input: ActivityListInput{},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					getEventsFunc: func(opts gitlab.ListEventsOptions) ([]gitlab.Event, error) {
						if opts.After == "" {
							t.Error("expected After to be set")
						}
						return []gitlab.Event{{ID: 1, ActionName: "pushed"}}, nil
					},
				}
			},
			wantCount: 1,
		},
		{
			name:  "custom days ago",
			input: ActivityListInput{DaysAgo: 7},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					getEventsFunc: func(opts gitlab.ListEventsOptions) ([]gitlab.Event, error) {
						return []gitlab.Event{{ID: 1}, {ID: 2}}, nil
					},
				}
			},
			wantCount: 2,
		},
		{
			name:  "API error",
			input: ActivityListInput{},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					getEventsFunc: func(opts gitlab.ListEventsOptions) ([]gitlab.Event, error) {
						return nil, errors.New("timeout")
					},
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, output, err := s.ActivityListHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if len(output.Events) != tt.wantCount {
				t.Errorf("got %d events, want %d", len(output.Events), tt.wantCount)
			}
		})
	}
}

func TestProjectListHandler(t *testing.T) {
	s := testServer(&mockGitLabClient{
		listProjectsFunc: func(opts gitlab.ListProjectsOptions) ([]gitlab.Project, error) {
			return []gitlab.Project{
				{ID: 1, Name: "test", PathWithNamespace: "group/test"},
			}, nil
		},
	})

	_, output, err := s.ProjectListHandler(context.Background(), nil, ProjectListInput{Search: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(output.Projects) != 1 {
		t.Errorf("got %d projects, want 1", len(output.Projects))
	}
}

func TestUserListHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   UserListInput
		setup   func() *mockGitLabClient
		wantErr bool
	}{
		{
			name:  "global search",
			input: UserListInput{Search: "dev"},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					listUsersFunc: func(opts gitlab.ListUsersOptions) ([]gitlab.User, error) {
						return []gitlab.User{{ID: 1, Username: "dev"}}, nil
					},
				}
			},
		},
		{
			name:  "project scoped",
			input: UserListInput{Project: "42", Search: "dev"},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					listProjectMembersFunc: func(projectID string, search string) ([]gitlab.User, error) {
						if projectID != "42" {
							t.Errorf("projectID = %s, want 42", projectID)
						}
						return []gitlab.User{{ID: 1, Username: "dev"}}, nil
					},
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, _, err := s.UserListHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLabelListHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   LabelListInput
		setup   func() *mockGitLabClient
		wantErr bool
	}{
		{
			name:  "list labels",
			input: LabelListInput{Project: "42"},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					listProjectLabelsFunc: func(projectID string, search string) ([]gitlab.Label, error) {
						return []gitlab.Label{{ID: 1, Name: "bug", Color: "#ff0000"}}, nil
					},
				}
			},
		},
		{
			name:    "missing project",
			input:   LabelListInput{},
			setup:   func() *mockGitLabClient { return &mockGitLabClient{} },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, _, err := s.LabelListHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigShowHandler(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		wantMasked  string
		wantTokenSet bool
	}{
		{
			name:         "normal token",
			token:        "glpat-abc123xyz",
			wantMasked:   "glpa****",
			wantTokenSet: true,
		},
		{
			name:         "short token",
			token:        "ab",
			wantMasked:   "****",
			wantTokenSet: true,
		},
		{
			name:         "empty token",
			token:        "",
			wantMasked:   "****",
			wantTokenSet: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				client: &mockGitLabClient{},
				config: &config.Config{
					GitLabURL:    "https://test.example.com",
					GitLabToken:  tt.token,
					MaxRetries:   3,
					Timeout:      5 * time.Minute,
					PollInterval: 5 * time.Second,
				},
			}

			_, output, err := s.ConfigShowHandler(context.Background(), nil, ConfigShowInput{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if output.TokenMasked != tt.wantMasked {
				t.Errorf("masked = %q, want %q", output.TokenMasked, tt.wantMasked)
			}
			if output.TokenSet != tt.wantTokenSet {
				t.Errorf("token_set = %v, want %v", output.TokenSet, tt.wantTokenSet)
			}
			if output.GitLabURL != "https://test.example.com" {
				t.Errorf("url = %q, want https://test.example.com", output.GitLabURL)
			}
		})
	}
}

func TestMRMergeHandler(t *testing.T) {
	tests := []struct {
		name        string
		input       MRMergeInput
		setupClient func() *mockGitLabClient
		wantMerged  bool
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful merge (already mergeable)",
			input: MRMergeInput{ProjectID: 1, MRIID: 10},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
					},
					mergeMRFunc: func(_, _ int) error { return nil },
				}
			},
			wantMerged: true,
		},
		{
			name:  "missing params",
			input: MRMergeInput{},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantErr:     true,
			errContains: "required",
		},
		{
			name:  "invalid timeout",
			input: MRMergeInput{ProjectID: 1, MRIID: 10, Timeout: "not-a-duration"},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{}
			},
			wantErr:     true,
			errContains: "invalid timeout",
		},
		{
			name:  "merge conflict",
			input: MRMergeInput{ProjectID: 1, MRIID: 10},
			setupClient: func() *mockGitLabClient {
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "conflict"}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "merge conflict",
		},
		{
			name:  "with auto-rebase on need_rebase then merge",
			input: MRMergeInput{ProjectID: 1, MRIID: 10, AutoRebase: true, Timeout: "500ms"},
			setupClient: func() *mockGitLabClient {
				callCount := 0
				return &mockGitLabClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						callCount++
						switch {
						case callCount == 1:
							return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
						case callCount == 2:
							// Rebase in progress
							return &gitlab.MergeRequest{RebaseInProgress: true, DetailedMergeStatus: "checking"}, nil
						default:
							// Rebase complete, now mergeable
							return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
						}
					},
					rebaseMRFunc: func(_, _ int) error { return nil },
					mergeMRFunc:  func(_, _ int) error { return nil },
				}
			},
			wantMerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setupClient())
			_, output, err := s.MRMergeHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				return
			}

			if output.Merged != tt.wantMerged {
				t.Errorf("merged = %v, want %v", output.Merged, tt.wantMerged)
			}
		})
	}
}

func TestMRRebaseHandler(t *testing.T) {
	tests := []struct {
		name    string
		input   MRRebaseInput
		setup   func() *mockGitLabClient
		wantErr bool
	}{
		{
			name:  "successful rebase",
			input: MRRebaseInput{ProjectID: 1, MRIID: 10},
			setup: func() *mockGitLabClient {
				callCount := 0
				return &mockGitLabClient{
					rebaseMRFunc: func(_, _ int) error { return nil },
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						callCount++
						if callCount == 1 {
							return &gitlab.MergeRequest{RebaseInProgress: true}, nil
						}
						return &gitlab.MergeRequest{
							RebaseInProgress:    false,
							DetailedMergeStatus: "mergeable",
						}, nil
					},
				}
			},
		},
		{
			name:    "missing params",
			input:   MRRebaseInput{},
			setup:   func() *mockGitLabClient { return &mockGitLabClient{} },
			wantErr: true,
		},
		{
			name:  "rebase API error",
			input: MRRebaseInput{ProjectID: 1, MRIID: 10},
			setup: func() *mockGitLabClient {
				return &mockGitLabClient{
					rebaseMRFunc: func(_, _ int) error { return errors.New("forbidden") },
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := testServer(tt.setup())
			_, _, err := s.MRRebaseHandler(context.Background(), nil, tt.input)

			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
