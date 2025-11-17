package auth

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/wayli-app/fluxbase/internal/database"
)

var (
	// ErrOTPNotFound is returned when an OTP code is not found
	ErrOTPNotFound = errors.New("otp code not found")
	// ErrOTPExpired is returned when an OTP code has expired
	ErrOTPExpired = errors.New("otp code has expired")
	// ErrOTPUsed is returned when an OTP code has already been used
	ErrOTPUsed = errors.New("otp code has already been used")
	// ErrOTPInvalid is returned when an OTP code is invalid
	ErrOTPInvalid = errors.New("otp code is invalid")
	// ErrOTPMaxAttemptsExceeded is returned when max verification attempts are exceeded
	ErrOTPMaxAttemptsExceeded = errors.New("maximum otp verification attempts exceeded")
)

// OTPCode represents a one-time password code
type OTPCode struct {
	ID          string     `json:"id" db:"id"`
	Email       *string    `json:"email,omitempty" db:"email"`
	Phone       *string    `json:"phone,omitempty" db:"phone"`
	Code        string     `json:"code" db:"code"`
	Type        string     `json:"type" db:"type"`       // email, sms
	Purpose     string     `json:"purpose" db:"purpose"` // signin, signup, recovery, email_change, phone_change
	ExpiresAt   time.Time  `json:"expires_at" db:"expires_at"`
	Used        bool       `json:"used" db:"used"`
	UsedAt      *time.Time `json:"used_at,omitempty" db:"used_at"`
	Attempts    int        `json:"attempts" db:"attempts"`
	MaxAttempts int        `json:"max_attempts" db:"max_attempts"`
	IPAddress   *string    `json:"ip_address,omitempty" db:"ip_address"`
	UserAgent   *string    `json:"user_agent,omitempty" db:"user_agent"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
}

// OTPRepository handles database operations for OTP codes
type OTPRepository struct {
	db *database.Connection
}

// NewOTPRepository creates a new OTP repository
func NewOTPRepository(db *database.Connection) *OTPRepository {
	return &OTPRepository{db: db}
}

// Create creates a new OTP code
func (r *OTPRepository) Create(ctx context.Context, email *string, phone *string, otpType, purpose string, expiryDuration time.Duration) (*OTPCode, error) {
	code, err := GenerateOTPCode(6)
	if err != nil {
		return nil, fmt.Errorf("failed to generate OTP code: %w", err)
	}

	otpCode := &OTPCode{
		ID:          uuid.New().String(),
		Email:       email,
		Phone:       phone,
		Code:        code,
		Type:        otpType,
		Purpose:     purpose,
		ExpiresAt:   time.Now().Add(expiryDuration),
		Used:        false,
		Attempts:    0,
		MaxAttempts: 3,
		CreatedAt:   time.Now(),
	}

	query := `
		INSERT INTO auth.otp_codes (id, email, phone, code, type, purpose, expires_at, used, attempts, max_attempts, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, email, phone, code, type, purpose, expires_at, used, used_at, attempts, max_attempts, ip_address, user_agent, created_at
	`

	err = r.db.QueryRow(ctx, query,
		otpCode.ID,
		otpCode.Email,
		otpCode.Phone,
		otpCode.Code,
		otpCode.Type,
		otpCode.Purpose,
		otpCode.ExpiresAt,
		otpCode.Used,
		otpCode.Attempts,
		otpCode.MaxAttempts,
		otpCode.CreatedAt,
	).Scan(
		&otpCode.ID,
		&otpCode.Email,
		&otpCode.Phone,
		&otpCode.Code,
		&otpCode.Type,
		&otpCode.Purpose,
		&otpCode.ExpiresAt,
		&otpCode.Used,
		&otpCode.UsedAt,
		&otpCode.Attempts,
		&otpCode.MaxAttempts,
		&otpCode.IPAddress,
		&otpCode.UserAgent,
		&otpCode.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return otpCode, nil
}

// GetByCode retrieves an OTP code by email/phone and code
func (r *OTPRepository) GetByCode(ctx context.Context, email *string, phone *string, code string) (*OTPCode, error) {
	var query string
	var args []interface{}

	if email != nil {
		query = `
			SELECT id, email, phone, code, type, purpose, expires_at, used, used_at, attempts, max_attempts, ip_address, user_agent, created_at
			FROM auth.otp_codes
			WHERE email = $1 AND code = $2 AND used = false
			ORDER BY created_at DESC
			LIMIT 1
		`
		args = []interface{}{*email, code}
	} else if phone != nil {
		query = `
			SELECT id, email, phone, code, type, purpose, expires_at, used, used_at, attempts, max_attempts, ip_address, user_agent, created_at
			FROM auth.otp_codes
			WHERE phone = $1 AND code = $2 AND used = false
			ORDER BY created_at DESC
			LIMIT 1
		`
		args = []interface{}{*phone, code}
	} else {
		return nil, errors.New("either email or phone must be provided")
	}

	otpCode := &OTPCode{}
	err := r.db.QueryRow(ctx, query, args...).Scan(
		&otpCode.ID,
		&otpCode.Email,
		&otpCode.Phone,
		&otpCode.Code,
		&otpCode.Type,
		&otpCode.Purpose,
		&otpCode.ExpiresAt,
		&otpCode.Used,
		&otpCode.UsedAt,
		&otpCode.Attempts,
		&otpCode.MaxAttempts,
		&otpCode.IPAddress,
		&otpCode.UserAgent,
		&otpCode.CreatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrOTPNotFound
		}
		return nil, err
	}

	return otpCode, nil
}

// IncrementAttempts increments the attempt counter for an OTP code
func (r *OTPRepository) IncrementAttempts(ctx context.Context, id string) error {
	query := `
		UPDATE auth.otp_codes
		SET attempts = attempts + 1
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrOTPNotFound
	}

	return nil
}

