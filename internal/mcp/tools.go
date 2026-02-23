package mcp

import (
	"context"
	"errors"
	"fmt"
	"time"

	sdkmcp "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/user/gitlab-cli/internal/gitlab"
	"github.com/user/gitlab-cli/internal/mergeops"
)

// --- mr-list ---

type MRListInput struct {
	Mine      bool `json:"mine,omitempty"      jsonschema:"Only MRs assigned to me"`
	Approved  bool `json:"approved,omitempty"   jsonschema:"Only approved MRs"`
	ProjectID int  `json:"project_id,omitempty" jsonschema:"Filter by project ID"`
}

type MRListOutput struct {
	MergeRequests []MRSummary `json:"merge_requests"`
}

func (s *Server) MRListHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRListInput) (*sdkmcp.CallToolResult, MRListOutput, error) {
	opts := gitlab.ListMROptions{State: "opened"}

	if input.Mine {
		opts.Scope = "assigned_to_me"
	}
	if input.Approved {
		opts.ApprovedByIDs = "Any"
	}
	if input.ProjectID > 0 {
		opts.ProjectID = input.ProjectID
	}

	mrs, err := s.client.ListMRs(opts)
	if err != nil {
		return nil, MRListOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	summaries := make([]MRSummary, len(mrs))
	for i, mr := range mrs {
		summaries[i] = toMRSummary(mr)
	}

	return nil, MRListOutput{MergeRequests: summaries}, nil
}

// --- mr-show ---

type MRShowInput struct {
	ProjectID int  `json:"project_id" jsonschema:"Project ID,required"`
	MRIID     int  `json:"mr_iid"     jsonschema:"Merge request IID,required"`
	Detail    bool `json:"detail,omitempty" jsonschema:"Include discussions and approvals"`
}

type MRShowOutput struct {
	MRDetail
}

func (s *Server) MRShowHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRShowInput) (*sdkmcp.CallToolResult, MRShowOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRShowOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	mr, err := s.client.GetMR(input.ProjectID, input.MRIID)
	if err != nil {
		return nil, MRShowOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	output := MRShowOutput{
		MRDetail: MRDetail{
			MergeRequest: toMRSummary(*mr),
			Labels:       mr.Labels,
			Reviewers:    toUserSummaries(mr.Reviewers),
		},
	}

	if input.Detail {
		discussions, err := s.client.GetMRDiscussions(input.ProjectID, input.MRIID)
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("failed to fetch discussions: %v", err))
		} else {
			output.Discussions = toDiscussionOutputs(discussions)
		}

		approvals, err := s.client.GetMRApprovals(input.ProjectID, input.MRIID)
		if err != nil {
			output.Warnings = append(output.Warnings, fmt.Sprintf("failed to fetch approvals: %v", err))
		} else {
			output.Approvals = toApprovalOutput(approvals)
		}
	}

	return nil, output, nil
}

// --- mr-create ---

type MRCreateInput struct {
	Project            string   `json:"project"       jsonschema:"Project ID or path (e.g. group/repo),required"`
	SourceBranch       string   `json:"source_branch" jsonschema:"Source branch name,required"`
	TargetBranch       string   `json:"target_branch" jsonschema:"Target branch name,required"`
	Title              string   `json:"title"         jsonschema:"MR title,required"`
	Description        string   `json:"description,omitempty"    jsonschema:"MR description"`
	Draft              bool     `json:"draft,omitempty"          jsonschema:"Create as draft MR"`
	Squash             bool     `json:"squash,omitempty"         jsonschema:"Enable squash on merge"`
	RemoveSourceBranch bool     `json:"remove_source_branch,omitempty" jsonschema:"Delete source branch after merge"`
	Labels             []string `json:"labels,omitempty"         jsonschema:"Labels to apply after creation"`
	ReviewerIDs        []int    `json:"reviewer_ids,omitempty"   jsonschema:"User IDs to add as reviewers"`
}

type MRCreateOutput struct {
	MergeRequest MRSummary `json:"merge_request"`
}

