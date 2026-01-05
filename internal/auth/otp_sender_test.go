package auth

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockEmailService is a mock implementation of RealEmailService for testing
type MockEmailService struct {
	sendCalls             []SendCall
	sendMagicLinkCalls    []SendMagicLinkCall
	sendPasswordResetCalls []SendPasswordResetCall
	sendVerificationCalls []SendVerificationCall
	sendError             error
	configured            bool
}

type SendCall struct {
	To      string
	Subject string
	Body    string
}

type SendMagicLinkCall struct {
	To    string
	Token string
	Link  string
}

type SendPasswordResetCall struct {
	To    string
	Token string
	Link  string
}

type SendVerificationCall struct {
	To    string
	Token string
	Link  string
}

func (m *MockEmailService) Send(ctx context.Context, to, subject, body string) error {
	m.sendCalls = append(m.sendCalls, SendCall{
		To:      to,
		Subject: subject,
		Body:    body,
	})
	return m.sendError
}

func (m *MockEmailService) SendMagicLink(ctx context.Context, to, token, link string) error {
	m.sendMagicLinkCalls = append(m.sendMagicLinkCalls, SendMagicLinkCall{
		To:    to,
		Token: token,
		Link:  link,
	})
	return m.sendError
}

func (m *MockEmailService) SendPasswordReset(ctx context.Context, to, token, link string) error {
	m.sendPasswordResetCalls = append(m.sendPasswordResetCalls, SendPasswordResetCall{
		To:    to,
		Token: token,
		Link:  link,
	})
	return m.sendError
}

func (m *MockEmailService) SendVerificationEmail(ctx context.Context, to, token, link string) error {
	m.sendVerificationCalls = append(m.sendVerificationCalls, SendVerificationCall{
		To:    to,
		Token: token,
		Link:  link,
	})
	return m.sendError
}

func (m *MockEmailService) IsConfigured() bool {
	return m.configured
}

func TestNewDefaultOTPSender(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}

	sender := NewDefaultOTPSender(mockEmail, "test@example.com", "TestApp")

	assert.NotNil(t, sender)
	assert.Equal(t, mockEmail, sender.emailService)
	assert.Equal(t, "test@example.com", sender.fromAddress)
	assert.Equal(t, "TestApp", sender.appName)
}

func TestNewDefaultOTPSender_EmptyAppName(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}

	sender := NewDefaultOTPSender(mockEmail, "test@example.com", "")

	assert.NotNil(t, sender)
	assert.Equal(t, "Fluxbase", sender.appName, "should use default app name")
}

func TestNewDefaultOTPSender_EmptyFromAddress(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}

	sender := NewDefaultOTPSender(mockEmail, "", "TestApp")

	assert.NotNil(t, sender)
	assert.Equal(t, "noreply@fluxbase.app", sender.fromAddress, "should use default from address")
}

func TestNewDefaultOTPSender_BothEmpty(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}

	sender := NewDefaultOTPSender(mockEmail, "", "")

	assert.NotNil(t, sender)
	assert.Equal(t, "Fluxbase", sender.appName)
	assert.Equal(t, "noreply@fluxbase.app", sender.fromAddress)
}

func TestGetEmailSubject(t *testing.T) {
	tests := []struct {
		name            string
		purpose         string
		appName         string
		expectedSubject string
	}{
		{
			name:            "signin purpose",
			purpose:         "signin",
			appName:         "Fluxbase",
			expectedSubject: "Your Fluxbase Sign In Code",
		},
		{
			name:            "signup purpose",
			purpose:         "signup",
			appName:         "Fluxbase",
			expectedSubject: "Verify your Fluxbase account",
		},
		{
			name:            "recovery purpose",
			purpose:         "recovery",
			appName:         "Fluxbase",
			expectedSubject: "Your Fluxbase Account Recovery Code",
		},
		{
			name:            "email_change purpose",
			purpose:         "email_change",
			appName:         "Fluxbase",
			expectedSubject: "Verify your new Fluxbase email",
		},
		{
			name:            "phone_change purpose",
			purpose:         "phone_change",
			appName:         "Fluxbase",
			expectedSubject: "Verify your new Fluxbase phone",
		},
		{
			name:            "unknown purpose",
			purpose:         "unknown",
			appName:         "Fluxbase",
			expectedSubject: "Your Fluxbase Verification Code",
		},
		{
			name:            "empty purpose",
			purpose:         "",
			appName:         "Fluxbase",
			expectedSubject: "Your Fluxbase Verification Code",
		},
		{
			name:            "custom app name",
			purpose:         "signin",
			appName:         "MyApp",
			expectedSubject: "Your MyApp Sign In Code",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := &MockEmailService{configured: true}
			sender := NewDefaultOTPSender(mockEmail, "test@example.com", tt.appName)

			subject := sender.getEmailSubject(tt.purpose)

			assert.Equal(t, tt.expectedSubject, subject)
		})
	}
}

