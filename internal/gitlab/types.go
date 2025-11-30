package gitlab

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

type MergeRequest struct {
	ID                   int    `json:"id"`
	IID                  int    `json:"iid"`
	ProjectID            int    `json:"project_id"`
	Title                string `json:"title"`
	Description          string `json:"description"`
	State                string `json:"state"`
	SourceBranch         string `json:"source_branch"`
	TargetBranch         string `json:"target_branch"`
	Author               User   `json:"author"`
	WebURL               string `json:"web_url"`
	DetailedMergeStatus  string `json:"detailed_merge_status"`
	HasConflicts         bool   `json:"has_conflicts"`
	RebaseInProgress     bool      `json:"rebase_in_progress"`
	MergeError           string    `json:"merge_error"`
	HeadPipeline         *Pipeline `json:"head_pipeline"`
	Labels               []string  `json:"labels"`
	Reviewers            []User    `json:"reviewers"`
}

type Pipeline struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
	Ref    string `json:"ref"`
	SHA    string `json:"sha"`
	WebURL string `json:"web_url"`
}

type PipelineJob struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Stage  string `json:"stage"`
}

type PipelineStats struct {
	Passed  int
	Running int
	Pending int
	Failed  int
}

type ListMROptions struct {
	State         string
	Scope         string
	ProjectID     int
	AuthorID      int
	PerPage       int
	ApprovedByIDs string
}

type Event struct {
	ID          int        `json:"id"`
	ActionName  string     `json:"action_name"`
	CreatedAt   string     `json:"created_at"`
	ProjectID   int        `json:"project_id"`
	TargetType  string     `json:"target_type"`
	TargetID    int        `json:"target_id"`
	TargetIID   int        `json:"target_iid"`
	TargetTitle string     `json:"target_title"`
	PushData    *PushData  `json:"push_data"`
	Note        *NoteData  `json:"note"`
}

type PushData struct {
	CommitCount int    `json:"commit_count"`
	Action      string `json:"action"`
	RefType     string `json:"ref_type"`
	Ref         string `json:"ref"`
	CommitTitle string `json:"commit_title"`
}

type NoteData struct {
	NoteableType string `json:"noteable_type"`
	Body         string `json:"body"`
}

type Commit struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Path              string `json:"path"`
	PathWithNamespace string `json:"path_with_namespace"`
	Visibility        string `json:"visibility"`
	WebURL            string `json:"web_url"`
	DefaultBranch     string `json:"default_branch"`
}

type Label struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

type Discussion struct {
	ID    string `json:"id"`
	Notes []Note `json:"notes"`
}

type Note struct {
	ID         int           `json:"id"`
	Author     User          `json:"author"`
	Body       string        `json:"body"`
	CreatedAt  string        `json:"created_at"`
	Resolved   bool          `json:"resolved"`
	Resolvable bool          `json:"resolvable"`
	System     bool          `json:"system"`
	Position   *DiffPosition `json:"position"`
}

type DiffPosition struct {
	NewPath string `json:"new_path"`
	NewLine int    `json:"new_line"`
	OldPath string `json:"old_path"`
	OldLine int    `json:"old_line"`
}

type ApprovalState struct {
	Approved  bool           `json:"approved"`
	Approvers []ApprovalUser `json:"approved_by"`
}

type ApprovalUser struct {
	User User `json:"user"`
}

type LabelEvent struct {
	ID        int    `json:"id"`
	Action    string `json:"action"`
	CreatedAt string `json:"created_at"`
	User      User   `json:"user"`
	Label     Label  `json:"label"`
}

type ActivityEntry struct {
	Date        string                 `json:"date"`
	Time        string                 `json:"time"`
	Type        string                 `json:"type"`
	Project     string                 `json:"project"`
	Source      string                 `json:"source,omitempty"`
	Target      string                 `json:"target,omitempty"`
	Description string                 `json:"description"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type ListEventsOptions struct {
	After  string
	Before string
}

type CreateMROptions struct {
	SourceBranch       string
	TargetBranch       string
	Title              string
	Description        string
	Draft              bool
	Squash             bool
	RemoveSourceBranch bool
	AllowCollaboration bool
}

type ListProjectsOptions struct {
	Search     string
	Owned      bool
	Membership bool
	PerPage    int
}

type ListUsersOptions struct {
	Search  string
	PerPage int
}
