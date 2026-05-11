package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/branching"
	"github.com/nimbleflux/fluxbase/internal/config"
)

// GitHubWebhookHandler handles GitHub webhook events for database branching
type GitHubWebhookHandler struct {
	manager *branching.Manager
	router  *branching.Router
	config  config.BranchingConfig
}

// NewGitHubWebhookHandler creates a new GitHub webhook handler
func NewGitHubWebhookHandler(manager *branching.Manager, router *branching.Router, cfg config.BranchingConfig) *GitHubWebhookHandler {
	return &GitHubWebhookHandler{
		manager: manager,
		router:  router,
		config:  cfg,
	}
}

// GitHubWebhookPayload represents the common fields in GitHub webhook payloads
type GitHubWebhookPayload struct {
	Action       string              `json:"action"`
	PullRequest  *GitHubPullRequest  `json:"pull_request,omitempty"`
	Issue        *GitHubIssue        `json:"issue,omitempty"`
	Label        *GitHubLabel        `json:"label,omitempty"`
	Repository   *GitHubRepository   `json:"repository,omitempty"`
	Sender       *GitHubUser         `json:"sender,omitempty"`
	Installation *GitHubInstallation `json:"installation,omitempty"`
}

// GitHubIssue represents a GitHub issue
type GitHubIssue struct {
	Number    int           `json:"number"`
	State     string        `json:"state"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	HTMLURL   string        `json:"html_url"`
	Labels    []GitHubLabel `json:"labels"`
	Assignees []GitHubUser  `json:"assignees"`
	User      *GitHubUser   `json:"user,omitempty"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// GitHubLabel represents a GitHub label
type GitHubLabel struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Color       string `json:"color"`
}

// GitHubPullRequest represents a GitHub pull request
type GitHubPullRequest struct {
	Number  int        `json:"number"`
	State   string     `json:"state"`
	Title   string     `json:"title"`
	HTMLURL string     `json:"html_url"`
	Merged  bool       `json:"merged"`
	Base    *GitHubRef `json:"base,omitempty"`
	Head    *GitHubRef `json:"head,omitempty"`
}

// GitHubRef represents a Git reference (branch)
type GitHubRef struct {
	Ref  string            `json:"ref"`
	SHA  string            `json:"sha"`
	Repo *GitHubRepository `json:"repo,omitempty"`
}