func TestGetEmailBody(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		purpose         string
		appName         string
		expectedAction  string
		expectedContent []string
	}{
		{
			name:           "signin purpose",
			code:           "123456",
			purpose:        "signin",
			appName:        "Fluxbase",
			expectedAction: "sign in to your account",
			expectedContent: []string{
				"123456",
				"sign in to your account",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "signup purpose",
			code:           "654321",
			purpose:        "signup",
			appName:        "Fluxbase",
			expectedAction: "complete your account registration",
			expectedContent: []string{
				"654321",
				"complete your account registration",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "recovery purpose",
			code:           "999888",
			purpose:        "recovery",
			appName:        "Fluxbase",
			expectedAction: "recover your account",
			expectedContent: []string{
				"999888",
				"recover your account",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "email_change purpose",
			code:           "111222",
			purpose:        "email_change",
			appName:        "Fluxbase",
			expectedAction: "verify your new email address",
			expectedContent: []string{
				"111222",
				"verify your new email address",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "phone_change purpose",
			code:           "333444",
			purpose:        "phone_change",
			appName:        "Fluxbase",
			expectedAction: "verify your new phone number",
			expectedContent: []string{
				"333444",
				"verify your new phone number",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "unknown purpose",
			code:           "555666",
			purpose:        "unknown",
			appName:        "Fluxbase",
			expectedAction: "complete verification",
			expectedContent: []string{
				"555666",
				"complete verification",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "empty purpose",
			code:           "777888",
			purpose:        "",
			appName:        "Fluxbase",
			expectedAction: "complete verification",
			expectedContent: []string{
				"777888",
				"complete verification",
				"expire in 15 minutes",
				"The Fluxbase Team",
			},
		},
		{
			name:           "custom app name",
			code:           "123456",
			purpose:        "signin",
			appName:        "MyApp",
			expectedAction: "sign in to your account",
			expectedContent: []string{
				"123456",
				"sign in to your account",
				"The MyApp Team",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := &MockEmailService{configured: true}
			sender := NewDefaultOTPSender(mockEmail, "test@example.com", tt.appName)

			body := sender.getEmailBody(tt.code, tt.purpose)

			// Verify all expected content is present
			for _, content := range tt.expectedContent {
				assert.Contains(t, body, content)
			}

			// Verify body structure
			assert.True(t, strings.HasPrefix(body, "Hello,"))
			assert.Contains(t, body, "Your verification code is:")
			assert.Contains(t, body, "didn't request this code")
		})
	}
}

func TestSendEmailOTP_Success(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}
	sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

	ctx := context.Background()
	err := sender.SendEmailOTP(ctx, "user@example.com", "123456", "signin")

	require.NoError(t, err)
	assert.Len(t, mockEmail.sendCalls, 1)

	call := mockEmail.sendCalls[0]
	assert.Equal(t, "user@example.com", call.To)
	assert.Equal(t, "Your TestApp Sign In Code", call.Subject)
	assert.Contains(t, call.Body, "123456")
	assert.Contains(t, call.Body, "sign in to your account")
}

func TestSendEmailOTP_AllPurposes(t *testing.T) {
	purposes := []string{"signin", "signup", "recovery", "email_change", "phone_change"}

	for _, purpose := range purposes {
		t.Run(purpose, func(t *testing.T) {
			mockEmail := &MockEmailService{configured: true}
			sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

			ctx := context.Background()
			err := sender.SendEmailOTP(ctx, "user@example.com", "123456", purpose)

			require.NoError(t, err)
			assert.Len(t, mockEmail.sendCalls, 1)
			assert.Equal(t, "user@example.com", mockEmail.sendCalls[0].To)
			assert.Contains(t, mockEmail.sendCalls[0].Body, "123456")
		})
	}
}

func TestSendEmailOTP_EmailServiceError(t *testing.T) {
	mockEmail := &MockEmailService{
		configured: true,
		sendError:  errors.New("email service unavailable"),
	}
	sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

	ctx := context.Background()
	err := sender.SendEmailOTP(ctx, "user@example.com", "123456", "signin")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send OTP email")
	assert.Contains(t, err.Error(), "email service unavailable")
}

func TestSendEmailOTP_DifferentCodes(t *testing.T) {
	tests := []struct {
		name string
		code string
	}{
		{"6 digit code", "123456"},
		{"4 digit code", "1234"},
		{"8 digit code", "12345678"},
		{"alphanumeric code", "ABC123"},
		{"special code", "XYZ-789"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmail := &MockEmailService{configured: true}
			sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

			ctx := context.Background()
			err := sender.SendEmailOTP(ctx, "user@example.com", tt.code, "signin")

			require.NoError(t, err)
			assert.Contains(t, mockEmail.sendCalls[0].Body, tt.code)
		})
	}
}

func TestSendSMSOTP_NotImplemented(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}
	sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

	ctx := context.Background()
	err := sender.SendSMSOTP(ctx, "+1234567890", "123456", "signin")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SMS OTP sending is not yet implemented")
}

func TestNoOpOTPSender_SendEmailOTP(t *testing.T) {
	sender := &NoOpOTPSender{}

	ctx := context.Background()
	err := sender.SendEmailOTP(ctx, "user@example.com", "123456", "signin")

	assert.NoError(t, err, "NoOpOTPSender should not return errors")
}

func TestNoOpOTPSender_SendSMSOTP(t *testing.T) {
	sender := &NoOpOTPSender{}

	ctx := context.Background()
	err := sender.SendSMSOTP(ctx, "+1234567890", "123456", "signin")

	assert.NoError(t, err, "NoOpOTPSender should not return errors")
}

func TestOTPSender_Integration(t *testing.T) {
	// Test a complete OTP sending flow
	mockEmail := &MockEmailService{configured: true}
	sender := NewDefaultOTPSender(mockEmail, "noreply@fluxbase.app", "Fluxbase")

	ctx := context.Background()

	// Send OTP for different purposes
	purposes := []struct {
		purpose string
		code    string
		email   string
	}{
		{"signin", "123456", "user1@example.com"},
		{"signup", "654321", "user2@example.com"},
		{"recovery", "999888", "user3@example.com"},
	}

	for _, p := range purposes {
		err := sender.SendEmailOTP(ctx, p.email, p.code, p.purpose)
		require.NoError(t, err)
	}

	// Verify all emails were sent
	assert.Len(t, mockEmail.sendCalls, 3)

	// Verify each call has correct recipient and code
	for i, p := range purposes {
		assert.Equal(t, p.email, mockEmail.sendCalls[i].To)
		assert.Contains(t, mockEmail.sendCalls[i].Body, p.code)
	}
}

func TestEmailBody_Format(t *testing.T) {
	mockEmail := &MockEmailService{configured: true}
	sender := NewDefaultOTPSender(mockEmail, "noreply@example.com", "TestApp")

	body := sender.getEmailBody("123456", "signin")

	// Verify body structure and formatting
	lines := strings.Split(body, "\n")
	assert.True(t, len(lines) > 5, "body should have multiple lines")

	// Check greeting
	assert.Equal(t, "Hello,", strings.TrimSpace(lines[0]))

	// Check code line format
	assert.Contains(t, body, "Your verification code is: 123456")

	// Check expiry message
	assert.Contains(t, body, "This code will expire in 15 minutes")

	// Check security message
	assert.Contains(t, body, "If you didn't request this code")

	// Check signature
	assert.Contains(t, body, "Best regards,")
	assert.Contains(t, body, "The TestApp Team")
}
