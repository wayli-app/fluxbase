package realtime

import (
	"testing"
)

func TestParseFilter(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      *Filter
		wantError bool
	}{
		{
			name:  "empty filter",
			input: "",
			want:  nil,
		},
		{
			name:  "equality filter",
			input: "user_id=eq.abc123",
			want: &Filter{
				Column:   "user_id",
				Operator: "eq",
				Value:    "abc123",
			},
		},
		{
			name:  "numeric comparison",
			input: "priority=gt.5",
			want: &Filter{
				Column:   "priority",
				Operator: "gt",
				Value:    "5",
			},
		},
		{
			name:  "IN operator",
			input: "status=in.(queued,running)",
			want: &Filter{
				Column:   "status",
				Operator: "in",
				Value:    "(queued,running)",
			},
		},
		{
			name:  "IS NULL",
			input: "deleted_at=is.null",
			want: &Filter{
				Column:   "deleted_at",
				Operator: "is",
				Value:    "null",
			},
		},
		{
			name:  "LIKE pattern",
			input: "email=like.*@gmail.com",
			want: &Filter{
				Column:   "email",
				Operator: "like",
				Value:    "*@gmail.com",
			},
		},
		{
			name:      "invalid format - no operator",
			input:     "user_id=abc123",
			wantError: true,
		},
		{
			name:      "invalid format - no value",
			input:     "user_id=eq.",
			wantError: true,
		},
		{
			name:      "invalid operator",
			input:     "user_id=invalid.abc123",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilter(tt.input)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseFilter() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseFilter() unexpected error: %v", err)
				return
			}

			if tt.want == nil && got != nil {
				t.Errorf("ParseFilter() = %v, want nil", got)
				return
			}

			if tt.want != nil {
				if got.Column != tt.want.Column {
					t.Errorf("ParseFilter().Column = %v, want %v", got.Column, tt.want.Column)
				}
				if got.Operator != tt.want.Operator {
					t.Errorf("ParseFilter().Operator = %v, want %v", got.Operator, tt.want.Operator)
				}
				if got.Value != tt.want.Value {
					t.Errorf("ParseFilter().Value = %v, want %v", got.Value, tt.want.Value)
				}
			}
		})
	}
}

