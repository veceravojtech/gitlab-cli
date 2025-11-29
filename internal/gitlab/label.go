package gitlab

import (
	"fmt"
	"net/url"
)

func (c *Client) ListProjectLabels(projectID string, search string) ([]Label, error) {
	encoded := url.PathEscape(projectID)
	params := url.Values{}
	params.Set("per_page", "100")

	if search != "" {
		params.Set("search", search)
	}

	path := fmt.Sprintf("/projects/%s/labels?%s", encoded, params.Encode())

	var labels []Label
	if err := c.get(path, &labels); err != nil {
		return nil, fmt.Errorf("listing labels: %w", err)
	}

	return labels, nil
}
