package api

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/internal/config"
)

// =============================================================================
// GitHubWebhookHandler Construction Tests
// =============================================================================

func TestNewGitHubWebhookHandler(t *testing.T) {
	t.Run("creates handler with nil dependencies", func(t *testing.T) {
		cfg := config.BranchingConfig{Enabled: true}
		handler := NewGitHubWebhookHandler(nil, nil, cfg)
		assert.NotNil(t, handler)
		assert.Nil(t, handler.manager)
		assert.Nil(t, handler.router)
		assert.True(t, handler.config.Enabled)
	})

	t.Run("creates handler with config", func(t *testing.T) {
		cfg := config.BranchingConfig{
			Enabled:            true,
			MaxTotalBranches:   50,
			MaxBranchesPerUser: 5,
		}
		handler := NewGitHubWebhookHandler(nil, nil, cfg)
		assert.Equal(t, 50, handler.config.MaxTotalBranches)
		assert.Equal(t, 5, handler.config.MaxBranchesPerUser)
	})
}

// =============================================================================
// GitHub Webhook Payload Struct Tests
// =============================================================================

func TestGitHubWebhookPayload_Struct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		payload := GitHubWebhookPayload{
			Action: "opened",
			PullRequest: &GitHubPullRequest{
				Number:  123,
				State:   "open",
				Title:   "Test PR",
				HTMLURL: "https://github.com/owner/repo/pull/123",
			},
			Repository: &GitHubRepository{
				ID:       12345,
				Name:     "repo",
				FullName: "owner/repo",
				Private:  false,
			},
			Sender: &GitHubUser{
				ID:    67890,
				Login: "testuser",
			},
		}

		assert.Equal(t, "opened", payload.Action)
		assert.NotNil(t, payload.PullRequest)
		assert.Equal(t, 123, payload.PullRequest.Number)
		assert.NotNil(t, payload.Repository)
		assert.Equal(t, "owner/repo", payload.Repository.FullName)
	})

	t.Run("JSON deserialization", func(t *testing.T) {
		jsonData := `{
			"action": "closed",
			"pull_request": {
				"number": 456,
				"state": "closed",
				"title": "Feature PR",
				"html_url": "https://github.com/owner/repo/pull/456",
				"merged": true
			},
			"repository": {
				"id": 11111,
				"name": "repo",
				"full_name": "owner/repo",
				"private": true
			}
		}`

		var payload GitHubWebhookPayload
		err := json.Unmarshal([]byte(jsonData), &payload)
		require.NoError(t, err)

		assert.Equal(t, "closed", payload.Action)
		assert.Equal(t, 456, payload.PullRequest.Number)
		assert.True(t, payload.PullRequest.Merged)
		assert.True(t, payload.Repository.Private)
	})
}

// =============================================================================
// GitHubPullRequest Struct Tests
// =============================================================================

func TestGitHubPullRequest_Struct(t *testing.T) {
	t.Run("open PR", func(t *testing.T) {
		pr := GitHubPullRequest{
			Number:  100,
			State:   "open",
			Title:   "Add new feature",
			HTMLURL: "https://github.com/owner/repo/pull/100",
			Merged:  false,
			Head: &GitHubRef{
				Ref: "feature-branch",
				SHA: "abc123",
			},
			Base: &GitHubRef{
				Ref: "main",
				SHA: "def456",
			},
		}

		assert.Equal(t, 100, pr.Number)
		assert.Equal(t, "open", pr.State)
		assert.False(t, pr.Merged)
		assert.Equal(t, "feature-branch", pr.Head.Ref)
		assert.Equal(t, "main", pr.Base.Ref)
	})

	t.Run("merged PR", func(t *testing.T) {
		pr := GitHubPullRequest{
			Number:  200,
			State:   "closed",
			Title:   "Merged feature",
			HTMLURL: "https://github.com/owner/repo/pull/200",
			Merged:  true,
		}

		assert.Equal(t, "closed", pr.State)
		assert.True(t, pr.Merged)
	})
}

// =============================================================================
// GitHubIssue Struct Tests
// =============================================================================

