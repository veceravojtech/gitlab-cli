package gitlab

import (
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) ListProjects(opts ListProjectsOptions) ([]Project, error) {
	params := url.Values{}

	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	} else {
		params.Set("per_page", "20")
	}

	if opts.Search != "" {
		params.Set("search", opts.Search)
	}

	if opts.Owned {
		params.Set("owned", "true")
	}

	if opts.Membership {
		params.Set("membership", "true")
	} else {
		params.Set("membership", "true") // Default to membership
	}

	path := "/projects?" + params.Encode()

	var projects []Project
	if err := c.get(path, &projects); err != nil {
		return nil, fmt.Errorf("listing projects: %w", err)
	}

	return projects, nil
}

func (c *Client) GetProjectByIDOrPath(idOrPath string) (*Project, error) {
	// URL-encode the path for paths like "group/repo"
	encoded := url.PathEscape(idOrPath)
	path := fmt.Sprintf("/projects/%s", encoded)

	var project Project
	if err := c.get(path, &project); err != nil {
		return nil, fmt.Errorf("getting project %s: %w", idOrPath, err)
	}

	return &project, nil
}
