// Package branching provides database branching for isolated development/testing environments.
package branching

import (
	"time"

	"github.com/google/uuid"
)

// BranchStatus represents the current state of a branch
type BranchStatus string

const (
	BranchStatusCreating  BranchStatus = "creating"
	BranchStatusReady     BranchStatus = "ready"
	BranchStatusMigrating BranchStatus = "migrating"
	BranchStatusError     BranchStatus = "error"
	BranchStatusDeleting  BranchStatus = "deleting"
	BranchStatusDeleted   BranchStatus = "deleted"
)

// BranchType represents the type of branch
type BranchType string

const (
	BranchTypeMain       BranchType = "main"       // The main/production branch
	BranchTypePreview    BranchType = "preview"    // Preview branches (auto-created from PRs)
	BranchTypePersistent BranchType = "persistent" // Persistent branches (manually created, not auto-deleted)
)

// DataCloneMode specifies how data should be cloned when creating a branch
type DataCloneMode string

const (
	DataCloneModeSchemaOnly DataCloneMode = "schema_only" // Clone schema only (tables, indexes, etc.)
	DataCloneModeFullClone  DataCloneMode = "full_clone"  // Clone schema and all data
	DataCloneModeSeedData   DataCloneMode = "seed_data"   // Clone schema and run seed data scripts
)

// Branch represents a database branch
type Branch struct {
	ID             uuid.UUID     `json:"id" db:"id"`
	Name           string        `json:"name" db:"name"`
	Slug           string        `json:"slug" db:"slug"`
	DatabaseName   string        `json:"database_name" db:"database_name"`
	Status         BranchStatus  `json:"status" db:"status"`
	Type           BranchType    `json:"type" db:"type"`
	ParentBranchID *uuid.UUID    `json:"parent_branch_id,omitempty" db:"parent_branch_id"`
	DataCloneMode  DataCloneMode `json:"data_clone_mode" db:"data_clone_mode"`
	GitHubPRNumber *int          `json:"github_pr_number,omitempty" db:"github_pr_number"`
	GitHubPRURL    *string       `json:"github_pr_url,omitempty" db:"github_pr_url"`
	GitHubRepo     *string       `json:"github_repo,omitempty" db:"github_repo"`
	ErrorMessage   *string       `json:"error_message,omitempty" db:"error_message"`
	SeedsPath      *string       `json:"seeds_path,omitempty" db:"seeds_path"`
	CreatedBy      *uuid.UUID    `json:"created_by,omitempty" db:"created_by"`
	CreatedAt      time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at" db:"updated_at"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty" db:"expires_at"`
}

// IsMain returns true if this is the main branch
func (b *Branch) IsMain() bool {
	return b.Type == BranchTypeMain
}

// IsReady returns true if the branch is ready for use
func (b *Branch) IsReady() bool {
	return b.Status == BranchStatusReady
}

// MigrationHistory tracks which migrations have been applied to a branch
type MigrationHistory struct {
	ID               uuid.UUID `json:"id" db:"id"`
	BranchID         uuid.UUID `json:"branch_id" db:"branch_id"`
	MigrationVersion int64     `json:"migration_version" db:"migration_version"`
	MigrationName    *string   `json:"migration_name,omitempty" db:"migration_name"`
	AppliedAt        time.Time `json:"applied_at" db:"applied_at"`
}

// ActivityAction represents the type of activity performed on a branch
type ActivityAction string

const (
	ActivityActionCreated       ActivityAction = "created"
	ActivityActionCloned        ActivityAction = "cloned"
	ActivityActionMigrated      ActivityAction = "migrated"
	ActivityActionReset         ActivityAction = "reset"
	ActivityActionDeleted       ActivityAction = "deleted"
	ActivityActionStatusChanged ActivityAction = "status_changed"
	ActivityActionAccessGranted ActivityAction = "access_granted"
	ActivityActionAccessRevoked ActivityAction = "access_revoked"
	ActivityActionSeeding       ActivityAction = "seeding"
)

// ActivityStatus represents the outcome of an activity
type ActivityStatus string

const (
	ActivityStatusStarted ActivityStatus = "started"
	ActivityStatusSuccess ActivityStatus = "success"
	ActivityStatusFailed  ActivityStatus = "failed"
)

// ActivityLog records actions performed on branches
type ActivityLog struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	BranchID     uuid.UUID      `json:"branch_id" db:"branch_id"`
	Action       ActivityAction `json:"action" db:"action"`
	Status       ActivityStatus `json:"status" db:"status"`
	Details      any            `json:"details,omitempty" db:"details"`
	ErrorMessage *string        `json:"error_message,omitempty" db:"error_message"`
	ExecutedBy   *uuid.UUID     `json:"executed_by,omitempty" db:"executed_by"`
	ExecutedAt   time.Time      `json:"executed_at" db:"executed_at"`
	DurationMs   *int           `json:"duration_ms,omitempty" db:"duration_ms"`
}

// GitHubConfig stores GitHub integration settings per repository
type GitHubConfig struct {
	ID                   uuid.UUID     `json:"id" db:"id"`
	Repository           string        `json:"repository" db:"repository"`
	AutoCreateOnPR       bool          `json:"auto_create_on_pr" db:"auto_create_on_pr"`
	AutoDeleteOnMerge    bool          `json:"auto_delete_on_merge" db:"auto_delete_on_merge"`
	DefaultDataCloneMode DataCloneMode `json:"default_data_clone_mode" db:"default_data_clone_mode"`
	WebhookSecret        *string       `json:"-" db:"webhook_secret"` // Hidden from JSON output
	CreatedAt            time.Time     `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at" db:"updated_at"`
}

