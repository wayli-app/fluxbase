// +build integration

package email

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MailHogMessage represents an email message from MailHog API
type MailHogMessage struct {
	ID      string `json:"ID"`
	From    struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"From"`
	To []struct {
		Mailbox string `json:"Mailbox"`
		Domain  string `json:"Domain"`
	} `json:"To"`
	Content struct {
		Headers map[string][]string `json:"Headers"`
		Body    string              `json:"Body"`
	} `json:"Content"`
	Created time.Time `json:"Created"`
}

// MailHogMessages represents the response from MailHog messages API
type MailHogMessages struct {
	Total int              `json:"total"`
	Count int              `json:"count"`
	Start int              `json:"start"`
	Items []MailHogMessage `json:"items"`
}

// getMailHogConfig returns configuration for MailHog SMTP server
func getMailHogConfig() *config.EmailConfig {
	mailhogHost := os.Getenv("MAILHOG_HOST")
	if mailhogHost == "" {
		mailhogHost = "localhost"
	}

	return &config.EmailConfig{
		Enabled:         true,
		Provider:        "smtp",
		SMTPHost:        mailhogHost,
		SMTPPort:        1025,
		SMTPUsername:    "",
		SMTPPassword:    "",
		SMTPTLS:         false,
		FromAddress:     "test@fluxbase.eu",
		FromName:        "Fluxbase Test",
		ReplyToAddress:  "reply@fluxbase.eu",
		MagicLinkExpiry: 15 * time.Minute,
	}
}

// getMailHogAPIURL returns the MailHog API URL
func getMailHogAPIURL() string {
	mailhogHost := os.Getenv("MAILHOG_HOST")
	if mailhogHost == "" {
		mailhogHost = "localhost"
	}
	return fmt.Sprintf("http://%s:8025/api", mailhogHost)
}

// deleteAllMailHogMessages deletes all messages from MailHog
func deleteAllMailHogMessages(t *testing.T) {
	req, err := http.NewRequest("DELETE", getMailHogAPIURL()+"/v1/messages", nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Logf("Warning: Could not delete MailHog messages: %v", err)
		return
	}
	defer resp.Body.Close()
}

// getMailHogMessages fetches all messages from MailHog
func getMailHogMessages(t *testing.T) []MailHogMessage {
	resp, err := http.Get(getMailHogAPIURL() + "/v2/messages")
	if err != nil {
		t.Fatalf("Failed to fetch MailHog messages: %v", err)
	}
	defer resp.Body.Close()

	var messages MailHogMessages
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("Failed to decode MailHog response: %v", err)
	}

	return messages.Items
}

// waitForEmail waits for an email to arrive in MailHog
func waitForEmail(t *testing.T, timeout time.Duration, checkFn func(MailHogMessage) bool) *MailHogMessage {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		messages := getMailHogMessages(t)

		for _, msg := range messages {
			if checkFn(msg) {
				return &msg
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func TestSMTPService_SendMagicLink_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Clean up before test
	deleteAllMailHogMessages(t)

	cfg := getMailHogConfig()
	service := NewSMTPService(cfg)

	ctx := context.Background()
	to := "user@example.com"
	token := "test-magic-link-token-123"
	link := "https://example.com/auth/verify?token=" + token

	// Send magic link email
	err := service.SendMagicLink(ctx, to, token, link)
	require.NoError(t, err)

	// Wait for email to arrive
	msg := waitForEmail(t, 5*time.Second, func(m MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox == "user" && m.To[0].Domain == "example.com"
	})

	require.NotNil(t, msg, "Email not received within timeout")

	// Verify email content
	assert.Equal(t, "test", msg.From.Mailbox)
	assert.Equal(t, "fluxbase.eu", msg.From.Domain)
	assert.Contains(t, msg.Content.Body, link)
	assert.Contains(t, msg.Content.Body, "Your Login Link")
	assert.Contains(t, msg.Content.Headers["Subject"], "Your login link")

	// Verify Reply-To header
	replyTo := msg.Content.Headers["Reply-To"]
	require.NotEmpty(t, replyTo)
	assert.Contains(t, replyTo[0], "reply@fluxbase.eu")
}

func TestSMTPService_SendVerificationEmail_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Clean up before test
	deleteAllMailHogMessages(t)

	cfg := getMailHogConfig()
	service := NewSMTPService(cfg)

	ctx := context.Background()
	to := "verify@example.com"
	token := "test-verification-token-456"
	link := "https://example.com/auth/verify-email?token=" + token

	// Send verification email
	err := service.SendVerificationEmail(ctx, to, token, link)
	require.NoError(t, err)

	// Wait for email to arrive
	msg := waitForEmail(t, 5*time.Second, func(m MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox == "verify" && m.To[0].Domain == "example.com"
	})

	require.NotNil(t, msg, "Email not received within timeout")

	// Verify email content
	assert.Contains(t, msg.Content.Body, link)
	assert.Contains(t, msg.Content.Body, "Verify Your Email")
	assert.Contains(t, msg.Content.Headers["Subject"], "Verify your email address")
}

func TestSMTPService_Send_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Clean up before test
	deleteAllMailHogMessages(t)

	cfg := getMailHogConfig()
	service := NewSMTPService(cfg)

	ctx := context.Background()
	to := "custom@example.com"
	subject := "Test Custom Email"
	body := "<html><body><h1>Test Body</h1><p>This is a custom email</p></body></html>"

	// Send custom email
	err := service.Send(ctx, to, subject, body)
	require.NoError(t, err)

	// Wait for email to arrive
	msg := waitForEmail(t, 5*time.Second, func(m MailHogMessage) bool {
		return len(m.To) > 0 && m.To[0].Mailbox == "custom" && m.To[0].Domain == "example.com"
	})

	require.NotNil(t, msg, "Email not received within timeout")

	// Verify email content
	assert.Contains(t, msg.Content.Headers["Subject"], subject)
	assert.Contains(t, msg.Content.Body, "Test Body")
	assert.Contains(t, msg.Content.Body, "This is a custom email")
}

func TestSMTPService_MultipleEmails_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Clean up before test
	deleteAllMailHogMessages(t)

	cfg := getMailHogConfig()
	service := NewSMTPService(cfg)

	ctx := context.Background()

	// Send multiple emails
	recipients := []string{"user1@example.com", "user2@example.com", "user3@example.com"}
	for i, to := range recipients {
		err := service.Send(ctx, to, fmt.Sprintf("Test Email %d", i+1), fmt.Sprintf("Body %d", i+1))
		require.NoError(t, err)
	}

	// Wait for all emails to arrive
	time.Sleep(1 * time.Second)

	messages := getMailHogMessages(t)
	assert.GreaterOrEqual(t, len(messages), 3, "Expected at least 3 emails")

	// Verify all recipients received emails
	receivedTo := make(map[string]bool)
	for _, msg := range messages {
		if len(msg.To) > 0 {
			email := msg.To[0].Mailbox + "@" + msg.To[0].Domain
			receivedTo[email] = true
		}
	}

	for _, recipient := range recipients {
		assert.True(t, receivedTo[recipient], "Expected email to %s", recipient)
	}
}

func TestMailHogConnectivity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Test that MailHog API is accessible
	resp, err := http.Get(getMailHogAPIURL() + "/v2/messages")
	require.NoError(t, err, "MailHog should be accessible at %s", getMailHogAPIURL())
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "MailHog API should return 200")
}
