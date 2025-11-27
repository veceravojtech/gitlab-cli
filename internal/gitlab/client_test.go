package gitlab

import (
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient("https://gitlab.example.com", "test-token")

	if client.baseURL != "https://gitlab.example.com" {
		t.Errorf("expected baseURL https://gitlab.example.com, got %s", client.baseURL)
	}
}