// BranchAccessLevel represents the level of access a user has to a branch
type BranchAccessLevel string

const (
	BranchAccessRead  BranchAccessLevel = "read"
	BranchAccessWrite BranchAccessLevel = "write"
	BranchAccessAdmin BranchAccessLevel = "admin"
)

// BranchAccess represents a user's access to a branch
type BranchAccess struct {
	ID          uuid.UUID         `json:"id" db:"id"`
	BranchID    uuid.UUID         `json:"branch_id" db:"branch_id"`
	UserID      uuid.UUID         `json:"user_id" db:"user_id"`
	AccessLevel BranchAccessLevel `json:"access_level" db:"access_level"`
	GrantedAt   time.Time         `json:"granted_at" db:"granted_at"`
	GrantedBy   *uuid.UUID        `json:"granted_by,omitempty" db:"granted_by"`
}

// CreateBranchRequest is the request to create a new branch
type CreateBranchRequest struct {
	Name           string        `json:"name" validate:"required,min=1,max=100"`
	ParentBranchID *uuid.UUID    `json:"parent_branch_id,omitempty"`
	DataCloneMode  DataCloneMode `json:"data_clone_mode,omitempty"`
	Type           BranchType    `json:"type,omitempty"`
	GitHubPRNumber *int          `json:"github_pr_number,omitempty"`
	GitHubPRURL    *string       `json:"github_pr_url,omitempty"`
	GitHubRepo     *string       `json:"github_repo,omitempty"`
	SeedsPath      *string       `json:"seeds_path,omitempty"`
	ExpiresAt      *time.Time    `json:"expires_at,omitempty"`
}

// ListBranchesFilter filters for listing branches
type ListBranchesFilter struct {
	Status     *BranchStatus `json:"status,omitempty"`
	Type       *BranchType   `json:"type,omitempty"`
	CreatedBy  *uuid.UUID    `json:"created_by,omitempty"`
	GitHubRepo *string       `json:"github_repo,omitempty"`
	Limit      int           `json:"limit,omitempty"`
	Offset     int           `json:"offset,omitempty"`
}