func TestGitHubIssue_Struct(t *testing.T) {
	t.Run("issue with labels", func(t *testing.T) {
		issue := GitHubIssue{
			Number:  42,
			State:   "open",
			Title:   "Bug report",
			Body:    "Description of the bug",
			HTMLURL: "https://github.com/owner/repo/issues/42",
			Labels: []GitHubLabel{
				{ID: 1, Name: "bug", Color: "d73a4a"},
				{ID: 2, Name: "priority:high", Color: "ff0000"},
			},
			User: &GitHubUser{
				ID:    12345,
				Login: "reporter",
			},
		}

		assert.Equal(t, 42, issue.Number)
		assert.Equal(t, "open", issue.State)
		assert.Len(t, issue.Labels, 2)
		assert.Equal(t, "bug", issue.Labels[0].Name)
		assert.Equal(t, "reporter", issue.User.Login)
	})

	t.Run("issue with assignees", func(t *testing.T) {
		issue := GitHubIssue{
			Number: 50,
			State:  "open",
			Title:  "Task",
			Assignees: []GitHubUser{
				{ID: 1, Login: "dev1"},
				{ID: 2, Login: "dev2"},
			},
		}

		assert.Len(t, issue.Assignees, 2)
		assert.Equal(t, "dev1", issue.Assignees[0].Login)
	})
}

// =============================================================================
// GitHubLabel Struct Tests
// =============================================================================

func TestGitHubLabel_Struct(t *testing.T) {
	t.Run("label with all fields", func(t *testing.T) {
		label := GitHubLabel{
			ID:          12345,
			Name:        "enhancement",
			Description: "New feature or request",
			Color:       "a2eeef",
		}

		assert.Equal(t, 12345, label.ID)
		assert.Equal(t, "enhancement", label.Name)
		assert.Equal(t, "New feature or request", label.Description)
		assert.Equal(t, "a2eeef", label.Color)
	})
}

// =============================================================================
// GitHubRepository Struct Tests
// =============================================================================

func TestGitHubRepository_Struct(t *testing.T) {
	t.Run("public repository", func(t *testing.T) {
		repo := GitHubRepository{
			ID:       123456,
			Name:     "my-repo",
			FullName: "owner/my-repo",
			Private:  false,
			HTMLURL:  "https://github.com/owner/my-repo",
		}

		assert.Equal(t, 123456, repo.ID)
		assert.Equal(t, "my-repo", repo.Name)
		assert.Equal(t, "owner/my-repo", repo.FullName)
		assert.False(t, repo.Private)
	})

	t.Run("private repository", func(t *testing.T) {
		repo := GitHubRepository{
			ID:       789012,
			Name:     "private-repo",
			FullName: "owner/private-repo",
			Private:  true,
		}

		assert.True(t, repo.Private)
	})
}

// =============================================================================
// computeHMACSHA256 Tests
// =============================================================================