// MarkAsUsed marks an OTP code as used
func (r *OTPRepository) MarkAsUsed(ctx context.Context, id string) error {
	query := `
		UPDATE auth.otp_codes
		SET used = true, used_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return ErrOTPNotFound
	}

	return nil
}

// Validate validates an OTP code
func (r *OTPRepository) Validate(ctx context.Context, email *string, phone *string, code string) (*OTPCode, error) {
	otpCode, err := r.GetByCode(ctx, email, phone, code)
	if err != nil {
		return nil, err
	}

	// Check if max attempts exceeded
	if otpCode.Attempts >= otpCode.MaxAttempts {
		return nil, ErrOTPMaxAttemptsExceeded
	}

	// Check if already used
	if otpCode.Used {
		return nil, ErrOTPUsed
	}

	// Check if expired
	if time.Now().After(otpCode.ExpiresAt) {
		return nil, ErrOTPExpired
	}

	// Verify the code matches
	if otpCode.Code != code {
		// Increment attempts on invalid code
		_ = r.IncrementAttempts(ctx, otpCode.ID)
		return nil, ErrOTPInvalid
	}

	return otpCode, nil
}

// DeleteExpired deletes all expired OTP codes
func (r *OTPRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM auth.otp_codes WHERE expires_at < NOW()`

	result, err := r.db.Exec(ctx, query)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected(), nil
}

// DeleteByEmail deletes all OTP codes for an email
func (r *OTPRepository) DeleteByEmail(ctx context.Context, email string) error {
	query := `DELETE FROM auth.otp_codes WHERE email = $1`

	_, err := r.db.Exec(ctx, query, email)
	return err
}

// DeleteByPhone deletes all OTP codes for a phone number
func (r *OTPRepository) DeleteByPhone(ctx context.Context, phone string) error {
	query := `DELETE FROM auth.otp_codes WHERE phone = $1`

	_, err := r.db.Exec(ctx, query, phone)
	return err
}