func (s *Server) MRCreateHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRCreateInput) (*sdkmcp.CallToolResult, MRCreateOutput, error) {
	if input.Project == "" || input.SourceBranch == "" || input.TargetBranch == "" || input.Title == "" {
		return nil, MRCreateOutput{}, fmt.Errorf("%w: project, source_branch, target_branch, and title are required", ErrMissingParam)
	}

	opts := gitlab.CreateMROptions{
		SourceBranch:       input.SourceBranch,
		TargetBranch:       input.TargetBranch,
		Title:              input.Title,
		Description:        input.Description,
		Draft:              input.Draft,
		Squash:             input.Squash,
		RemoveSourceBranch: input.RemoveSourceBranch,
	}

	mr, err := s.client.CreateMR(input.Project, opts)
	if err != nil {
		return nil, MRCreateOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	// Apply labels if provided
	if len(input.Labels) > 0 {
		mr, err = s.client.UpdateMRLabels(mr.ProjectID, mr.IID, input.Labels)
		if err != nil {
			return nil, MRCreateOutput{}, fmt.Errorf("%w: failed to set labels: %v", ErrGitLabAPI, err)
		}
	}

	// Set reviewers if provided
	if len(input.ReviewerIDs) > 0 {
		mr, err = s.client.UpdateMRReviewers(mr.ProjectID, mr.IID, input.ReviewerIDs)
		if err != nil {
			return nil, MRCreateOutput{}, fmt.Errorf("%w: failed to set reviewers: %v", ErrGitLabAPI, err)
		}
	}

	return nil, MRCreateOutput{MergeRequest: toMRSummary(*mr)}, nil
}

// --- mr-merge ---

type MRMergeInput struct {
	ProjectID    int    `json:"project_id"             jsonschema:"Project ID,required"`
	MRIID        int    `json:"mr_iid"                 jsonschema:"Merge request IID,required"`
	AutoRebase   bool   `json:"auto_rebase,omitempty"   jsonschema:"Automatically rebase if needed"`
	MaxRetries   int    `json:"max_retries,omitempty"   jsonschema:"Max rebase attempts (default 3)"`
	Timeout      string `json:"timeout,omitempty"       jsonschema:"Overall timeout duration (default 5m)"`
}

type MRMergeOutput struct {
	Merged   bool `json:"merged"`
	Attempts int  `json:"attempts"`
}

func (s *Server) MRMergeHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRMergeInput) (*sdkmcp.CallToolResult, MRMergeOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRMergeOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	timeout := 5 * time.Minute
	if input.Timeout != "" {
		parsed, err := time.ParseDuration(input.Timeout)
		if err != nil {
			return nil, MRMergeOutput{}, fmt.Errorf("%w: invalid timeout %q: %v", ErrInvalidInput, input.Timeout, err)
		}
		timeout = parsed
	}

	maxRetries := 3
	if input.MaxRetries > 0 {
		maxRetries = input.MaxRetries
	}

	opts := mergeops.MergeOptions{
		ProjectID:    input.ProjectID,
		MRIID:        input.MRIID,
		AutoRebase:   input.AutoRebase,
		MaxRetries:   maxRetries,
		Timeout:      timeout,
		PollInterval: s.config.PollInterval,
	}

	result, err := mergeops.MergeWithRebase(ctx, s.client, opts, nil)
	if err != nil {
		return nil, MRMergeOutput{}, mapMergeopsError(err)
	}

	return nil, MRMergeOutput{Merged: result.Merged, Attempts: result.Attempts}, nil
}

// --- mr-rebase ---

type MRRebaseInput struct {
	ProjectID int `json:"project_id" jsonschema:"Project ID,required"`
	MRIID     int `json:"mr_iid"     jsonschema:"Merge request IID,required"`
}

type MRRebaseOutput struct {
	Rebased     bool   `json:"rebased"`
	MergeStatus string `json:"merge_status"`
}

func (s *Server) MRRebaseHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRRebaseInput) (*sdkmcp.CallToolResult, MRRebaseOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRRebaseOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	if err := s.client.RebaseMR(input.ProjectID, input.MRIID); err != nil {
		return nil, MRRebaseOutput{}, fmt.Errorf("%w: %v", ErrRebaseFailed, err)
	}

	// Add timeout bound to prevent infinite polling
	ctx, cancel := context.WithTimeout(ctx, s.config.Timeout)
	defer cancel()

	// Poll until rebase completes
	for {
		select {
		case <-ctx.Done():
			return nil, MRRebaseOutput{}, fmt.Errorf("%w: context cancelled", ErrMergeTimeout)
		default:
		}

		sleepCtx(ctx, s.config.PollInterval)

		mr, err := s.client.GetMR(input.ProjectID, input.MRIID)
		if err != nil {
			return nil, MRRebaseOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
		}

		if !mr.RebaseInProgress {
			if mr.MergeError != "" {
				return nil, MRRebaseOutput{}, fmt.Errorf("%w: %s", ErrRebaseFailed, mr.MergeError)
			}
			return nil, MRRebaseOutput{Rebased: true, MergeStatus: mr.DetailedMergeStatus}, nil
		}
	}
}

