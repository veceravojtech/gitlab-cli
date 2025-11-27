package gitlab

type User struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
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
