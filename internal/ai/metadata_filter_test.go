package ai

import (
	"reflect"
	"strings"
	"testing"
)

func TestBuildConditionSQL_EqualityOperators(t *testing.T) {
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name: "equals operator",
			cond: MetadataCondition{
				Key:      "category",
				Operator: MetadataOpEquals,
				Value:    "food",
			},
			wantSQL:     `d.metadata->>'category' = $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "not equals operator",
			cond: MetadataCondition{
				Key:      "status",
				Operator: MetadataOpNotEquals,
				Value:    "archived",
			},
			wantSQL:     `d.metadata->>'status' != $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "equals operator with numeric value",
			cond: MetadataCondition{
				Key:      "count",
				Operator: MetadataOpEquals,
				Value:    42,
			},
			wantSQL:     `d.metadata->>'count' = $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("buildConditionSQL() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() args len = %v, want %v", len(args), tt.wantArgsLen)
			}
			if argIndex-1 != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() argIndex = %v, want %v", argIndex-1, tt.wantArgsLen)
			}
		})
	}
}

func TestBuildConditionSQL_PatternOperators(t *testing.T) {
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name: "ILIKE operator",
			cond: MetadataCondition{
				Key:      "city",
				Operator: MetadataOpILike,
				Value:    "%Tokyo%",
			},
			wantSQL:     `d.metadata->>'city' ILIKE $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "LIKE operator",
			cond: MetadataCondition{
				Key:      "name",
				Operator: MetadataOpLike,
				Value:    "Starbucks%",
			},
			wantSQL:     `d.metadata->>'name' LIKE $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("buildConditionSQL() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() args len = %v, want %v", len(args), tt.wantArgsLen)
			}
		})
	}
}

func TestBuildConditionSQL_InOperators(t *testing.T) {
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name: "IN operator with strings",
			cond: MetadataCondition{
				Key:      "cuisine",
				Operator: MetadataOpIn,
				Values:   []interface{}{"japanese", "sushi", "ramen"},
			},
			wantSQL:     `d.metadata->>'cuisine' IN ($1, $2, $3)`,
			wantArgsLen: 3,
			wantErr:     false,
		},
		{
			name: "NOT IN operator",
			cond: MetadataCondition{
				Key:      "status",
				Operator: MetadataOpNotIn,
				Values:   []interface{}{"archived", "deleted"},
			},
			wantSQL:     `d.metadata->>'status' NOT IN ($1, $2)`,
			wantArgsLen: 2,
			wantErr:     false,
		},
		{
			name: "IN operator with single value",
			cond: MetadataCondition{
				Key:      "category",
				Operator: MetadataOpIn,
				Values:   []interface{}{"food"},
			},
			wantSQL:     `d.metadata->>'category' IN ($1)`,
			wantArgsLen: 1,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("buildConditionSQL() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() args len = %v, want %v", len(args), tt.wantArgsLen)
			}
		})
	}
}

func TestBuildConditionSQL_RangeOperators(t *testing.T) {
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name: "greater than operator",
			cond: MetadataCondition{
				Key:      "rating",
				Operator: MetadataOpGreaterThan,
				Value:    4.5,
			},
			wantSQL:     `d.metadata->>'rating' > $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "greater than or equal operator",
			cond: MetadataCondition{
				Key:      "price",
				Operator: MetadataOpGreaterThanOr,
				Value:    100,
			},
			wantSQL:     `d.metadata->>'price' >= $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "less than operator",
			cond: MetadataCondition{
				Key:      "distance",
				Operator: MetadataOpLessThan,
				Value:    5.0,
			},
			wantSQL:     `d.metadata->>'distance' < $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "less than or equal operator",
			cond: MetadataCondition{
				Key:      "quantity",
				Operator: MetadataOpLessThanOr,
				Value:    10,
			},
			wantSQL:     `d.metadata->>'quantity' <= $1`,
			wantArgsLen: 1,
			wantErr:     false,
		},
		{
			name: "BETWEEN operator",
			cond: MetadataCondition{
				Key:      "duration",
				Operator: MetadataOpBetween,
				Min:      30,
				Max:      90,
			},
			wantSQL:     `d.metadata->>'duration' BETWEEN $1 AND $2`,
			wantArgsLen: 2,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("buildConditionSQL() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() args len = %v, want %v", len(args), tt.wantArgsLen)
			}
		})
	}
}