func TestFilter_Matches(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		record map[string]interface{}
		want   bool
	}{
		{
			name:   "nil filter matches all",
			filter: nil,
			record: map[string]interface{}{"user_id": "abc123"},
			want:   true,
		},
		{
			name: "equality match",
			filter: &Filter{
				Column:   "user_id",
				Operator: "eq",
				Value:    "abc123",
			},
			record: map[string]interface{}{"user_id": "abc123"},
			want:   true,
		},
		{
			name: "equality no match",
			filter: &Filter{
				Column:   "user_id",
				Operator: "eq",
				Value:    "abc123",
			},
			record: map[string]interface{}{"user_id": "xyz789"},
			want:   false,
		},
		{
			name: "not equal match",
			filter: &Filter{
				Column:   "status",
				Operator: "neq",
				Value:    "completed",
			},
			record: map[string]interface{}{"status": "queued"},
			want:   true,
		},
		{
			name: "not equal no match",
			filter: &Filter{
				Column:   "status",
				Operator: "neq",
				Value:    "completed",
			},
			record: map[string]interface{}{"status": "completed"},
			want:   false,
		},
		{
			name: "greater than match - int",
			filter: &Filter{
				Column:   "priority",
				Operator: "gt",
				Value:    "5",
			},
			record: map[string]interface{}{"priority": 8},
			want:   true,
		},
		{
			name: "greater than no match - int",
			filter: &Filter{
				Column:   "priority",
				Operator: "gt",
				Value:    "5",
			},
			record: map[string]interface{}{"priority": 3},
			want:   false,
		},
		{
			name: "greater than or equal match",
			filter: &Filter{
				Column:   "priority",
				Operator: "gte",
				Value:    "5",
			},
			record: map[string]interface{}{"priority": 5},
			want:   true,
		},
		{
			name: "less than match - float",
			filter: &Filter{
				Column:   "progress",
				Operator: "lt",
				Value:    "100",
			},
			record: map[string]interface{}{"progress": 75.5},
			want:   true,
		},
		{
			name: "less than or equal match",
			filter: &Filter{
				Column:   "progress",
				Operator: "lte",
				Value:    "50",
			},
			record: map[string]interface{}{"progress": 50},
			want:   true,
		},
		{
			name: "IS NULL match",
			filter: &Filter{
				Column:   "deleted_at",
				Operator: "is",
				Value:    "null",
			},
			record: map[string]interface{}{"deleted_at": nil},
			want:   true,
		},
		{
			name: "IS NULL no match",
			filter: &Filter{
				Column:   "deleted_at",
				Operator: "is",
				Value:    "null",
			},
			record: map[string]interface{}{"deleted_at": "2024-01-01"},
			want:   false,
		},
		{
			name: "IS TRUE match",
			filter: &Filter{
				Column:   "active",
				Operator: "is",
				Value:    "true",
			},
			record: map[string]interface{}{"active": true},
			want:   true,
		},
		{
			name: "IS FALSE match",
			filter: &Filter{
				Column:   "active",
				Operator: "is",
				Value:    "false",
			},
			record: map[string]interface{}{"active": false},
			want:   true,
		},
		{
			name: "IN operator match - first value",
			filter: &Filter{
				Column:   "status",
				Operator: "in",
				Value:    "(queued,running,completed)",
			},
			record: map[string]interface{}{"status": "queued"},
			want:   true,
		},
		{
			name: "IN operator match - middle value",
			filter: &Filter{
				Column:   "status",
				Operator: "in",
				Value:    "(queued,running,completed)",
			},
			record: map[string]interface{}{"status": "running"},
			want:   true,
		},
		{
			name: "IN operator no match",
			filter: &Filter{
				Column:   "status",
				Operator: "in",
				Value:    "(queued,running)",
			},
			record: map[string]interface{}{"status": "completed"},
			want:   false,
		},
		{
			name: "LIKE pattern match - suffix",
			filter: &Filter{
				Column:   "email",
				Operator: "like",
				Value:    "*@gmail.com",
			},
			record: map[string]interface{}{"email": "user@gmail.com"},
			want:   true,
		},
		{
			name: "LIKE pattern match - prefix",
			filter: &Filter{
				Column:   "name",
				Operator: "like",
				Value:    "John*",
			},
			record: map[string]interface{}{"name": "John Doe"},
			want:   true,
		},
		{
			name: "LIKE pattern match - contains",
			filter: &Filter{
				Column:   "description",
				Operator: "like",
				Value:    "*important*",
			},
			record: map[string]interface{}{"description": "This is important text"},
			want:   true,
		},
		{
			name: "LIKE pattern no match",
			filter: &Filter{
				Column:   "email",
				Operator: "like",
				Value:    "*@gmail.com",
			},
			record: map[string]interface{}{"email": "user@yahoo.com"},
			want:   false,
		},
		{
			name: "ILIKE pattern match - case insensitive",
			filter: &Filter{
				Column:   "name",
				Operator: "ilike",
				Value:    "*JOHN*",
			},
			record: map[string]interface{}{"name": "john doe"},
			want:   true,
		},
		{
			name: "missing column",
			filter: &Filter{
				Column:   "user_id",
				Operator: "eq",
				Value:    "abc123",
			},
			record: map[string]interface{}{"status": "queued"},
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Matches(tt.record)
			if got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompareNumeric(t *testing.T) {
	tests := []struct {
		name        string
		recordValue interface{}
		filterValue string
		want        int
	}{
		{
			name:        "int equal",
			recordValue: 5,
			filterValue: "5",
			want:        0,
		},
		{
			name:        "int less than",
			recordValue: 3,
			filterValue: "5",
			want:        -1,
		},
		{
			name:        "int greater than",
			recordValue: 8,
			filterValue: "5",
			want:        1,
		},
		{
			name:        "float equal",
			recordValue: 5.5,
			filterValue: "5.5",
			want:        0,
		},
		{
			name:        "float less than",
			recordValue: 3.2,
			filterValue: "5.5",
			want:        -1,
		},
		{
			name:        "float greater than",
			recordValue: 7.8,
			filterValue: "5.5",
			want:        1,
		},
		{
			name:        "int64 equal",
			recordValue: int64(100),
			filterValue: "100",
			want:        0,
		},
		{
			name:        "string numeric equal",
			recordValue: "42",
			filterValue: "42",
			want:        0,
		},
		{
			name:        "string numeric less than",
			recordValue: "25",
			filterValue: "42",
			want:        -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compareNumeric(tt.recordValue, tt.filterValue)
			if got != tt.want {
				t.Errorf("compareNumeric(%v, %v) = %v, want %v",
					tt.recordValue, tt.filterValue, got, tt.want)
			}
		})
	}
}

func BenchmarkFilterMatches(b *testing.B) {
	filter := &Filter{
		Column:   "user_id",
		Operator: "eq",
		Value:    "abc123",
	}
	record := map[string]interface{}{
		"user_id":    "abc123",
		"status":     "queued",
		"priority":   5,
		"created_at": "2024-01-01",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(record)
	}
}

func BenchmarkFilterMatchesNumeric(b *testing.B) {
	filter := &Filter{
		Column:   "priority",
		Operator: "gt",
		Value:    "5",
	}
	record := map[string]interface{}{
		"user_id":  "abc123",
		"priority": 8,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(record)
	}
}

func BenchmarkFilterMatchesIn(b *testing.B) {
	filter := &Filter{
		Column:   "status",
		Operator: "in",
		Value:    "(queued,running,completed,failed)",
	}
	record := map[string]interface{}{
		"status": "running",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		filter.Matches(record)
	}
}

func BenchmarkParseFilter(b *testing.B) {
	filterStr := "user_id=eq.abc123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = ParseFilter(filterStr)
	}
}
