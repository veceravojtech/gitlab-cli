package gitlab

import (
	"fmt"
	"net/url"
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