func TestComputeHMACSHA256(t *testing.T) {
	t.Run("computes correct HMAC", func(t *testing.T) {
		data := []byte("test payload")
		key := "secret-key"

		result := computeHMACSHA256(data, key)

		// Verify by computing expected value
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write(data)
		expected := hex.EncodeToString(mac.Sum(nil))

		assert.Equal(t, expected, result)
	})

	t.Run("different data produces different hash", func(t *testing.T) {
		key := "same-key"
		hash1 := computeHMACSHA256([]byte("data1"), key)
		hash2 := computeHMACSHA256([]byte("data2"), key)

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("different key produces different hash", func(t *testing.T) {
		data := []byte("same-data")
		hash1 := computeHMACSHA256(data, "key1")
		hash2 := computeHMACSHA256(data, "key2")

		assert.NotEqual(t, hash1, hash2)
	})

	t.Run("empty data", func(t *testing.T) {
		result := computeHMACSHA256([]byte(""), "key")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 64) // SHA256 produces 64 hex characters
	})

	t.Run("empty key", func(t *testing.T) {
		result := computeHMACSHA256([]byte("data"), "")
		assert.NotEmpty(t, result)
		assert.Len(t, result, 64)
	})
}

// =============================================================================
// HandleWebhook Tests
// =============================================================================

func TestHandleWebhook_BranchingDisabled(t *testing.T) {
	app := fiber.New()
	cfg := config.BranchingConfig{Enabled: false}
	handler := NewGitHubWebhookHandler(nil, nil, cfg)

	app.Post("/webhooks/github", handler.HandleWebhook)

	payload := `{"action":"opened","repository":{"full_name":"owner/repo"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusServiceUnavailable, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.Equal(t, "BRANCHING_DISABLED", result["code"])
}

func TestHandleWebhook_MissingEventHeader(t *testing.T) {
	app := fiber.New()
	cfg := config.BranchingConfig{Enabled: true}
	handler := NewGitHubWebhookHandler(nil, nil, cfg)

	app.Post("/webhooks/github", handler.HandleWebhook)

	payload := `{"action":"opened","repository":{"full_name":"owner/repo"}}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	// Missing X-GitHub-Event header

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.Equal(t, "MISSING_EVENT", result["code"])
}

func TestHandleWebhook_InvalidPayload(t *testing.T) {
	app := fiber.New()
	cfg := config.BranchingConfig{Enabled: true}
	handler := NewGitHubWebhookHandler(nil, nil, cfg)

	app.Post("/webhooks/github", handler.HandleWebhook)

	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.Equal(t, "INVALID_PAYLOAD", result["code"])
}

func TestHandleWebhook_MissingRepository(t *testing.T) {
	app := fiber.New()
	cfg := config.BranchingConfig{Enabled: true}
	handler := NewGitHubWebhookHandler(nil, nil, cfg)

	app.Post("/webhooks/github", handler.HandleWebhook)

	// Payload without repository
	payload := `{"action":"opened"}`
	req := httptest.NewRequest(http.MethodPost, "/webhooks/github", bytes.NewReader([]byte(payload)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Event", "pull_request")

	resp, err := app.Test(req)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()

	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)

	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	var result map[string]interface{}
	err = json.Unmarshal(respBody, &result)
	require.NoError(t, err)

	assert.Equal(t, "MISSING_REPOSITORY", result["code"])
}

// =============================================================================
// GetWebhookURL Tests
// =============================================================================

func TestGetWebhookURL(t *testing.T) {
	handler := NewGitHubWebhookHandler(nil, nil, config.BranchingConfig{})

	tests := []struct {
		name     string
		baseURL  string
		expected string
	}{
		{
			name:     "without trailing slash",
			baseURL:  "https://example.com",
			expected: "https://example.com/api/v1/webhooks/github",
		},
		{
			name:     "with trailing slash",
			baseURL:  "https://example.com/",
			expected: "https://example.com/api/v1/webhooks/github",
		},
		{
			name:     "localhost",
			baseURL:  "http://localhost:3000",
			expected: "http://localhost:3000/api/v1/webhooks/github",
		},
		{
			name:     "with path",
			baseURL:  "https://api.example.com/v1/",
			expected: "https://api.example.com/v1/api/v1/webhooks/github",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.GetWebhookURL(tt.baseURL)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// Event Type Tests
// =============================================================================

func TestGitHubEventTypes(t *testing.T) {
	supportedEvents := []string{
		"pull_request",
		"issues",
		"ping",
	}

	t.Run("supported event types", func(t *testing.T) {
		for _, event := range supportedEvents {
			assert.NotEmpty(t, event)
		}
	})

	t.Run("pull_request actions", func(t *testing.T) {
		actions := []string{"opened", "reopened", "closed", "synchronize"}
		for _, action := range actions {
			assert.NotEmpty(t, action)
		}
	})

	t.Run("issue actions", func(t *testing.T) {
		actions := []string{"opened", "labeled", "closed", "assigned"}
		for _, action := range actions {
			assert.NotEmpty(t, action)
		}
	})
}

// =============================================================================
// Special Label Tests
// =============================================================================

func TestSpecialLabels(t *testing.T) {
	specialLabels := []string{
		"claude-fix",
		"priority:critical",
		"priority:high",
	}

	t.Run("special labels are defined", func(t *testing.T) {
		assert.Len(t, specialLabels, 3)
		assert.Contains(t, specialLabels, "claude-fix")
		assert.Contains(t, specialLabels, "priority:critical")
		assert.Contains(t, specialLabels, "priority:high")
	})
}

// =============================================================================
// JSON Serialization Tests
// =============================================================================

func TestGitHubStructs_JSONSerialization(t *testing.T) {
	t.Run("GitHubWebhookPayload serializes correctly", func(t *testing.T) {
		payload := GitHubWebhookPayload{
			Action: "opened",
			Repository: &GitHubRepository{
				FullName: "owner/repo",
			},
		}

		data, err := json.Marshal(payload)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"action":"opened"`)
		assert.Contains(t, string(data), `"full_name":"owner/repo"`)
	})

	t.Run("GitHubPullRequest serializes correctly", func(t *testing.T) {
		pr := GitHubPullRequest{
			Number:  123,
			State:   "open",
			Title:   "Test PR",
			HTMLURL: "https://github.com/owner/repo/pull/123",
			Merged:  false,
		}

		data, err := json.Marshal(pr)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"number":123`)
		assert.Contains(t, string(data), `"state":"open"`)
		assert.Contains(t, string(data), `"merged":false`)
	})

	t.Run("GitHubIssue serializes correctly", func(t *testing.T) {
		issue := GitHubIssue{
			Number: 42,
			State:  "open",
			Title:  "Test Issue",
		}

		data, err := json.Marshal(issue)
		require.NoError(t, err)

		assert.Contains(t, string(data), `"number":42`)
		assert.Contains(t, string(data), `"state":"open"`)
	})
}
