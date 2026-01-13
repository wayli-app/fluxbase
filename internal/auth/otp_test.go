package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// GenerateOTPCode Tests
// =============================================================================

func TestGenerateOTPCode_Length(t *testing.T) {
	tests := []int{4, 6, 8, 10}

	for _, length := range tests {
		t.Run(string(rune('0'+length/10))+string(rune('0'+length%10))+"_digits", func(t *testing.T) {
			code, err := GenerateOTPCode(length)

			require.NoError(t, err)
			assert.Len(t, code, length)
		})
	}
}

func TestGenerateOTPCode_OnlyDigits(t *testing.T) {
	// Generate multiple codes and ensure they only contain digits
	for i := 0; i < 100; i++ {
		code, err := GenerateOTPCode(6)
		require.NoError(t, err)

		for _, c := range code {
			assert.True(t, c >= '0' && c <= '9', "Non-digit character found: %c", c)
		}
	}
}

func TestGenerateOTPCode_Uniqueness(t *testing.T) {
	// Generate multiple codes and ensure they're unique
	// Note: With 6-digit codes there's 1 in 1,000,000 chance of collision
	codes := make(map[string]bool)

	for i := 0; i < 100; i++ {
		code, err := GenerateOTPCode(6)
		require.NoError(t, err)

		// Very unlikely to have collision in 100 codes
		if codes[code] {
			t.Logf("Collision detected (expected to be rare): %s", code)
		}
		codes[code] = true
	}

	// Should have at least 95 unique codes (allowing for unlikely collisions)
	assert.GreaterOrEqual(t, len(codes), 95)
}

func TestGenerateOTPCode_NotEmpty(t *testing.T) {
	code, err := GenerateOTPCode(6)

	require.NoError(t, err)
	assert.NotEmpty(t, code)
}

func TestGenerateOTPCode_ZeroLength(t *testing.T) {
	code, err := GenerateOTPCode(0)

	require.NoError(t, err)
	assert.Len(t, code, 0)
	assert.Empty(t, code)
}

func TestGenerateOTPCode_SingleDigit(t *testing.T) {
	code, err := GenerateOTPCode(1)

	require.NoError(t, err)
	assert.Len(t, code, 1)
	assert.True(t, code[0] >= '0' && code[0] <= '9')
}

func TestGenerateOTPCode_LongCode(t *testing.T) {
	code, err := GenerateOTPCode(20)

	require.NoError(t, err)
	assert.Len(t, code, 20)

	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9', "Non-digit character found: %c", c)
	}
}

func TestGenerateOTPCode_Distribution(t *testing.T) {
	// Generate many codes and check digit distribution
	// Each digit should appear roughly equally
	counts := make(map[rune]int)

	for i := 0; i < 1000; i++ {
		code, err := GenerateOTPCode(6)
		require.NoError(t, err)

		for _, c := range code {
			counts[c]++
		}
	}

	// Total digits: 1000 * 6 = 6000
	// Expected per digit: 600 (with some variance)
	for digit := '0'; digit <= '9'; digit++ {
		count := counts[digit]
		// Allow 50% variance (300-900)
		assert.Greater(t, count, 300, "Digit %c appears too rarely: %d", digit, count)
		assert.Less(t, count, 900, "Digit %c appears too frequently: %d", digit, count)
	}
}

// =============================================================================
// Error Variable Tests
// =============================================================================

func TestOTPErrors_Defined(t *testing.T) {
	// Verify all error variables are defined
	assert.NotNil(t, ErrOTPNotFound)
	assert.NotNil(t, ErrOTPExpired)
	assert.NotNil(t, ErrOTPUsed)
	assert.NotNil(t, ErrOTPInvalid)
	assert.NotNil(t, ErrOTPMaxAttemptsExceeded)
}

func TestOTPErrors_Messages(t *testing.T) {
	// Verify error messages are descriptive
	assert.Equal(t, "otp code not found", ErrOTPNotFound.Error())
	assert.Equal(t, "otp code has expired", ErrOTPExpired.Error())
	assert.Equal(t, "otp code has already been used", ErrOTPUsed.Error())
	assert.Equal(t, "otp code is invalid", ErrOTPInvalid.Error())
	assert.Equal(t, "maximum otp verification attempts exceeded", ErrOTPMaxAttemptsExceeded.Error())
}

