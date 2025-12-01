package gitlab

import (
	"fmt"
	"net/url"
	"time"
)

func (c *Client) GetEvents(opts ListEventsOptions) ([]Event, error) {
	params := url.Values{}
	params.Set("per_page", "100")

	if opts.After != "" {
		params.Set("after", opts.After)
	}
	if opts.Before != "" {
		params.Set("before", opts.Before)
	}

	var allEvents []Event
	page := 1

	for {
		params.Set("page", fmt.Sprintf("%d", page))
		path := "/events?" + params.Encode()

		var events []Event
		if err := c.get(path, &events); err != nil {
			return nil, fmt.Errorf("fetching events: %w", err)
		}

		if len(events) == 0 {
			break
		}

		allEvents = append(allEvents, events...)
		page++

		if len(events) < 100 {
			break
		}
	}

	return allEvents, nil
}

func (c *Client) GetProject(projectID int) (*Project, error) {
	path := fmt.Sprintf("/projects/%d", projectID)

	var project Project
	if err := c.get(path, &project); err != nil {
		return nil, fmt.Errorf("fetching project %d: %w", projectID, err)
	}

	return &project, nil
}

func (c *Client) GetCommits(projectID int, refName string, limit int) ([]Commit, error) {
	params := url.Values{}
	params.Set("ref_name", refName)
	params.Set("per_page", fmt.Sprintf("%d", limit))

	path := fmt.Sprintf("/projects/%d/repository/commits?%s", projectID, params.Encode())

	var commits []Commit
	if err := c.get(path, &commits); err != nil {
		return nil, fmt.Errorf("fetching commits for project %d ref %s: %w", projectID, refName, err)
	}

	return commits, nil
}

// CommitDateRange holds the result of analyzing commit dates
type CommitDateRange struct {
	OldestDate string
	NewestDate string
	SpanDays   int
	TotalCount int
	FetchError string
}

// GetCommitDateRange fetches commits for a branch and calculates the date span
// Returns the span in days between oldest and newest commit
// Limits to maxCommits to avoid excessive API calls
func (c *Client) GetCommitDateRange(projectID int, refName string, maxCommits int) CommitDateRange {
	result := CommitDateRange{}

	commits, err := c.GetCommits(projectID, refName, maxCommits)
	if err != nil {
		result.FetchError = err.Error()
		return result
	}

	if len(commits) == 0 {
		return result
	}

	result.TotalCount = len(commits)

	// Find oldest and newest dates
	var oldest, newest time.Time
	for i, commit := range commits {
		if commit.AuthoredDate == "" {
			continue
		}

		// Parse ISO 8601 date (GitLab format: 2025-11-29T22:51:00.000+00:00)
		t, err := time.Parse(time.RFC3339, commit.AuthoredDate)
		if err != nil {
			// Try alternative format without timezone
			t, err = time.Parse("2006-01-02T15:04:05.000Z", commit.AuthoredDate)
			if err != nil {
				continue
			}
		}

		if i == 0 || t.Before(oldest) {
			oldest = t
			result.OldestDate = commit.AuthoredDate
		}
		if i == 0 || t.After(newest) {
			newest = t
			result.NewestDate = commit.AuthoredDate
		}
	}

	// Calculate span in days
	if !oldest.IsZero() && !newest.IsZero() {
		result.SpanDays = int(newest.Sub(oldest).Hours() / 24)
	}

	return result
}
