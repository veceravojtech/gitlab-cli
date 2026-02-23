package mcp

// --- Shared output types ---

type MRSummary struct {
	ID           int    `json:"id"`
	IID          int    `json:"iid"`
	ProjectID    int    `json:"project_id"`
	Title        string `json:"title"`
	State        string `json:"state"`
	Author       string `json:"author"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	WebURL       string `json:"web_url"`
	MergeStatus  string `json:"merge_status"`
}

type MRDetail struct {
	MergeRequest MRSummary          `json:"merge_request"`
	Labels       []string           `json:"labels"`
	Reviewers    []UserSummary      `json:"reviewers"`
	Discussions  []DiscussionOutput `json:"discussions,omitempty"`
	Approvals    *ApprovalOutput    `json:"approvals,omitempty"`
	Warnings     []string           `json:"warnings,omitempty"`
}

type UserSummary struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type DiscussionOutput struct {
	ID    string       `json:"id"`
	Notes []NoteOutput `json:"notes"`
}

type NoteOutput struct {
	ID         int    `json:"id"`
	Author     string `json:"author"`
	Body       string `json:"body"`
	CreatedAt  string `json:"created_at"`
	Resolved   bool   `json:"resolved"`
	Resolvable bool   `json:"resolvable"`
	System     bool   `json:"system"`
}

type ApprovalOutput struct {
	Approved  bool     `json:"approved"`
	Approvers []string `json:"approvers"`
}

type EventOutput struct {
	ID          int    `json:"id"`
	ActionName  string `json:"action_name"`
	CreatedAt   string `json:"created_at"`
	ProjectID   int    `json:"project_id"`
	TargetType  string `json:"target_type,omitempty"`
	TargetID    int    `json:"target_id,omitempty"`
	TargetIID   int    `json:"target_iid,omitempty"`
	TargetTitle string `json:"target_title,omitempty"`
}

type ProjectOutput struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	PathWithNamespace string `json:"path_with_namespace"`
	WebURL            string `json:"web_url"`
	DefaultBranch     string `json:"default_branch"`
}

type LabelOutput struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description,omitempty"`
}