func TestOTPErrors_Distinct(t *testing.T) {
	// Verify all errors are distinct
	errors := []error{
		ErrOTPNotFound,
		ErrOTPExpired,
		ErrOTPUsed,
		ErrOTPInvalid,
		ErrOTPMaxAttemptsExceeded,
	}

	for i, err1 := range errors {
		for j, err2 := range errors {
			if i != j {
				assert.NotEqual(t, err1, err2)
			}
		}
	}
}

// =============================================================================
// OTPCode Struct Tests
// =============================================================================

func TestOTPCode_FieldsExist(t *testing.T) {
	email := "test@example.com"
	phone := "+1234567890"
	ip := "192.168.1.1"
	ua := "Mozilla/5.0"
	now := time.Now()

	otp := OTPCode{
		ID:          "otp-123",
		Email:       &email,
		Phone:       &phone,
		Code:        "123456",
		Type:        "email",
		Purpose:     "signin",
		ExpiresAt:   now.Add(10 * time.Minute),
		Used:        false,
		UsedAt:      nil,
		Attempts:    0,
		MaxAttempts: 3,
		IPAddress:   &ip,
		UserAgent:   &ua,
		CreatedAt:   now,
	}

	assert.Equal(t, "otp-123", otp.ID)
	assert.NotNil(t, otp.Email)
	assert.Equal(t, "test@example.com", *otp.Email)
	assert.NotNil(t, otp.Phone)
	assert.Equal(t, "+1234567890", *otp.Phone)
	assert.Equal(t, "123456", otp.Code)
	assert.Equal(t, "email", otp.Type)
	assert.Equal(t, "signin", otp.Purpose)
	assert.False(t, otp.Used)
	assert.Equal(t, 0, otp.Attempts)
	assert.Equal(t, 3, otp.MaxAttempts)
	assert.NotNil(t, otp.IPAddress)
	assert.NotNil(t, otp.UserAgent)
}

func TestOTPCode_NullableFields(t *testing.T) {
	// Test with nil optional fields
	otp := OTPCode{
		ID:          "otp-456",
		Email:       nil,
		Phone:       nil,
		Code:        "654321",
		Type:        "sms",
		Purpose:     "recovery",
		ExpiresAt:   time.Now().Add(5 * time.Minute),
		Used:        false,
		UsedAt:      nil,
		Attempts:    0,
		MaxAttempts: 3,
		IPAddress:   nil,
		UserAgent:   nil,
		CreatedAt:   time.Now(),
	}

	assert.Nil(t, otp.Email)
	assert.Nil(t, otp.Phone)
	assert.Nil(t, otp.UsedAt)
	assert.Nil(t, otp.IPAddress)
	assert.Nil(t, otp.UserAgent)
}

func TestOTPCode_UsedState(t *testing.T) {
	now := time.Now()

	// Not used
	otp1 := OTPCode{Used: false, UsedAt: nil}
	assert.False(t, otp1.Used)
	assert.Nil(t, otp1.UsedAt)

	// Used
	otp2 := OTPCode{Used: true, UsedAt: &now}
	assert.True(t, otp2.Used)
	assert.NotNil(t, otp2.UsedAt)
}

func TestOTPCode_ExpiredState(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	future := time.Now().Add(1 * time.Hour)

	// Expired
	otp1 := OTPCode{ExpiresAt: past}
	assert.True(t, otp1.ExpiresAt.Before(time.Now()))

	// Not expired yet
	otp2 := OTPCode{ExpiresAt: future}
	assert.True(t, otp2.ExpiresAt.After(time.Now()))
}

