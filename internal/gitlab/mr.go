package gitlab

import (
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) ListMRs(opts ListMROptions) ([]MergeRequest, error) {
	params := url.Values{}

	if opts.State != "" {
		params.Set("state", opts.State)
	} else {
		params.Set("state", "opened")
	}

	if opts.Scope != "" {
		params.Set("scope", opts.Scope)
	} else {
		params.Set("scope", "all")
	}

	if opts.ProjectID > 0 {
		params.Set("project_id", strconv.Itoa(opts.ProjectID))
	}

	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	} else {
		params.Set("per_page", "20")
	}

	if opts.ApprovedByIDs != "" {
		params.Set("approved_by_ids", opts.ApprovedByIDs)
	}

	path := "/merge_requests?" + params.Encode()

	var mrs []MergeRequest
	if err := c.get(path, &mrs); err != nil {
		return nil, fmt.Errorf("listing MRs: %w", err)
	}

	return mrs, nil
}

func (c *Client) GetMR(projectID, iid int) (*MergeRequest, error) {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d?include_rebase_in_progress=true", projectID, iid)

	var mr MergeRequest
	if err := c.get(path, &mr); err != nil {
		return nil, fmt.Errorf("getting MR: %w", err)
	}

	return &mr, nil
}

func (c *Client) GetMRByGlobalID(id int) (*MergeRequest, error) {
	// GitLab's global MR endpoint may not be available on all instances
	// So we search through all MRs to find the one with matching ID
	path := fmt.Sprintf("/merge_requests?scope=all&state=opened&per_page=100")

	var mrs []MergeRequest
	if err := c.get(path, &mrs); err != nil {
		return nil, fmt.Errorf("getting MR: %w", err)
	}

	// Find the MR with matching ID
	for _, mr := range mrs {
		if mr.ID == id {
			// Get full details including rebase status
			return c.GetMR(mr.ProjectID, mr.IID)
		}
	}

	// If not found in recent MRs, try direct endpoint (might work on some instances)
	path = fmt.Sprintf("/merge_requests/%d", id)
	var mr MergeRequest
	if err := c.get(path, &mr); err != nil {
		return nil, fmt.Errorf("MR with ID %d not found", id)
	}

	return c.GetMR(mr.ProjectID, mr.IID)
}

func (c *Client) RebaseMR(projectID, iid int) error {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/rebase", projectID, iid)

	if err := c.put(path, nil); err != nil {
		return fmt.Errorf("triggering rebase: %w", err)
	}

	return nil
}

func (c *Client) MergeMR(projectID, iid int) error {
	path := fmt.Sprintf("/projects/%d/merge_requests/%d/merge", projectID, iid)

	if err := c.put(path, nil); err != nil {
		return fmt.Errorf("merging MR: %w", err)
	}

	return nil
}

func (c *Client) GetPipelineJobs(projectID, pipelineID int) ([]PipelineJob, error) {
	path := fmt.Sprintf("/projects/%d/pipelines/%d/jobs?per_page=100", projectID, pipelineID)

	var jobs []PipelineJob
	if err := c.get(path, &jobs); err != nil {
		return nil, fmt.Errorf("getting pipeline jobs: %w", err)
	}

	return jobs, nil
}

func (c *Client) GetPipelineStats(projectID, pipelineID int) (*PipelineStats, error) {
	jobs, err := c.GetPipelineJobs(projectID, pipelineID)
	if err != nil {
		return nil, err
	}

	stats := &PipelineStats{}
	for _, job := range jobs {
		switch job.Status {
		case "success":
			stats.Passed++
		case "running":
			stats.Running++
		case "pending", "created":
			stats.Pending++
		case "failed":
			stats.Failed++
		}
	}

	return stats, nil
}

func (c *Client) CreateMR(projectID string, opts CreateMROptions) (*MergeRequest, error) {
	encoded := url.PathEscape(projectID)
	path := fmt.Sprintf("/projects/%s/merge_requests", encoded)

	body := map[string]interface{}{
		"source_branch": opts.SourceBranch,
		"target_branch": opts.TargetBranch,
		"title":         opts.Title,
	}

	if opts.Description != "" {
		body["description"] = opts.Description
	}

	if opts.Draft {
		body["draft"] = true
	}

	if opts.Squash {
		body["squash"] = true
	}

	if opts.RemoveSourceBranch {
		body["remove_source_branch"] = true
	}

	if opts.AllowCollaboration {
		body["allow_collaboration"] = true
	}

	var mr MergeRequest
	if err := c.post(path, body, &mr); err != nil {
		return nil, fmt.Errorf("creating MR: %w", err)
	}

	return &mr, nil
}