// GitHubRepository represents a GitHub repository
type GitHubRepository struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	FullName string `json:"full_name"`
	Private  bool   `json:"private"`
	HTMLURL  string `json:"html_url"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	ID    int    `json:"id"`
	Login string `json:"login"`
}

// GitHubInstallation represents a GitHub App installation
type GitHubInstallation struct {
	ID int `json:"id"`
}

// HandleWebhook handles incoming GitHub webhook requests
func (h *GitHubWebhookHandler) HandleWebhook(c fiber.Ctx) error {
	if !h.config.Enabled {
		return SendErrorWithCode(c, fiber.StatusServiceUnavailable, "Database branching is not enabled", "BRANCHING_DISABLED")
	}

	// Get event type from header
	eventType := c.Get("X-GitHub-Event")
	if eventType == "" {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Missing X-GitHub-Event header", "MISSING_EVENT")
	}

	// Get delivery ID for logging
	deliveryID := c.Get("X-GitHub-Delivery")

	log.Info().
		Str("event", eventType).
		Str("delivery_id", deliveryID).
		Msg("Received GitHub webhook")

	// Parse the payload to get repository info for signature verification
	var payload GitHubWebhookPayload
	if err := json.Unmarshal(c.Body(), &payload); err != nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Failed to parse webhook payload: "+err.Error(), "INVALID_PAYLOAD")
	}

	// Get repository full name
	if payload.Repository == nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Missing repository in webhook payload", "MISSING_REPOSITORY")
	}

	repoFullName := payload.Repository.FullName

	// Verify webhook signature if configured
	if err := h.verifySignature(c, repoFullName); err != nil {
		log.Warn().
			Err(err).
			Str("repository", repoFullName).
			Str("delivery_id", deliveryID).
			Msg("Webhook signature verification failed")
		return SendErrorWithCode(c, fiber.StatusUnauthorized, "Webhook signature verification failed", "INVALID_SIGNATURE")
	}

	// Handle different event types
	switch eventType {
	case "pull_request":
		return h.handlePullRequestEvent(c, &payload)
	case "issues":
		return h.handleIssueEvent(c, &payload)
	case "ping":
		return h.handlePingEvent(c, &payload)
	default:
		// Ignore other events
		log.Debug().
			Str("event", eventType).
			Str("repository", repoFullName).
			Msg("Ignoring unhandled GitHub event")
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ignored",
			"message": "Event type not handled",
		})
	}
}

// verifySignature verifies the webhook signature using the configured secret
func (h *GitHubWebhookHandler) verifySignature(c fiber.Ctx, repository string) error {
	signature := c.Get("X-Hub-Signature-256")

	// Get the webhook configuration for this repository
	ghConfig, err := h.manager.GetStorage().GetGitHubConfig(c.RequestCtx(), repository)
	if err != nil && !errors.Is(err, branching.ErrGitHubConfigNotFound) {
		return fmt.Errorf("failed to get GitHub config: %w", err)
	}

	// Check if webhook secret is configured
	hasSecret := ghConfig != nil && ghConfig.WebhookSecret != nil && *ghConfig.WebhookSecret != ""

	if signature == "" {
		// No signature provided
		if hasSecret {
			// Secret is configured but no signature provided - reject
			return fmt.Errorf("webhook signature required but not provided")
		}

		// No config or no secret configured - log warning and reject
		// This prevents unauthenticated webhook abuse
		if ghConfig == nil {
			log.Warn().
				Str("repository", repository).
				Msg("GitHub webhook received for unconfigured repository - rejecting. Configure repository in /admin/branches/github/configs to enable webhooks.")
			return fmt.Errorf("repository not configured for webhooks: %s", repository)
		}

		// Config exists but no secret - log warning and allow (explicit opt-in to insecure mode)
		log.Warn().
			Str("repository", repository).
			Msg("GitHub webhook accepted without signature verification - configure webhook_secret for security")
		return nil
	}

	// Signature was provided - verify it if we have a secret
	if !hasSecret {
		// Signature provided but no secret configured - accept (GitHub is sending signatures)
		// Log info to encourage configuring the secret
		log.Info().
			Str("repository", repository).
			Msg("GitHub webhook signature ignored - no webhook_secret configured. Configure secret to enable verification.")
		return nil
	}

	// Verify the signature
	expected := computeHMACSHA256(c.Body(), *ghConfig.WebhookSecret)
	expectedSignature := "sha256=" + expected

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// computeHMACSHA256 computes HMAC-SHA256 of data with the given key
func computeHMACSHA256(data []byte, key string) string {
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write(data)
	return hex.EncodeToString(mac.Sum(nil))
}

// handlePullRequestEvent handles pull request events
func (h *GitHubWebhookHandler) handlePullRequestEvent(c fiber.Ctx, payload *GitHubWebhookPayload) error {
	if payload.PullRequest == nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Missing pull_request in payload", "MISSING_PULL_REQUEST")
	}

	pr := payload.PullRequest
	repo := payload.Repository.FullName

	log.Info().
		Str("action", payload.Action).
		Int("pr_number", pr.Number).
		Str("repository", repo).
		Msg("Processing pull request event")

	// Get GitHub config for this repository
	ghConfig, err := h.manager.GetStorage().GetGitHubConfig(c.RequestCtx(), repo)
	if err != nil && !errors.Is(err, branching.ErrGitHubConfigNotFound) {
		log.Error().Err(err).Str("repository", repo).Msg("Failed to get GitHub config")
		return SendErrorWithCode(c, fiber.StatusInternalServerError, "Failed to get GitHub configuration", "CONFIG_ERROR")
	}

	// Use default settings if no config
	autoCreate := true
	autoDelete := true
	if ghConfig != nil {
		autoCreate = ghConfig.AutoCreateOnPR
		autoDelete = ghConfig.AutoDeleteOnMerge
	}

	switch payload.Action {
	case "opened", "reopened":
		if !autoCreate {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "skipped",
				"message": "Auto-create is disabled for this repository",
			})
		}
		return h.createBranchForPR(c, repo, pr)

	case "closed":
		if !autoDelete {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "skipped",
				"message": "Auto-delete is disabled for this repository",
			})
		}
		return h.deleteBranchForPR(c, repo, pr)

	case "synchronize":
		// PR was updated (new commits pushed)
		// Could trigger migrations here in the future
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "acknowledged",
			"message": "PR synchronize event acknowledged",
		})

	default:
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ignored",
			"message": "Pull request action not handled: " + payload.Action,
		})
	}
}

// createBranchForPR creates a database branch for a pull request
func (h *GitHubWebhookHandler) createBranchForPR(c fiber.Ctx, repo string, pr *GitHubPullRequest) error {
	branch, err := h.manager.CreateBranchFromGitHubPR(c.RequestCtx(), repo, pr.Number, pr.HTMLURL)
	if err != nil {
		log.Error().Err(err).
			Str("repository", repo).
			Int("pr_number", pr.Number).
			Msg("Failed to create branch for PR")

		if errors.Is(err, branching.ErrBranchExists) {
			// Branch already exists, return success
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "exists",
				"message": "Branch already exists for this PR",
			})
		}

		if errors.Is(err, branching.ErrMaxBranchesReached) {
			return SendErrorWithCode(c, fiber.StatusForbidden, "Maximum number of branches has been reached", "MAX_BRANCHES_REACHED")
		}

		return SendErrorWithCode(c, fiber.StatusInternalServerError, "Failed to create branch: "+err.Error(), "CREATE_FAILED")
	}

	log.Info().
		Str("branch_slug", branch.Slug).
		Str("repository", repo).
		Int("pr_number", pr.Number).
		Msg("Created branch for PR")

	// Warmup the connection pool
	go func() {
		if err := h.router.WarmupPool(c.RequestCtx(), branch.Slug); err != nil {
			log.Warn().Err(err).Str("slug", branch.Slug).Msg("Failed to warmup branch pool")
		}
	}()

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"status":      "created",
		"branch_id":   branch.ID,
		"branch_slug": branch.Slug,
		"database":    branch.DatabaseName,
		"pr_number":   pr.Number,
	})
}

// deleteBranchForPR deletes the database branch for a pull request
func (h *GitHubWebhookHandler) deleteBranchForPR(c fiber.Ctx, repo string, pr *GitHubPullRequest) error {
	// Find branch by PR
	branch, err := h.manager.GetStorage().GetBranchByGitHubPR(c.RequestCtx(), repo, pr.Number)
	if err != nil {
		if errors.Is(err, branching.ErrBranchNotFound) {
			// Branch doesn't exist, return success
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "not_found",
				"message": "No branch exists for this PR",
			})
		}
		log.Error().Err(err).
			Str("repository", repo).
			Int("pr_number", pr.Number).
			Msg("Failed to find branch for PR")
		return SendErrorWithCode(c, fiber.StatusInternalServerError, "Failed to find branch: "+err.Error(), "FIND_FAILED")
	}

	// Close the connection pool first
	h.router.ClosePool(branch.Slug)

	// Delete the branch
	if err := h.manager.DeleteBranch(c.RequestCtx(), branch.ID, nil); err != nil {
		log.Error().Err(err).
			Str("repository", repo).
			Int("pr_number", pr.Number).
			Str("branch_id", branch.ID.String()).
			Msg("Failed to delete branch for PR")
		return SendErrorWithCode(c, fiber.StatusInternalServerError, "Failed to delete branch: "+err.Error(), "DELETE_FAILED")
	}

	log.Info().
		Str("branch_slug", branch.Slug).
		Str("repository", repo).
		Int("pr_number", pr.Number).
		Bool("merged", pr.Merged).
		Msg("Deleted branch for PR")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":      "deleted",
		"branch_slug": branch.Slug,
		"pr_number":   pr.Number,
		"merged":      pr.Merged,
	})
}

// handlePingEvent handles GitHub ping events (sent when webhook is first configured)
func (h *GitHubWebhookHandler) handlePingEvent(c fiber.Ctx, payload *GitHubWebhookPayload) error {
	repo := ""
	if payload.Repository != nil {
		repo = payload.Repository.FullName
	}

	log.Info().
		Str("repository", repo).
		Msg("Received GitHub ping event")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":  "pong",
		"message": "Webhook configured successfully",
	})
}

// handleIssueEvent handles issue events (opened, labeled, closed, etc.)
func (h *GitHubWebhookHandler) handleIssueEvent(c fiber.Ctx, payload *GitHubWebhookPayload) error {
	if payload.Issue == nil {
		return SendErrorWithCode(c, fiber.StatusBadRequest, "Missing issue in payload", "MISSING_ISSUE")
	}

	issue := payload.Issue
	repo := payload.Repository.FullName

	log.Info().
		Str("action", payload.Action).
		Int("issue_number", issue.Number).
		Str("repository", repo).
		Str("title", issue.Title).
		Msg("Processing issue event")

	switch payload.Action {
	case "opened":
		return h.handleIssueOpened(c, repo, issue, payload.Sender)

	case "labeled":
		if payload.Label == nil {
			return c.Status(fiber.StatusOK).JSON(fiber.Map{
				"status":  "ignored",
				"message": "No label in payload",
			})
		}
		return h.handleIssueLabeled(c, repo, issue, payload.Label, payload.Sender)

	case "closed":
		return h.handleIssueClosed(c, repo, issue, payload.Sender)

	case "assigned":
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "acknowledged",
			"message": "Issue assignment acknowledged",
			"issue":   issue.Number,
		})

	default:
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":  "ignored",
			"message": "Issue action not handled: " + payload.Action,
		})
	}
}

// handleIssueOpened handles when a new issue is opened
func (h *GitHubWebhookHandler) handleIssueOpened(c fiber.Ctx, repo string, issue *GitHubIssue, sender *GitHubUser) error {
	senderLogin := ""
	if sender != nil {
		senderLogin = sender.Login
	}

	log.Info().
		Str("repository", repo).
		Int("issue_number", issue.Number).
		Str("title", issue.Title).
		Str("sender", senderLogin).
		Msg("New issue opened")

	// Check if the issue has the claude-fix label from the start
	for _, label := range issue.Labels {
		if label.Name == "claude-fix" {
			log.Info().
				Int("issue_number", issue.Number).
				Msg("Issue opened with claude-fix label - automation will be triggered via GitHub Actions")
		}
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "acknowledged",
		"message":      "Issue opened event processed",
		"issue_number": issue.Number,
		"title":        issue.Title,
	})
}

// handleIssueLabeled handles when a label is added to an issue
func (h *GitHubWebhookHandler) handleIssueLabeled(c fiber.Ctx, repo string, issue *GitHubIssue, label *GitHubLabel, sender *GitHubUser) error {
	senderLogin := ""
	if sender != nil {
		senderLogin = sender.Login
	}

	log.Info().
		Str("repository", repo).
		Int("issue_number", issue.Number).
		Str("label", label.Name).
		Str("sender", senderLogin).
		Msg("Label added to issue")

	// Handle specific labels that trigger automation
	switch label.Name {
	case "claude-fix":
		log.Info().
			Int("issue_number", issue.Number).
			Str("repository", repo).
			Msg("claude-fix label detected - GitHub Actions workflow will handle automation")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":       "acknowledged",
			"message":      "claude-fix label detected, automation triggered via GitHub Actions",
			"issue_number": issue.Number,
			"label":        label.Name,
		})

	case "priority:critical", "priority:high":
		log.Warn().
			Int("issue_number", issue.Number).
			Str("repository", repo).
			Str("priority", label.Name).
			Msg("High priority issue labeled")

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":       "acknowledged",
			"message":      "High priority issue noted",
			"issue_number": issue.Number,
			"label":        label.Name,
		})

	default:
		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"status":       "acknowledged",
			"message":      "Label added",
			"issue_number": issue.Number,
			"label":        label.Name,
		})
	}
}

// handleIssueClosed handles when an issue is closed
func (h *GitHubWebhookHandler) handleIssueClosed(c fiber.Ctx, repo string, issue *GitHubIssue, sender *GitHubUser) error {
	senderLogin := ""
	if sender != nil {
		senderLogin = sender.Login
	}

	log.Info().
		Str("repository", repo).
		Int("issue_number", issue.Number).
		Str("title", issue.Title).
		Str("sender", senderLogin).
		Msg("Issue closed")

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"status":       "acknowledged",
		"message":      "Issue closed",
		"issue_number": issue.Number,
	})
}

// GetWebhookURL returns the webhook URL for configuration
func (h *GitHubWebhookHandler) GetWebhookURL(baseURL string) string {
	return strings.TrimSuffix(baseURL, "/") + "/api/v1/webhooks/github"
}

// fiber:context-methods migrated