// GenerateOTPCode generates a secure random numeric OTP code
func GenerateOTPCode(length int) (string, error) {
	const digits = "0123456789"
	code := make([]byte, length)

	for i := range code {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		code[i] = digits[num.Int64()]
	}

	return string(code), nil
}

// OTPSender defines the interface for sending OTP codes
type OTPSender interface {
	SendEmailOTP(ctx context.Context, to, code, purpose string) error
	SendSMSOTP(ctx context.Context, to, code, purpose string) error
}

// OTPService provides OTP functionality
type OTPService struct {
	repo        *OTPRepository
	userRepo    *UserRepository
	otpSender   OTPSender
	otpDuration time.Duration
}

// NewOTPService creates a new OTP service
func NewOTPService(
	repo *OTPRepository,
	userRepo *UserRepository,
	otpSender OTPSender,
	otpDuration time.Duration,
) *OTPService {
	return &OTPService{
		repo:        repo,
		userRepo:    userRepo,
		otpSender:   otpSender,
		otpDuration: otpDuration,
	}
}

// SendEmailOTP sends an OTP code via email
func (s *OTPService) SendEmailOTP(ctx context.Context, email, purpose string) error {
	// Invalidate old OTP codes for this email
	_ = s.repo.DeleteByEmail(ctx, email)

	// Create new OTP code
	otpCode, err := s.repo.Create(ctx, &email, nil, "email", purpose, s.otpDuration)
	if err != nil {
		return fmt.Errorf("failed to create OTP code: %w", err)
	}

	// Send email
	if err := s.otpSender.SendEmailOTP(ctx, email, otpCode.Code, purpose); err != nil {
		return fmt.Errorf("failed to send OTP email: %w", err)
	}

	return nil
}

// SendSMSOTP sends an OTP code via SMS
func (s *OTPService) SendSMSOTP(ctx context.Context, phone, purpose string) error {
	// Invalidate old OTP codes for this phone
	_ = s.repo.DeleteByPhone(ctx, phone)

	// Create new OTP code
	otpCode, err := s.repo.Create(ctx, nil, &phone, "sms", purpose, s.otpDuration)
	if err != nil {
		return fmt.Errorf("failed to create OTP code: %w", err)
	}

	// Send SMS
	if err := s.otpSender.SendSMSOTP(ctx, phone, otpCode.Code, purpose); err != nil {
		return fmt.Errorf("failed to send OTP SMS: %w", err)
	}

	return nil
}

// VerifyEmailOTP verifies an OTP code sent via email
func (s *OTPService) VerifyEmailOTP(ctx context.Context, email, code string) (*OTPCode, error) {
	// Validate the code
	otpCode, err := s.repo.Validate(ctx, &email, nil, code)
	if err != nil {
		return nil, err
	}

	// Mark as used
	if err := s.repo.MarkAsUsed(ctx, otpCode.ID); err != nil {
		return nil, fmt.Errorf("failed to mark OTP code as used: %w", err)
	}

	return otpCode, nil
}

// VerifySMSOTP verifies an OTP code sent via SMS
func (s *OTPService) VerifySMSOTP(ctx context.Context, phone, code string) (*OTPCode, error) {
	// Validate the code
	otpCode, err := s.repo.Validate(ctx, nil, &phone, code)
	if err != nil {
		return nil, err
	}

	// Mark as used
	if err := s.repo.MarkAsUsed(ctx, otpCode.ID); err != nil {
		return nil, fmt.Errorf("failed to mark OTP code as used: %w", err)
	}

	return otpCode, nil
}

// ResendEmailOTP resends an OTP code to an email
func (s *OTPService) ResendEmailOTP(ctx context.Context, email, purpose string) error {
	return s.SendEmailOTP(ctx, email, purpose)
}

// ResendSMSOTP resends an OTP code to a phone number
func (s *OTPService) ResendSMSOTP(ctx context.Context, phone, purpose string) error {
	return s.SendSMSOTP(ctx, phone, purpose)
}