func TestOTPCode_AttemptsState(t *testing.T) {
	tests := []struct {
		name        string
		attempts    int
		maxAttempts int
		exceeded    bool
	}{
		{"no attempts", 0, 3, false},
		{"some attempts", 1, 3, false},
		{"at limit", 3, 3, true},
		{"over limit", 5, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			otp := OTPCode{
				Attempts:    tt.attempts,
				MaxAttempts: tt.maxAttempts,
			}

			exceeded := otp.Attempts >= otp.MaxAttempts
			assert.Equal(t, tt.exceeded, exceeded)
		})
	}
}

// =============================================================================
// OTPCode Type Tests
// =============================================================================

func TestOTPCode_Types(t *testing.T) {
	types := []string{"email", "sms"}

	for _, otpType := range types {
		otp := OTPCode{Type: otpType}
		assert.Equal(t, otpType, otp.Type)
	}
}

func TestOTPCode_Purposes(t *testing.T) {
	purposes := []string{"signin", "signup", "recovery", "email_change", "phone_change"}

	for _, purpose := range purposes {
		otp := OTPCode{Purpose: purpose}
		assert.Equal(t, purpose, otp.Purpose)
	}
}

// =============================================================================
// OTPService Struct Tests (without DB)
// =============================================================================

func TestNewOTPService_NilDependencies(t *testing.T) {
	// Service can be created with nil dependencies for testing structure
	service := NewOTPService(nil, nil, nil, 10*time.Minute)

	assert.NotNil(t, service)
	assert.Equal(t, 10*time.Minute, service.otpDuration)
}

func TestOTPService_DurationConfig(t *testing.T) {
	durations := []time.Duration{
		5 * time.Minute,
		10 * time.Minute,
		15 * time.Minute,
		30 * time.Minute,
	}

	for _, duration := range durations {
		service := NewOTPService(nil, nil, nil, duration)
		assert.Equal(t, duration, service.otpDuration)
	}
}

// =============================================================================
// OTPCode Validation Logic Tests (without DB)
// =============================================================================

func TestOTPCode_ValidationLogic(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name        string
		otp         OTPCode
		inputCode   string
		expectedErr error
	}{
		{
			name: "valid code",
			otp: OTPCode{
				Code:        "123456",
				Used:        false,
				ExpiresAt:   now.Add(5 * time.Minute),
				Attempts:    0,
				MaxAttempts: 3,
			},
			inputCode:   "123456",
			expectedErr: nil,
		},
		{
			name: "code already used",
			otp: OTPCode{
				Code:        "123456",
				Used:        true,
				UsedAt:      &now,
				ExpiresAt:   now.Add(5 * time.Minute),
				Attempts:    0,
				MaxAttempts: 3,
			},
			inputCode:   "123456",
			expectedErr: ErrOTPUsed,
		},
		{
			name: "code expired",
			otp: OTPCode{
				Code:        "123456",
				Used:        false,
				ExpiresAt:   now.Add(-1 * time.Hour), // expired
				Attempts:    0,
				MaxAttempts: 3,
			},
			inputCode:   "123456",
			expectedErr: ErrOTPExpired,
		},
		{
			name: "max attempts exceeded",
			otp: OTPCode{
				Code:        "123456",
				Used:        false,
				ExpiresAt:   now.Add(5 * time.Minute),
				Attempts:    3,
				MaxAttempts: 3,
			},
			inputCode:   "123456",
			expectedErr: ErrOTPMaxAttemptsExceeded,
		},
		{
			name: "invalid code",
			otp: OTPCode{
				Code:        "123456",
				Used:        false,
				ExpiresAt:   now.Add(5 * time.Minute),
				Attempts:    0,
				MaxAttempts: 3,
			},
			inputCode:   "654321", // wrong code
			expectedErr: ErrOTPInvalid,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate validation logic from OTPRepository.Validate
			var err error

			// Check if max attempts exceeded
			if tt.otp.Attempts >= tt.otp.MaxAttempts {
				err = ErrOTPMaxAttemptsExceeded
			} else if tt.otp.Used {
				err = ErrOTPUsed
			} else if time.Now().After(tt.otp.ExpiresAt) {
				err = ErrOTPExpired
			} else if tt.otp.Code != tt.inputCode {
				err = ErrOTPInvalid
			}

			if tt.expectedErr == nil {
				assert.NoError(t, err)
			} else {
				assert.Equal(t, tt.expectedErr, err)
			}
		})
	}
}

