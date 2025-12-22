package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizePaginationParams(t *testing.T) {
	const defaultLimit = 100
	const maxLimit = 1000

	tests := []struct {
		name           string
		inputLimit     int
		inputOffset    int
		expectedLimit  int
		expectedOffset int
	}{
		{
			name:           "valid limit and offset",
			inputLimit:     50,
			inputOffset:    10,
			expectedLimit:  50,
			expectedOffset: 10,
		},
		{
			name:           "zero limit uses default",
			inputLimit:     0,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "negative limit uses default",
			inputLimit:     -10,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "limit exceeds max uses default",
			inputLimit:     1500,
			inputOffset:    0,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
		{
			name:           "exactly max limit is valid",
			inputLimit:     1000,
			inputOffset:    0,
			expectedLimit:  1000,
			expectedOffset: 0,
		},
		{
			name:           "negative offset becomes zero",
			inputLimit:     50,
			inputOffset:    -5,
			expectedLimit:  50,
			expectedOffset: 0,
		},
		{
			name:           "both invalid - uses defaults",
			inputLimit:     -1,
			inputOffset:    -1,
			expectedLimit:  defaultLimit,
			expectedOffset: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limit, offset := NormalizePaginationParams(tt.inputLimit, tt.inputOffset, defaultLimit, maxLimit)
			assert.Equal(t, tt.expectedLimit, limit)
			assert.Equal(t, tt.expectedOffset, offset)
		})
	}
}
