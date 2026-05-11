package tenantdb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildJoinCondition(t *testing.T) {
	tests := []struct {
		name       string
		cols       string
		leftAlias  string
		rightAlias string
		expected   string
	}{
		{
			name:       "single column",
			cols:       "email",
			leftAlias:  "n",
			rightAlias: "e",
			expected:   "n.email = e.email",
		},
		{
			name:       "multiple columns",
			cols:       "name, namespace",
			leftAlias:  "n",
			rightAlias: "e",
			expected:   "n.name = e.name AND n.namespace = e.namespace",
		},
		{
			name:       "spaces around columns",
			cols:       " name , namespace ",
			leftAlias:  "n",
			rightAlias: "e",
			expected:   "n.name = e.name AND n.namespace = e.namespace",
		},
		{
			name:       "three columns",
			cols:       "a, b, c",
			leftAlias:  "n",
			rightAlias: "e",
			expected:   "n.a = e.a AND n.b = e.b AND n.c = e.c",
		},
		{
			name:       "single column with spaces",
			cols:       " col1 ",
			leftAlias:  "n",
			rightAlias: "e",
			expected:   "n.col1 = e.col1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildJoinCondition(tt.cols, tt.leftAlias, tt.rightAlias)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackfillTenantIDToDefault_SkipsWithoutDB(t *testing.T) {
	t.Skip("BackfillTenantIDToDefault requires a real database pool; tested in integration/E2E")
}