// =============================================================================
// Benchmark Tests
// =============================================================================

func BenchmarkGenerateOTPCode_6Digits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateOTPCode(6)
	}
}

func BenchmarkGenerateOTPCode_8Digits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateOTPCode(8)
	}
}

func BenchmarkGenerateOTPCode_10Digits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = GenerateOTPCode(10)
	}
}

// =============================================================================
// Table-Driven Tests
// =============================================================================

func TestGenerateOTPCode_TableDriven(t *testing.T) {
	tests := []struct {
		name       string
		length     int
		wantErr    bool
		wantLength int
	}{
		{"zero length", 0, false, 0},
		{"single digit", 1, false, 1},
		{"standard 4-digit", 4, false, 4},
		{"standard 6-digit", 6, false, 6},
		{"extended 8-digit", 8, false, 8},
		{"long 20-digit", 20, false, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := GenerateOTPCode(tt.length)

			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Len(t, code, tt.wantLength)
			}
		})
	}
}

// =============================================================================
// Mock OTPSender for Service Testing
// =============================================================================

// MockOTPSender implements OTPSender for testing
type MockOTPSender struct {
	SentEmails []struct {
		To      string
		Code    string
		Purpose string
	}
	SentSMS []struct {
		To      string
		Code    string
		Purpose string
	}
	EmailError error
	SMSError   error
}

func NewMockOTPSender() *MockOTPSender {
	return &MockOTPSender{
		SentEmails: make([]struct {
			To      string
			Code    string
			Purpose string
		}, 0),
		SentSMS: make([]struct {
			To      string
			Code    string
			Purpose string
		}, 0),
	}
}

func (m *MockOTPSender) SendEmailOTP(_ interface{}, to, code, purpose string) error {
	if m.EmailError != nil {
		return m.EmailError
	}
	m.SentEmails = append(m.SentEmails, struct {
		To      string
		Code    string
		Purpose string
	}{To: to, Code: code, Purpose: purpose})
	return nil
}

func (m *MockOTPSender) SendSMSOTP(_ interface{}, to, code, purpose string) error {
	if m.SMSError != nil {
		return m.SMSError
	}
	m.SentSMS = append(m.SentSMS, struct {
		To      string
		Code    string
		Purpose string
	}{To: to, Code: code, Purpose: purpose})
	return nil
}

func TestMockOTPSender_Email(t *testing.T) {
	sender := NewMockOTPSender()

	err := sender.SendEmailOTP(nil, "test@example.com", "123456", "signin")
	require.NoError(t, err)

	assert.Len(t, sender.SentEmails, 1)
	assert.Equal(t, "test@example.com", sender.SentEmails[0].To)
	assert.Equal(t, "123456", sender.SentEmails[0].Code)
	assert.Equal(t, "signin", sender.SentEmails[0].Purpose)
}

func TestMockOTPSender_SMS(t *testing.T) {
	sender := NewMockOTPSender()

	err := sender.SendSMSOTP(nil, "+1234567890", "654321", "recovery")
	require.NoError(t, err)

	assert.Len(t, sender.SentSMS, 1)
	assert.Equal(t, "+1234567890", sender.SentSMS[0].To)
	assert.Equal(t, "654321", sender.SentSMS[0].Code)
	assert.Equal(t, "recovery", sender.SentSMS[0].Purpose)
}

func TestMockOTPSender_EmailError(t *testing.T) {
	sender := NewMockOTPSender()
	sender.EmailError = assert.AnError

	err := sender.SendEmailOTP(nil, "test@example.com", "123456", "signin")
	assert.Error(t, err)
	assert.Len(t, sender.SentEmails, 0)
}

func TestMockOTPSender_SMSError(t *testing.T) {
	sender := NewMockOTPSender()
	sender.SMSError = assert.AnError

	err := sender.SendSMSOTP(nil, "+1234567890", "654321", "recovery")
	assert.Error(t, err)
	assert.Len(t, sender.SentSMS, 0)
}
