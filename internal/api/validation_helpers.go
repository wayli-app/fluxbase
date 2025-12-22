package api

// NormalizePaginationParams validates and normalizes limit/offset pagination parameters.
// It enforces the maximum limit and ensures offset is non-negative.
// Returns the normalized (limit, offset) values.
func NormalizePaginationParams(limit, offset, defaultLimit, maxLimit int) (int, int) {
	// Enforce maximum and minimum limit
	if limit <= 0 || limit > maxLimit {
		limit = defaultLimit
	}

	// Ensure offset is non-negative
	if offset < 0 {
		offset = 0
	}

	return limit, offset
}