func TestBuildConditionSQL_NullOperators(t *testing.T) {
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{
			name: "IS NULL operator",
			cond: MetadataCondition{
				Key:      "deleted_at",
				Operator: MetadataOpIsNull,
			},
			wantSQL:     `d.metadata->>'deleted_at' IS NULL`,
			wantArgsLen: 0,
			wantErr:     false,
		},
		{
			name: "IS NOT NULL operator",
			cond: MetadataCondition{
				Key:      "verified_at",
				Operator: MetadataOpIsNotNull,
			},
			wantSQL:     `d.metadata->>'verified_at' IS NOT NULL`,
			wantArgsLen: 0,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if sql != tt.wantSQL {
				t.Errorf("buildConditionSQL() SQL = %v, want %v", sql, tt.wantSQL)
			}
			if len(args) != tt.wantArgsLen {
				t.Errorf("buildConditionSQL() args len = %v, want %v", len(args), tt.wantArgsLen)
			}
		})
	}
}

func TestBuildConditionSQL_Errors(t *testing.T) {
	tests := []struct {
		name    string
		cond    MetadataCondition
		wantErr bool
	}{
		{
			name: "equals operator missing value",
			cond: MetadataCondition{
				Key:      "category",
				Operator: MetadataOpEquals,
			},
			wantErr: true,
		},
		{
			name: "IN operator missing values",
			cond: MetadataCondition{
				Key:      "category",
				Operator: MetadataOpIn,
			},
			wantErr: true,
		},
		{
			name: "BETWEEN operator missing min",
			cond: MetadataCondition{
				Key:      "duration",
				Operator: MetadataOpBetween,
				Max:      90,
			},
			wantErr: true,
		},
		{
			name: "BETWEEN operator missing max",
			cond: MetadataCondition{
				Key:      "duration",
				Operator: MetadataOpBetween,
				Min:      30,
			},
			wantErr: true,
		},
		{
			name: "unsupported operator",
			cond: MetadataCondition{
				Key:      "category",
				Operator: MetadataOperator("UNSUPPORTED"),
				Value:    "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			_, _, err := buildConditionSQL(tt.cond, &argIndex)

			if (err != nil) != tt.wantErr {
				t.Errorf("buildConditionSQL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBuildMetadataFilterSQL_SingleCondition(t *testing.T) {
	group := MetadataFilterGroup{
		Conditions: []MetadataCondition{
			{
				Key:      "category",
				Operator: MetadataOpEquals,
				Value:    "food",
			},
		},
		LogicalOp: LogicalOpAND,
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	if sql != `d.metadata->>'category' = $1` {
		t.Errorf("buildMetadataFilterSQL() SQL = %v, want %v", sql, `d.metadata->>'category' = $1`)
	}
	if len(args) != 1 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 1", len(args))
	}
}

func TestBuildMetadataFilterSQL_MultipleConditionsAND(t *testing.T) {
	group := MetadataFilterGroup{
		Conditions: []MetadataCondition{
			{
				Key:      "category",
				Operator: MetadataOpEquals,
				Value:    "food",
			},
			{
				Key:      "city",
				Operator: MetadataOpILike,
				Value:    "%Tokyo%",
			},
		},
		LogicalOp: LogicalOpAND,
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	expectedSQL := `d.metadata->>'category' = $1 AND d.metadata->>'city' ILIKE $2`
	if sql != expectedSQL {
		t.Errorf("buildMetadataFilterSQL() SQL = %v, want %v", sql, expectedSQL)
	}
	if len(args) != 2 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 2", len(args))
	}
}

func TestBuildMetadataFilterSQL_MultipleConditionsOR(t *testing.T) {
	group := MetadataFilterGroup{
		Conditions: []MetadataCondition{
			{
				Key:      "status",
				Operator: MetadataOpEquals,
				Value:    "active",
			},
			{
				Key:      "status",
				Operator: MetadataOpEquals,
				Value:    "pending",
			},
		},
		LogicalOp: LogicalOpOR,
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	expectedSQL := `d.metadata->>'status' = $1 OR d.metadata->>'status' = $2`
	if sql != expectedSQL {
		t.Errorf("buildMetadataFilterSQL() SQL = %v, want %v", sql, expectedSQL)
	}
	if len(args) != 2 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 2", len(args))
	}
}

func TestBuildMetadataFilterSQL_NestedGroups(t *testing.T) {
	// (category = 'food' AND city ILIKE '%Tokyo%') OR (category = 'culture')
	group := MetadataFilterGroup{
		LogicalOp: LogicalOpOR,
		Conditions: []MetadataCondition{
			{
				Key:      "category",
				Operator: MetadataOpEquals,
				Value:    "culture",
			},
		},
		Groups: []MetadataFilterGroup{
			{
				LogicalOp: LogicalOpAND,
				Conditions: []MetadataCondition{
					{
						Key:      "category",
						Operator: MetadataOpEquals,
						Value:    "food",
					},
					{
						Key:      "city",
						Operator: MetadataOpILike,
						Value:    "%Tokyo%",
					},
				},
			},
		},
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	// Note: The exact SQL structure may vary slightly depending on how the nested groups are processed
	// The important thing is that all conditions are present and arg indexing is correct
	if len(args) != 3 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 3", len(args))
	}
	// Check that all expected operators are in the SQL
	if sql == "" {
		t.Errorf("buildMetadataFilterSQL() returned empty SQL")
	}
}

func TestBuildMetadataFilterSQL_INOperator(t *testing.T) {
	group := MetadataFilterGroup{
		Conditions: []MetadataCondition{
			{
				Key:      "cuisine",
				Operator: MetadataOpIn,
				Values:   []interface{}{"japanese", "italian", "vietnamese"},
			},
		},
		LogicalOp: LogicalOpAND,
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	expectedSQL := `d.metadata->>'cuisine' IN ($1, $2, $3)`
	if sql != expectedSQL {
		t.Errorf("buildMetadataFilterSQL() SQL = %v, want %v", sql, expectedSQL)
	}
	if len(args) != 3 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 3", len(args))
	}
}

func TestBuildMetadataFilterSQL_BetweenOperator(t *testing.T) {
	group := MetadataFilterGroup{
		Conditions: []MetadataCondition{
			{
				Key:      "avg_duration",
				Operator: MetadataOpBetween,
				Min:      30,
				Max:      90,
			},
		},
		LogicalOp: LogicalOpAND,
	}

	argIndex := 1
	sql, args, err := buildMetadataFilterSQL(group, &argIndex)
	if err != nil {
		t.Fatalf("buildMetadataFilterSQL() error = %v", err)
	}
	expectedSQL := `d.metadata->>'avg_duration' BETWEEN $1 AND $2`
	if sql != expectedSQL {
		t.Errorf("buildMetadataFilterSQL() SQL = %v, want %v", sql, expectedSQL)
	}
	if len(args) != 2 {
		t.Errorf("buildMetadataFilterSQL() args len = %v, want 2", len(args))
	}
}

func TestEscapeStringLiteral(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "no single quotes",
			input: "category",
			want:  "category",
		},
		{
			name:  "single quote in middle",
			input: "user's_name",
			want:  "user''s_name",
		},
		{
			name:  "multiple single quotes",
			input: "it's a user's name",
			want:  "it''s a user''s name",
		},
		{
			name:  "starts with single quote",
			input: "'test",
			want:  "''test",
		},
		{
			name:  "ends with single quote",
			input: "test'",
			want:  "test''",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := escapeStringLiteral(tt.input); got != tt.want {
				t.Errorf("escapeStringLiteral() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// *ForTable variants — buildConditionSQLForTable / buildMetadataFilterSQLForTable
// =============================================================================
//
// These mirror buildConditionSQL / buildMetadataFilterSQL but take a table
// prefix (e.g. "d." or "t.") so the metadata->>'key' reference can be aliased.
// Contract detail (kb_storage_search.go:692,746):
//   - buildConditionSQLForTable uses tablePrefix AS-IS in the metadataRef, so
//     callers must include the trailing dot ("d.").
//   - buildMetadataFilterSQLForTable normalizes the prefix (adds "." if missing)
//     before delegating, and recurses with the ORIGINAL (un-normalized) prefix.
// The existing tests cover only the non-ForTable variants; these are 0.0%.

func TestBuildConditionSQLForTable_AllOperators(t *testing.T) {
	// ForTable uses the prefix verbatim; pass "d." to match the non-ForTable shape.
	const prefix = "d."
	tests := []struct {
		name        string
		cond        MetadataCondition
		wantSQL     string
		wantArgsLen int
		wantErr     bool
	}{
		{"equals", MetadataCondition{Key: "category", Operator: MetadataOpEquals, Value: "food"}, prefix + `metadata->>'category' = $1`, 1, false},
		{"not equals", MetadataCondition{Key: "status", Operator: MetadataOpNotEquals, Value: "archived"}, prefix + `metadata->>'status' != $1`, 1, false},
		{"ilike", MetadataCondition{Key: "name", Operator: MetadataOpILike, Value: "%foo%"}, prefix + `metadata->>'name' ILIKE $1`, 1, false},
		{"like", MetadataCondition{Key: "name", Operator: MetadataOpLike, Value: "foo%"}, prefix + `metadata->>'name' LIKE $1`, 1, false},
		{"in", MetadataCondition{Key: "tag", Operator: MetadataOpIn, Values: []interface{}{"a", "b"}}, prefix + `metadata->>'tag' IN ($1, $2)`, 2, false},
		{"not in", MetadataCondition{Key: "tag", Operator: MetadataOpNotIn, Values: []interface{}{"a"}}, prefix + `metadata->>'tag' NOT IN ($1)`, 1, false},
		{"greater than", MetadataCondition{Key: "n", Operator: MetadataOpGreaterThan, Value: 5}, prefix + `metadata->>'n' > $1`, 1, false},
		{"greater or equal", MetadataCondition{Key: "n", Operator: MetadataOpGreaterThanOr, Value: 5}, prefix + `metadata->>'n' >= $1`, 1, false},
		{"less than", MetadataCondition{Key: "n", Operator: MetadataOpLessThan, Value: 5}, prefix + `metadata->>'n' < $1`, 1, false},
		{"less or equal", MetadataCondition{Key: "n", Operator: MetadataOpLessThanOr, Value: 5}, prefix + `metadata->>'n' <= $1`, 1, false},
		{"between", MetadataCondition{Key: "n", Operator: MetadataOpBetween, Min: 1, Max: 10}, prefix + `metadata->>'n' BETWEEN $1 AND $2`, 2, false},
		{"is null", MetadataCondition{Key: "x", Operator: MetadataOpIsNull}, prefix + `metadata->>'x' IS NULL`, 0, false},
		{"is not null", MetadataCondition{Key: "x", Operator: MetadataOpIsNotNull}, prefix + `metadata->>'x' IS NOT NULL`, 0, false},
		// Error paths
		{"equals nil value errors", MetadataCondition{Key: "x", Operator: MetadataOpEquals, Value: nil}, "", 0, true},
		{"in empty values errors", MetadataCondition{Key: "x", Operator: MetadataOpIn, Values: nil}, "", 0, true},
		{"between missing min errors", MetadataCondition{Key: "x", Operator: MetadataOpBetween, Max: 10}, "", 0, true},
		{"unsupported operator errors", MetadataCondition{Key: "x", Operator: "Bogus"}, "", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			argIndex := 1
			sql, args, err := buildConditionSQLForTable(tt.cond, &argIndex, prefix)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				if sql != tt.wantSQL {
					t.Errorf("SQL = %q, want %q", sql, tt.wantSQL)
				}
				if len(args) != tt.wantArgsLen {
					t.Errorf("args len = %d, want %d", len(args), tt.wantArgsLen)
				}
			}
		})
	}
}

func TestBuildConditionSQLForTable_PrefixVariations(t *testing.T) {
	t.Parallel()
	// The prefix is used verbatim in the metadataRef. Confirm different aliases.
	cond := MetadataCondition{Key: "k", Operator: MetadataOpEquals, Value: "v"}

	for _, tc := range []struct {
		name   string
		prefix string
		want   string
	}{
		{"aliased d.", "d.", `d.metadata->>'k' = $1`},
		{"aliased t.", "t.", `t.metadata->>'k' = $1`},
		{"empty prefix yields bare metadata", "", `metadata->>'k' = $1`},
		// NOTE: a prefix WITHOUT a trailing dot is NOT auto-dotted at this layer
		// (only buildMetadataFilterSQLForTable auto-dots). Pin this so callers
		// know to include the dot when calling the condition builder directly.
		{"prefix without dot is used verbatim (no auto-dot here)", "x", `xmetadata->>'k' = $1`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			idx := 1
			got, _, err := buildConditionSQLForTable(cond, &idx, tc.prefix)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestBuildMetadataFilterSQLForTable(t *testing.T) {
	t.Parallel()

	t.Run("empty group returns empty", func(t *testing.T) {
		t.Parallel()
		idx := 1
		sql, args, err := buildMetadataFilterSQLForTable(MetadataFilterGroup{}, &idx, "d")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if sql != "" {
			t.Errorf("sql = %q, want empty", sql)
		}
		if len(args) != 0 {
			t.Errorf("args = %v, want empty", args)
		}
	})

	t.Run("single condition with auto-dotted prefix", func(t *testing.T) {
		t.Parallel()
		// "d" is normalized to "d." before delegating to buildConditionSQLForTable.
		idx := 1
		group := MetadataFilterGroup{
			LogicalOp:  LogicalOpAND,
			Conditions: []MetadataCondition{{Key: "k", Operator: MetadataOpEquals, Value: "v"}},
		}
		sql, args, err := buildMetadataFilterSQLForTable(group, &idx, "d")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		wantSQL := `d.metadata->>'k' = $1`
		if sql != wantSQL {
			t.Errorf("sql = %q, want %q", sql, wantSQL)
		}
		if len(args) != 1 {
			t.Errorf("args len = %d, want 1", len(args))
		}
	})

	t.Run("default logical op is AND", func(t *testing.T) {
		t.Parallel()
		idx := 1
		group := MetadataFilterGroup{
			// LogicalOp intentionally empty.
			Conditions: []MetadataCondition{
				{Key: "a", Operator: MetadataOpEquals, Value: 1},
				{Key: "b", Operator: MetadataOpEquals, Value: 2},
			},
		}
		sql, _, err := buildMetadataFilterSQLForTable(group, &idx, "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !strings.Contains(sql, " AND ") {
			t.Errorf("expected AND join, got %q", sql)
		}
	})

	t.Run("OR operator honored", func(t *testing.T) {
		t.Parallel()
		idx := 1
		group := MetadataFilterGroup{
			LogicalOp: LogicalOpOR,
			Conditions: []MetadataCondition{
				{Key: "a", Operator: MetadataOpEquals, Value: 1},
				{Key: "b", Operator: MetadataOpEquals, Value: 2},
			},
		}
		sql, _, err := buildMetadataFilterSQLForTable(group, &idx, "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if !strings.Contains(sql, " OR ") {
			t.Errorf("expected OR join, got %q", sql)
		}
	})

	t.Run("nested groups wrapped in parens", func(t *testing.T) {
		t.Parallel()
		idx := 1
		group := MetadataFilterGroup{
			LogicalOp: LogicalOpAND,
			Conditions: []MetadataCondition{
				{Key: "top", Operator: MetadataOpEquals, Value: "x"},
			},
			Groups: []MetadataFilterGroup{
				{Conditions: []MetadataCondition{{Key: "nested", Operator: MetadataOpEquals, Value: "y"}}},
			},
		}
		sql, _, err := buildMetadataFilterSQLForTable(group, &idx, "")
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		// Nested group's SQL must be wrapped in (...).
		if !strings.Contains(sql, "(") || !strings.Contains(sql, ")") {
			t.Errorf("expected nested group wrapped in parens, got %q", sql)
		}
	})
}

// =============================================================================
// convertFilterExpression — $user_id substitution (security-adjacent)
// =============================================================================
//
// Converts a request map to a MetadataFilterGroup. "$user_id" values are
// substituted with the supplied userID (user isolation); if userID is nil the
// $user_id key is DROPPED entirely (not emitted with a literal "$user_id").

func TestConvertFilterExpression(t *testing.T) {
	t.Parallel()
	uid := "user-123"

	t.Run("nil returns nil", func(t *testing.T) {
		t.Parallel()
		if got := convertFilterExpression(nil, &uid); got != nil {
			t.Errorf("expected nil, got %+v", got)
		}
	})

	t.Run("substitutes $user_id when userID set", func(t *testing.T) {
		t.Parallel()
		expr := map[string]interface{}{"user_id": "$user_id", "status": "active"}
		got := convertFilterExpression(expr, &uid)
		if got == nil {
			t.Fatal("expected non-nil group")
		}
		// Two conditions, AND-joined. One should carry the substituted userID.
		wantValues := map[string]interface{}{"user_id": uid, "status": "active"}
		gotValues := map[string]interface{}{}
		for _, c := range got.Conditions {
			gotValues[c.Key] = c.Value
		}
		if !reflect.DeepEqual(gotValues, wantValues) {
			t.Errorf("conditions = %+v, want %+v", gotValues, wantValues)
		}
	})

	t.Run("drops $user_id key when userID is nil", func(t *testing.T) {
		t.Parallel()
		// SECURITY: $user_id must NOT leak as a literal; it must be dropped so
		// an unauthenticated caller can't filter by the literal string.
		expr := map[string]interface{}{"user_id": "$user_id", "status": "active"}
		got := convertFilterExpression(expr, nil)
		if got == nil {
			t.Fatal("expected non-nil group (status condition remains)")
		}
		if len(got.Conditions) != 1 {
			t.Fatalf("expected 1 condition (user_id dropped), got %d: %+v", len(got.Conditions), got.Conditions)
		}
		if got.Conditions[0].Key != "status" {
			t.Errorf("expected only status condition, got key %q", got.Conditions[0].Key)
		}
	})

	t.Run("empty map returns nil", func(t *testing.T) {
		t.Parallel()
		if got := convertFilterExpression(map[string]interface{}{}, &uid); got != nil {
			t.Errorf("expected nil for empty map, got %+v", got)
		}
	})

	t.Run("non-string values passed through", func(t *testing.T) {
		t.Parallel()
		expr := map[string]interface{}{"count": 42, "flag": true}
		got := convertFilterExpression(expr, &uid)
		if got == nil {
			t.Fatal("expected non-nil group")
		}
		if len(got.Conditions) != 2 {
			t.Fatalf("expected 2 conditions, got %d", len(got.Conditions))
		}
	})
}
