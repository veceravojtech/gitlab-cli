package mergeops

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/user/gitlab-cli/internal/gitlab"
)

// Merge operation errors
var (
	ErrMergeConflict  = errors.New("merge conflict")
	ErrMergeTimeout   = errors.New("merge timeout exceeded")
	ErrRebaseFailed   = errors.New("rebase failed")
	ErrPipelineFailed = errors.New("pipeline failed")
	ErrGitLabAPI      = errors.New("gitlab API error")
)

// MergeClient is the subset of gitlab.Client methods needed for merge operations.
type MergeClient interface {
	GetMR(projectID, iid int) (*gitlab.MergeRequest, error)
	RebaseMR(projectID, iid int) error
	MergeMR(projectID, iid int) error
}

// MergeOptions configures the merge-with-rebase loop.
type MergeOptions struct {
	ProjectID    int
	MRIID        int
	AutoRebase   bool
	MaxRetries   int
	Timeout      time.Duration
	PollInterval time.Duration
}

// MergeResult holds the outcome of a merge operation.
type MergeResult struct {
	Merged   bool
	Attempts int
}

// StatusCallback is called when the merge loop status changes.
// status is the high-level state (e.g., "merging", "rebasing").
// detail is a human-readable message.
type StatusCallback func(status, detail string)

// MergeWithRebase attempts to merge an MR, automatically rebasing if needed.
// The callback is nil-safe and called for status updates.
func MergeWithRebase(ctx context.Context, client MergeClient, opts MergeOptions, callback StatusCallback) (*MergeResult, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 5 * time.Minute
	}
	if opts.PollInterval <= 0 {
		opts.PollInterval = 5 * time.Second
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}

	ctx, cancel := context.WithTimeout(ctx, opts.Timeout)
	defer cancel()

	notify := func(status, detail string) {
		if callback != nil {
			callback(status, detail)
		}
	}

	attempt := 0
	var lastStatus string

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("%w: %s", ErrMergeTimeout, opts.Timeout)
		default:
		}

		mr, err := client.GetMR(opts.ProjectID, opts.MRIID)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
		}

		if mr.DetailedMergeStatus != lastStatus {
			notify("status", mr.DetailedMergeStatus)
			lastStatus = mr.DetailedMergeStatus
		}

		switch mr.DetailedMergeStatus {
		case "mergeable", "can_be_merged":
			notify("merging", "Merging...")
			if err := client.MergeMR(opts.ProjectID, opts.MRIID); err != nil {
				if opts.AutoRebase && strings.Contains(err.Error(), "rebase") {
					notify("rebase_needed", "Merge failed, needs rebase")
					lastStatus = ""
					continue
				}
				return nil, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
			}
			return &MergeResult{Merged: true, Attempts: attempt}, nil

		case "need_rebase", "cannot_be_merged_recheck":
			if !opts.AutoRebase {
				return nil, fmt.Errorf("%w: MR needs rebase (auto-rebase disabled)", ErrRebaseFailed)
			}

			attempt++
			if attempt > opts.MaxRetries {
				return nil, fmt.Errorf("%w: max retries exceeded (%d)", ErrRebaseFailed, opts.MaxRetries)
			}

			notify("rebasing", fmt.Sprintf("Triggering rebase... (attempt %d/%d)", attempt, opts.MaxRetries))
			if err := client.RebaseMR(opts.ProjectID, opts.MRIID); err != nil {
				return nil, fmt.Errorf("%w: %v", ErrRebaseFailed, err)
			}

			// Poll until rebase completes
			if err := waitForRebase(ctx, client, opts, notify); err != nil {
				return nil, err
			}

			lastStatus = ""

		case "conflict", "cannot_be_merged":
			return nil, fmt.Errorf("%w: conflicts detected", ErrMergeConflict)

		case "checking", "unchecked", "ci_still_running":
			if mr.HeadPipeline != nil {
				switch mr.HeadPipeline.Status {
				case "success":
					notify("merging", "CI complete, merging...")
					if err := client.MergeMR(opts.ProjectID, opts.MRIID); err != nil {
						if opts.AutoRebase && strings.Contains(err.Error(), "rebase") {
							notify("rebase_needed", "Merge failed, needs rebase")
							lastStatus = ""
							continue
						}
						return nil, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
					}
					return &MergeResult{Merged: true, Attempts: attempt}, nil
				case "failed":
					return nil, fmt.Errorf("%w: pipeline failed", ErrPipelineFailed)
				case "canceled":
					return nil, fmt.Errorf("%w: pipeline was canceled", ErrPipelineFailed)
				}
			}
			notify("waiting", "Waiting for CI")
			sleep(ctx, opts.PollInterval)

		default:
			return nil, fmt.Errorf("%w: unexpected merge status: %s", ErrGitLabAPI, mr.DetailedMergeStatus)
		}
	}
}

func waitForRebase(ctx context.Context, client MergeClient, opts MergeOptions, notify func(string, string)) error {
	notify("waiting", "Waiting for rebase")
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("%w: %s", ErrMergeTimeout, opts.Timeout)
		default:
		}

		sleep(ctx, opts.PollInterval)

		mr, err := client.GetMR(opts.ProjectID, opts.MRIID)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrGitLabAPI, err)
		}

		if !mr.RebaseInProgress {
			if mr.MergeError != "" {
				return fmt.Errorf("%w: %s", ErrRebaseFailed, mr.MergeError)
			}
			notify("rebase_complete", "Rebase complete")
			return nil
		}
	}
}

func sleep(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
