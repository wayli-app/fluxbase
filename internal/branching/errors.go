package branching

import "errors"

// Common errors for the branching package
var (
	// ErrBranchNotFound is returned when a branch cannot be found
	ErrBranchNotFound = errors.New("branch not found")

	// ErrBranchExists is returned when trying to create a branch that already exists
	ErrBranchExists = errors.New("branch already exists")

	// ErrCannotDeleteMainBranch is returned when trying to delete the main branch
	ErrCannotDeleteMainBranch = errors.New("cannot delete main branch")

	// ErrBranchNotReady is returned when an operation requires a ready branch
	ErrBranchNotReady = errors.New("branch is not ready")

	// ErrMaxBranchesReached is returned when the maximum number of branches has been reached
	ErrMaxBranchesReached = errors.New("maximum number of branches reached")

	// ErrMaxUserBranchesReached is returned when a user has reached their branch limit
	ErrMaxUserBranchesReached = errors.New("maximum branches per user reached")

	// ErrInvalidSlug is returned when a branch slug is invalid
	ErrInvalidSlug = errors.New("invalid branch slug")

	// ErrGitHubConfigNotFound is returned when GitHub config cannot be found
	ErrGitHubConfigNotFound = errors.New("github config not found")

	// ErrBranchingDisabled is returned when branching is disabled in config
	ErrBranchingDisabled = errors.New("database branching is disabled")

	// ErrAccessDenied is returned when a user doesn't have access to a branch
	ErrAccessDenied = errors.New("access denied to branch")

	// ErrDatabaseOperationFailed is returned when a database operation fails
	ErrDatabaseOperationFailed = errors.New("database operation failed")
)
