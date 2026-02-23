package mergeops

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/user/gitlab-cli/internal/gitlab"
)

type mockMergeClient struct {
	getMRFunc    func(projectID, iid int) (*gitlab.MergeRequest, error)
	rebaseMRFunc func(projectID, iid int) error
	mergeMRFunc  func(projectID, iid int) error

	getMRCalls    int
	rebaseMRCalls int
	mergeMRCalls  int
}

func (m *mockMergeClient) GetMR(projectID, iid int) (*gitlab.MergeRequest, error) {
	m.getMRCalls++
	if m.getMRFunc != nil {
		return m.getMRFunc(projectID, iid)
	}
	return nil, errors.New("getMR not configured")
}

func (m *mockMergeClient) RebaseMR(projectID, iid int) error {
	m.rebaseMRCalls++
	if m.rebaseMRFunc != nil {
		return m.rebaseMRFunc(projectID, iid)
	}
	return nil
}

func (m *mockMergeClient) MergeMR(projectID, iid int) error {
	m.mergeMRCalls++
	if m.mergeMRFunc != nil {
		return m.mergeMRFunc(projectID, iid)
	}
	return nil
}

func defaultOpts() MergeOptions {
	return MergeOptions{
		ProjectID:    1,
		MRIID:        10,
		AutoRebase:   true,
		MaxRetries:   3,
		Timeout:      500 * time.Millisecond,
		PollInterval: 1 * time.Millisecond,
	}
}

func TestMergeWithRebase(t *testing.T) {
	tests := []struct {
		name        string
		opts        MergeOptions
		setupClient func() *mockMergeClient
		wantMerged  bool
		wantErr     bool
		errContains string
		errTarget   error
	}{
		{
			name: "happy path - mergeable immediately",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
					},
				}
			},
			wantMerged: true,
		},
		{
			name: "happy path - can_be_merged status",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "can_be_merged"}, nil
					},
				}
			},
			wantMerged: true,
		},
		{
			name: "rebase then merge",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				callCount := 0
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						callCount++
						switch {
						case callCount == 1:
							return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
						case callCount == 2:
							// Rebase in progress
							return &gitlab.MergeRequest{
								DetailedMergeStatus: "checking",
								RebaseInProgress:    true,
							}, nil
						case callCount == 3:
							// Rebase done
							return &gitlab.MergeRequest{
								DetailedMergeStatus: "checking",
								RebaseInProgress:    false,
							}, nil
						default:
							return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
						}
					},
				}
			},
			wantMerged: true,
		},
		{
			name: "conflict detected",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "conflict"}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "conflicts detected",
			errTarget:   ErrMergeConflict,
		},
		{
			name: "cannot_be_merged status",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "cannot_be_merged"}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "conflicts detected",
			errTarget:   ErrMergeConflict,
		},
		{
			name: "pipeline failed",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							DetailedMergeStatus: "ci_still_running",
							HeadPipeline:        &gitlab.Pipeline{Status: "failed"},
						}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "pipeline failed",
			errTarget:   ErrPipelineFailed,
		},
		{
			name: "pipeline canceled",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							DetailedMergeStatus: "checking",
							HeadPipeline:        &gitlab.Pipeline{Status: "canceled"},
						}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "pipeline was canceled",
			errTarget:   ErrPipelineFailed,
		},
		{
			name: "ci wait then merge on success",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				callCount := 0
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						callCount++
						if callCount == 1 {
							return &gitlab.MergeRequest{
								DetailedMergeStatus: "ci_still_running",
								HeadPipeline:        &gitlab.Pipeline{Status: "running"},
							}, nil
						}
						return &gitlab.MergeRequest{
							DetailedMergeStatus: "ci_still_running",
							HeadPipeline:        &gitlab.Pipeline{Status: "success"},
						}, nil
					},
				}
			},
			wantMerged: true,
		},
		{
			name: "context cancellation",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{
							DetailedMergeStatus: "ci_still_running",
							HeadPipeline:        &gitlab.Pipeline{Status: "running"},
						}, nil
					},
				}
			},
			wantErr: true,
			// Will timeout since CI never completes
			errTarget: ErrMergeTimeout,
		},
		{
			name: "no auto-rebase when need rebase",
			opts: func() MergeOptions {
				o := defaultOpts()
				o.AutoRebase = false
				return o
			}(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "auto-rebase disabled",
			errTarget:   ErrRebaseFailed,
		},
		{
			name: "max retries exceeded",
			opts: func() MergeOptions {
				o := defaultOpts()
				o.MaxRetries = 1
				return o
			}(),
			setupClient: func() *mockMergeClient {
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "max retries exceeded",
			errTarget:   ErrRebaseFailed,
		},
		{
			name: "rebase fails with merge error",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				callCount := 0
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						callCount++
						if callCount == 1 {
							return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
						}
						return &gitlab.MergeRequest{
							DetailedMergeStatus: "checking",
							RebaseInProgress:    false,
							MergeError:          "rebase failed: conflict in file.go",
						}, nil
					},
				}
			},
			wantErr:     true,
			errContains: "rebase failed: conflict in file.go",
			errTarget:   ErrRebaseFailed,
		},
		{
			name: "merge fails with rebase hint and auto-rebase",
			opts: defaultOpts(),
			setupClient: func() *mockMergeClient {
				mergeCallCount := 0
				getMRCallCount := 0
				return &mockMergeClient{
					getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
						getMRCallCount++
						if getMRCallCount <= 2 {
							return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
						}
						// After rebase, become mergeable again
						if getMRCallCount == 3 {
							return &gitlab.MergeRequest{DetailedMergeStatus: "need_rebase"}, nil
						}
						if getMRCallCount == 4 {
							// Rebase done
							return &gitlab.MergeRequest{RebaseInProgress: false}, nil
						}
						return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
					},
					mergeMRFunc: func(_, _ int) error {
						mergeCallCount++
						if mergeCallCount == 1 {
							return errors.New("rebase the source branch")
						}
						return nil
					},
				}
			},
			wantMerged: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			ctx := context.Background()

			result, err := MergeWithRebase(ctx, client, tt.opts, nil)

			if (err != nil) != tt.wantErr {
				t.Fatalf("MergeWithRebase() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errContains)
				}
				if tt.errTarget != nil && !errors.Is(err, tt.errTarget) {
					t.Errorf("errors.Is(%v, %v) = false, want true", err, tt.errTarget)
				}
				return
			}

			if result == nil {
				t.Fatal("result is nil")
			}
			if result.Merged != tt.wantMerged {
				t.Errorf("Merged = %v, want %v", result.Merged, tt.wantMerged)
			}
		})
	}
}

func TestStatusCallback(t *testing.T) {
	var statuses []string
	callback := func(status, detail string) {
		statuses = append(statuses, status+":"+detail)
	}

	client := &mockMergeClient{
		getMRFunc: func(_, _ int) (*gitlab.MergeRequest, error) {
			return &gitlab.MergeRequest{DetailedMergeStatus: "mergeable"}, nil
		},
	}

	opts := defaultOpts()
	_, err := MergeWithRebase(context.Background(), client, opts, callback)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(statuses) == 0 {
		t.Error("callback was never called")
	}
}