// sleepCtx sleeps for the given duration or returns early if context is cancelled.
func sleepCtx(ctx context.Context, d time.Duration) {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}

// --- mr-label ---

type MRLabelInput struct {
	ProjectID int      `json:"project_id" jsonschema:"Project ID,required"`
	MRIID     int      `json:"mr_iid"     jsonschema:"Merge request IID,required"`
	Add       []string `json:"add,omitempty"    jsonschema:"Labels to add"`
	Remove    []string `json:"remove,omitempty" jsonschema:"Labels to remove"`
}

type MRLabelOutput struct {
	Labels []string `json:"labels"`
}

func (s *Server) MRLabelHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRLabelInput) (*sdkmcp.CallToolResult, MRLabelOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRLabelOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	mr, err := s.client.GetMR(input.ProjectID, input.MRIID)
	if err != nil {
		return nil, MRLabelOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	// If no add/remove, just return current labels
	if len(input.Add) == 0 && len(input.Remove) == 0 {
		return nil, MRLabelOutput{Labels: mr.Labels}, nil
	}

	// Compute new label set
	labelSet := make(map[string]bool)
	for _, l := range mr.Labels {
		labelSet[l] = true
	}
	for _, l := range input.Add {
		labelSet[l] = true
	}
	for _, l := range input.Remove {
		delete(labelSet, l)
	}

	newLabels := make([]string, 0, len(labelSet))
	for l := range labelSet {
		newLabels = append(newLabels, l)
	}

	mr, err = s.client.UpdateMRLabels(input.ProjectID, input.MRIID, newLabels)
	if err != nil {
		return nil, MRLabelOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	return nil, MRLabelOutput{Labels: mr.Labels}, nil
}

// --- mr-reviewer ---

type MRReviewerInput struct {
	ProjectID int   `json:"project_id"       jsonschema:"Project ID,required"`
	MRIID     int   `json:"mr_iid"           jsonschema:"Merge request IID,required"`
	Add       []int `json:"add,omitempty"     jsonschema:"User IDs to add as reviewers"`
	Remove    []int `json:"remove,omitempty"  jsonschema:"User IDs to remove from reviewers"`
}

type MRReviewerOutput struct {
	Reviewers []UserSummary `json:"reviewers"`
}

func (s *Server) MRReviewerHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRReviewerInput) (*sdkmcp.CallToolResult, MRReviewerOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRReviewerOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	mr, err := s.client.GetMR(input.ProjectID, input.MRIID)
	if err != nil {
		return nil, MRReviewerOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	// If no add/remove, just return current reviewers
	if len(input.Add) == 0 && len(input.Remove) == 0 {
		return nil, MRReviewerOutput{Reviewers: toUserSummaries(mr.Reviewers)}, nil
	}

	// Compute new reviewer set
	reviewerSet := make(map[int]bool)
	for _, r := range mr.Reviewers {
		reviewerSet[r.ID] = true
	}
	for _, id := range input.Add {
		reviewerSet[id] = true
	}
	for _, id := range input.Remove {
		delete(reviewerSet, id)
	}

	newIDs := make([]int, 0, len(reviewerSet))
	for id := range reviewerSet {
		newIDs = append(newIDs, id)
	}

	mr, err = s.client.UpdateMRReviewers(input.ProjectID, input.MRIID, newIDs)
	if err != nil {
		return nil, MRReviewerOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	return nil, MRReviewerOutput{Reviewers: toUserSummaries(mr.Reviewers)}, nil
}

// --- mr-auto-merge ---

type MRAutoMergeInput struct {
	ProjectID int  `json:"project_id" jsonschema:"Project ID,required"`
	MRIID     int  `json:"mr_iid"     jsonschema:"Merge request IID,required"`
	Cancel    bool `json:"cancel,omitempty" jsonschema:"Cancel auto-merge instead of enabling it"`
}

type MRAutoMergeOutput struct {
	Enabled bool `json:"enabled"`
}

func (s *Server) MRAutoMergeHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input MRAutoMergeInput) (*sdkmcp.CallToolResult, MRAutoMergeOutput, error) {
	if input.ProjectID == 0 || input.MRIID == 0 {
		return nil, MRAutoMergeOutput{}, fmt.Errorf("%w: project_id and mr_iid are required", ErrMissingParam)
	}

	if input.Cancel {
		if err := s.client.CancelAutoMerge(input.ProjectID, input.MRIID); err != nil {
			return nil, MRAutoMergeOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
		}
		return nil, MRAutoMergeOutput{Enabled: false}, nil
	}

	if err := s.client.SetAutoMerge(input.ProjectID, input.MRIID); err != nil {
		return nil, MRAutoMergeOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}
	return nil, MRAutoMergeOutput{Enabled: true}, nil
}

// --- activity-list ---

type ActivityListInput struct {
	DaysAgo int    `json:"days_ago,omitempty" jsonschema:"Number of days back to fetch (default 30)"`
	After   string `json:"after,omitempty"    jsonschema:"Start date (YYYY-MM-DD)"`
	Before  string `json:"before,omitempty"   jsonschema:"End date (YYYY-MM-DD)"`
}

type ActivityListOutput struct {
	Events []EventOutput `json:"events"`
}

func (s *Server) ActivityListHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input ActivityListInput) (*sdkmcp.CallToolResult, ActivityListOutput, error) {
	opts := gitlab.ListEventsOptions{}

	if input.After != "" {
		opts.After = input.After
	} else if input.DaysAgo > 0 {
		opts.After = time.Now().AddDate(0, 0, -input.DaysAgo).Format("2006-01-02")
	} else {
		opts.After = time.Now().AddDate(0, 0, -30).Format("2006-01-02")
	}

	if input.Before != "" {
		opts.Before = input.Before
	}

	events, err := s.client.GetEvents(opts)
	if err != nil {
		return nil, ActivityListOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	outputs := make([]EventOutput, len(events))
	for i, e := range events {
		outputs[i] = EventOutput{
			ID:          e.ID,
			ActionName:  e.ActionName,
			CreatedAt:   e.CreatedAt,
			ProjectID:   e.ProjectID,
			TargetType:  e.TargetType,
			TargetID:    e.TargetID,
			TargetIID:   e.TargetIID,
			TargetTitle: e.TargetTitle,
		}
	}

	return nil, ActivityListOutput{Events: outputs}, nil
}

// --- project-list ---

type ProjectListInput struct {
	Search     string `json:"search,omitempty"     jsonschema:"Search projects by name"`
	Owned      bool   `json:"owned,omitempty"      jsonschema:"Only projects owned by me"`
	Membership bool   `json:"membership,omitempty" jsonschema:"Only projects I am a member of"`
}

type ProjectListOutput struct {
	Projects []ProjectOutput `json:"projects"`
}

func (s *Server) ProjectListHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input ProjectListInput) (*sdkmcp.CallToolResult, ProjectListOutput, error) {
	opts := gitlab.ListProjectsOptions{
		Search:     input.Search,
		Owned:      input.Owned,
		Membership: input.Membership,
	}

	projects, err := s.client.ListProjects(opts)
	if err != nil {
		return nil, ProjectListOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	outputs := make([]ProjectOutput, len(projects))
	for i, p := range projects {
		outputs[i] = ProjectOutput{
			ID:                p.ID,
			Name:              p.Name,
			PathWithNamespace: p.PathWithNamespace,
			WebURL:            p.WebURL,
			DefaultBranch:     p.DefaultBranch,
		}
	}

	return nil, ProjectListOutput{Projects: outputs}, nil
}

// --- user-list ---

type UserListInput struct {
	Search  string `json:"search,omitempty"  jsonschema:"Search users by name or username"`
	Project string `json:"project,omitempty" jsonschema:"Project ID or path to scope search to project members"`
}

type UserListOutput struct {
	Users []UserSummary `json:"users"`
}

func (s *Server) UserListHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input UserListInput) (*sdkmcp.CallToolResult, UserListOutput, error) {
	var users []gitlab.User
	var err error

	if input.Project != "" {
		users, err = s.client.ListProjectMembers(input.Project, input.Search)
	} else {
		users, err = s.client.ListUsers(gitlab.ListUsersOptions{Search: input.Search})
	}

	if err != nil {
		return nil, UserListOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	summaries := make([]UserSummary, len(users))
	for i, u := range users {
		summaries[i] = UserSummary{ID: u.ID, Username: u.Username, Name: u.Name}
	}

	return nil, UserListOutput{Users: summaries}, nil
}

// --- label-list ---

type LabelListInput struct {
	Project string `json:"project" jsonschema:"Project ID or path,required"`
	Search  string `json:"search,omitempty" jsonschema:"Filter labels by name"`
}

type LabelListOutput struct {
	Labels []LabelOutput `json:"labels"`
}

func (s *Server) LabelListHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input LabelListInput) (*sdkmcp.CallToolResult, LabelListOutput, error) {
	if input.Project == "" {
		return nil, LabelListOutput{}, fmt.Errorf("%w: project is required", ErrMissingParam)
	}

	labels, err := s.client.ListProjectLabels(input.Project, input.Search)
	if err != nil {
		return nil, LabelListOutput{}, fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	}

	outputs := make([]LabelOutput, len(labels))
	for i, l := range labels {
		outputs[i] = LabelOutput{
			ID:          l.ID,
			Name:        l.Name,
			Color:       l.Color,
			Description: l.Description,
		}
	}

	return nil, LabelListOutput{Labels: outputs}, nil
}

// --- config-show ---

type ConfigShowInput struct{}

type ConfigShowOutput struct {
	GitLabURL    string `json:"gitlab_url"`
	TokenSet     bool   `json:"token_set"`
	TokenMasked  string `json:"token_masked"`
	MaxRetries   int    `json:"max_retries"`
	Timeout      string `json:"timeout"`
	PollInterval string `json:"poll_interval"`
}

func (s *Server) ConfigShowHandler(ctx context.Context, req *sdkmcp.CallToolRequest, input ConfigShowInput) (*sdkmcp.CallToolResult, ConfigShowOutput, error) {
	masked := "****"
	if len(s.config.GitLabToken) >= 4 {
		masked = s.config.GitLabToken[:4] + "****"
	}

	return nil, ConfigShowOutput{
		GitLabURL:    s.config.GitLabURL,
		TokenSet:     s.config.GitLabToken != "",
		TokenMasked:  masked,
		MaxRetries:   s.config.MaxRetries,
		Timeout:      s.config.Timeout.String(),
		PollInterval: s.config.PollInterval.String(),
	}, nil
}

// --- Helper functions ---

func toMRSummary(mr gitlab.MergeRequest) MRSummary {
	return MRSummary{
		ID:           mr.ID,
		IID:          mr.IID,
		ProjectID:    mr.ProjectID,
		Title:        mr.Title,
		State:        mr.State,
		Author:       mr.Author.Username,
		SourceBranch: mr.SourceBranch,
		TargetBranch: mr.TargetBranch,
		WebURL:       mr.WebURL,
		MergeStatus:  mr.DetailedMergeStatus,
	}
}

func toUserSummaries(users []gitlab.User) []UserSummary {
	summaries := make([]UserSummary, len(users))
	for i, u := range users {
		summaries[i] = UserSummary{ID: u.ID, Username: u.Username, Name: u.Name}
	}
	return summaries
}

func toDiscussionOutputs(discussions []gitlab.Discussion) []DiscussionOutput {
	outputs := make([]DiscussionOutput, len(discussions))
	for i, d := range discussions {
		notes := make([]NoteOutput, len(d.Notes))
		for j, n := range d.Notes {
			notes[j] = NoteOutput{
				ID:         n.ID,
				Author:     n.Author.Username,
				Body:       n.Body,
				CreatedAt:  n.CreatedAt,
				Resolved:   n.Resolved,
				Resolvable: n.Resolvable,
				System:     n.System,
			}
		}
		outputs[i] = DiscussionOutput{ID: d.ID, Notes: notes}
	}
	return outputs
}

func toApprovalOutput(state *gitlab.ApprovalState) *ApprovalOutput {
	if state == nil {
		return nil
	}
	approvers := make([]string, len(state.Approvers))
	for i, a := range state.Approvers {
		approvers[i] = a.User.Username
	}
	return &ApprovalOutput{
		Approved:  state.Approved,
		Approvers: approvers,
	}
}

// mapMergeopsError translates mergeops sentinel errors to mcp sentinel errors
// so callers can use errors.Is with mcp error types consistently.
func mapMergeopsError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, mergeops.ErrMergeConflict):
		return fmt.Errorf("%w: %v", ErrMergeConflict, err)
	case errors.Is(err, mergeops.ErrMergeTimeout):
		return fmt.Errorf("%w: %v", ErrMergeTimeout, err)
	case errors.Is(err, mergeops.ErrRebaseFailed):
		return fmt.Errorf("%w: %v", ErrRebaseFailed, err)
	case errors.Is(err, mergeops.ErrPipelineFailed):
		return fmt.Errorf("%w: %v", ErrPipelineFailed, err)
	case errors.Is(err, mergeops.ErrGitLabAPI):
		return fmt.Errorf("%w: %v", ErrGitLabAPI, err)
	default:
		return fmt.Errorf("merge operation failed: %w", err)
	}
}

