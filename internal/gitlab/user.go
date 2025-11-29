package gitlab

import (
	"fmt"
	"net/url"
	"strconv"
)

func (c *Client) ListUsers(opts ListUsersOptions) ([]User, error) {
	params := url.Values{}

	if opts.PerPage > 0 {
		params.Set("per_page", strconv.Itoa(opts.PerPage))
	} else {
		params.Set("per_page", "20")
	}

	if opts.Search != "" {
		params.Set("search", opts.Search)
	}

	path := "/users?" + params.Encode()

	var users []User
	if err := c.get(path, &users); err != nil {
		return nil, fmt.Errorf("listing users: %w", err)
	}

	return users, nil
}

func (c *Client) GetUserByUsername(username string) (*User, error) {
	params := url.Values{}
	params.Set("username", username)

	path := "/users?" + params.Encode()

	var users []User
	if err := c.get(path, &users); err != nil {
		return nil, fmt.Errorf("finding user %s: %w", username, err)
	}

	if len(users) == 0 {
		return nil, fmt.Errorf("user '%s' not found", username)
	}

	return &users[0], nil
}

func (c *Client) ListProjectMembers(projectID string, search string) ([]User, error) {
	encoded := url.PathEscape(projectID)
	params := url.Values{}
	params.Set("per_page", "100")

	if search != "" {
		params.Set("query", search)
	}

	path := fmt.Sprintf("/projects/%s/members/all?%s", encoded, params.Encode())

	var members []User
	if err := c.get(path, &members); err != nil {
		return nil, fmt.Errorf("listing project members: %w", err)
	}

	return members, nil
}

// ResolveUserID takes a username or numeric ID and returns the user ID
func (c *Client) ResolveUserID(userRef string) (int, error) {
	// Try to parse as integer first
	if id, err := strconv.Atoi(userRef); err == nil {
		return id, nil
	}

	// Otherwise, look up by username
	user, err := c.GetUserByUsername(userRef)
	if err != nil {
		return 0, err
	}

	return user.ID, nil
}
