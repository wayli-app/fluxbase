package api

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
	"github.com/stretchr/testify/assert"
)

// =============================================================================
// DateTimeScalar Tests
// =============================================================================

func TestDateTimeScalar_Name(t *testing.T) {
	assert.Equal(t, "DateTime", DateTimeScalar.Name())
}

func TestDateTimeScalar_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "time.Time value",
			input:    time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC),
			expected: "2024-06-15T10:30:00Z",
		},
		{
			name: "pointer to time.Time",
			input: func() *time.Time {
				t := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
				return &t
			}(),
			expected: "2024-01-01T00:00:00Z",
		},
		{
			name:     "nil pointer to time.Time",
			input:    (*time.Time)(nil),
			expected: nil,
		},
		{
			name:     "string passthrough",
			input:    "2024-06-15T10:30:00Z",
			expected: "2024-06-15T10:30:00Z",
		},
		{
			name:     "unsupported type returns nil",
			input:    12345,
			expected: nil,
		},
		{
			name:     "nil returns nil",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DateTimeScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDateTimeScalar_ParseValue(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		expectTime bool
		expectNil  bool
	}{
		{
			name:       "valid RFC3339 string",
			input:      "2024-06-15T10:30:00Z",
			expectTime: true,
		},
		{
			name:       "valid RFC3339 with timezone",
			input:      "2024-06-15T10:30:00+02:00",
			expectTime: true,
		},
		{
			name:      "invalid date string",
			input:     "not-a-date",
			expectNil: true,
		},
		{
			name:      "non-string input",
			input:     12345,
			expectNil: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DateTimeScalar.ParseValue(tt.input)
			if tt.expectNil {
				assert.Nil(t, result)
			} else if tt.expectTime {
				_, ok := result.(time.Time)
				assert.True(t, ok, "expected time.Time type")
			}
		})
	}
}

func TestDateTimeScalar_ParseLiteral(t *testing.T) {
	tests := []struct {
		name       string
		input      ast.Value
		expectTime bool
		expectNil  bool
	}{
		{
			name:       "valid string literal",
			input:      &ast.StringValue{Value: "2024-06-15T10:30:00Z"},
			expectTime: true,
		},
		{
			name:      "invalid string literal",
			input:     &ast.StringValue{Value: "not-a-date"},
			expectNil: true,
		},
		{
			name:      "non-string literal type",
			input:     &ast.IntValue{Value: "12345"},
			expectNil: true,
		},
		{
			name:      "nil literal",
			input:     nil,
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DateTimeScalar.ParseLiteral(tt.input)
			if tt.expectNil {
				assert.Nil(t, result)
			} else if tt.expectTime {
				_, ok := result.(time.Time)
				assert.True(t, ok, "expected time.Time type")
			}
		})
	}
}

// =============================================================================
// UUIDScalar Tests
// =============================================================================

func TestUUIDScalar_Name(t *testing.T) {
	assert.Equal(t, "UUID", UUIDScalar.Name())
}

func TestUUIDScalar_Serialize(t *testing.T) {
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "uuid.UUID value",
			input:    testUUID,
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "pointer to uuid.UUID",
			input:    &testUUID,
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "nil pointer to uuid.UUID",
			input:    (*uuid.UUID)(nil),
			expected: nil,
		},
		{
			name:     "string passthrough",
			input:    "550e8400-e29b-41d4-a716-446655440000",
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "16-byte slice (binary UUID)",
			input:    testUUID[:],
			expected: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name:     "short byte slice falls back to string",
			input:    []byte("short"),
			expected: "short",
		},
		{
			name:     "other type uses fmt.Sprintf",
			input:    12345,
			expected: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UUIDScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestUUIDScalar_ParseValue(t *testing.T) {
	tests := []struct {
		name       string
		input      interface{}
		expectUUID bool
		expectNil  bool
	}{
		{
			name:       "valid UUID string",
			input:      "550e8400-e29b-41d4-a716-446655440000",
			expectUUID: true,
		},
		{
			name:      "invalid UUID string",
			input:     "not-a-uuid",
			expectNil: true,
		},
		{
			name:      "non-string input",
			input:     12345,
			expectNil: true,
		},
		{
			name:      "empty string",
			input:     "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UUIDScalar.ParseValue(tt.input)
			if tt.expectNil {
				assert.Nil(t, result)
			} else if tt.expectUUID {
				_, ok := result.(uuid.UUID)
				assert.True(t, ok, "expected uuid.UUID type")
			}
		})
	}
}

func TestUUIDScalar_ParseLiteral(t *testing.T) {
	tests := []struct {
		name       string
		input      ast.Value
		expectUUID bool
		expectNil  bool
	}{
		{
			name:       "valid string literal",
			input:      &ast.StringValue{Value: "550e8400-e29b-41d4-a716-446655440000"},
			expectUUID: true,
		},
		{
			name:      "invalid string literal",
			input:     &ast.StringValue{Value: "not-a-uuid"},
			expectNil: true,
		},
		{
			name:      "non-string literal",
			input:     &ast.IntValue{Value: "12345"},
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := UUIDScalar.ParseLiteral(tt.input)
			if tt.expectNil {
				assert.Nil(t, result)
			} else if tt.expectUUID {
				_, ok := result.(uuid.UUID)
				assert.True(t, ok, "expected uuid.UUID type")
			}
		})
	}
}

// =============================================================================
// JSONScalar Tests
// =============================================================================

func TestJSONScalar_Name(t *testing.T) {
	assert.Equal(t, "JSON", JSONScalar.Name())
}

func TestJSONScalar_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "map[string]interface{}",
			input:    map[string]interface{}{"key": "value"},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "[]interface{}",
			input:    []interface{}{"a", "b", "c"},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "valid JSON string",
			input:    `{"key": "value"}`,
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "JSON array string",
			input:    `[1, 2, 3]`,
			expected: []interface{}{float64(1), float64(2), float64(3)},
		},
		{
			name:     "invalid JSON string passthrough",
			input:    "not-json",
			expected: "not-json",
		},
		{
			name:     "valid JSON bytes",
			input:    []byte(`{"key": "value"}`),
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "invalid JSON bytes passthrough",
			input:    []byte("not-json"),
			expected: "not-json",
		},
		{
			name:     "other type passthrough",
			input:    12345,
			expected: 12345,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestJSONScalar_ParseValue(t *testing.T) {
	t.Run("passthrough any value", func(t *testing.T) {
		testCases := []interface{}{
			map[string]interface{}{"key": "value"},
			[]interface{}{"a", "b"},
			"string",
			12345,
			nil,
		}
		for _, v := range testCases {
			assert.Equal(t, v, JSONScalar.ParseValue(v))
		}
	})
}

func TestJSONScalar_ParseLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    ast.Value
		expected interface{}
	}{
		{
			name:     "valid JSON string literal",
			input:    &ast.StringValue{Value: `{"key": "value"}`},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name:     "invalid JSON string literal",
			input:    &ast.StringValue{Value: "plain string"},
			expected: "plain string",
		},
		{
			name: "object value",
			input: &ast.ObjectValue{
				Fields: []*ast.ObjectField{
					{Name: &ast.Name{Value: "key"}, Value: &ast.StringValue{Value: "value"}},
				},
			},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name: "list value",
			input: &ast.ListValue{
				Values: []ast.Value{
					&ast.StringValue{Value: "a"},
					&ast.StringValue{Value: "b"},
				},
			},
			expected: []interface{}{"a", "b"},
		},
		{
			name:     "unsupported literal type",
			input:    &ast.BooleanValue{Value: true},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := JSONScalar.ParseLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// BigIntScalar Tests
// =============================================================================

func TestBigIntScalar_Name(t *testing.T) {
	assert.Equal(t, "BigInt", BigIntScalar.Name())
}

func TestBigIntScalar_Serialize(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "int64 value",
			input:    int64(9223372036854775807),
			expected: "9223372036854775807",
		},
		{
			name:     "int64 negative",
			input:    int64(-9223372036854775808),
			expected: "-9223372036854775808",
		},
		{
			name: "pointer to int64",
			input: func() *int64 {
				n := int64(12345)
				return &n
			}(),
			expected: "12345",
		},
		{
			name:     "nil pointer to int64",
			input:    (*int64)(nil),
			expected: nil,
		},
		{
			name:     "int value",
			input:    12345,
			expected: "12345",
		},
		{
			name:     "string passthrough",
			input:    "9223372036854775807",
			expected: "9223372036854775807",
		},
		{
			name:     "other type uses fmt.Sprintf",
			input:    true,
			expected: "true",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BigIntScalar.Serialize(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBigIntScalar_ParseValue(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
	}{
		{
			name:     "valid string",
			input:    "9223372036854775807",
			expected: int64(9223372036854775807),
		},
		{
			name:     "invalid string",
			input:    "not-a-number",
			expected: nil,
		},
		{
			name:     "int value",
			input:    12345,
			expected: int64(12345),
		},
		{
			name:     "float64 value",
			input:    float64(12345.67),
			expected: int64(12345),
		},
		{
			name:     "unsupported type",
			input:    true,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BigIntScalar.ParseValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBigIntScalar_ParseLiteral(t *testing.T) {
	tests := []struct {
		name     string
		input    ast.Value
		expected interface{}
	}{
		{
			name:     "valid string literal",
			input:    &ast.StringValue{Value: "9223372036854775807"},
			expected: int64(9223372036854775807),
		},
		{
			name:     "invalid string literal",
			input:    &ast.StringValue{Value: "not-a-number"},
			expected: nil,
		},
		{
			name:     "int literal",
			input:    &ast.IntValue{Value: "12345"},
			expected: int64(12345),
		},
		{
			name:     "unsupported literal type",
			input:    &ast.BooleanValue{Value: true},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := BigIntScalar.ParseLiteral(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// AST Helper Functions Tests
// =============================================================================

func TestParseObjectValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *ast.ObjectValue
		expected map[string]interface{}
	}{
		{
			name: "simple object",
			input: &ast.ObjectValue{
				Fields: []*ast.ObjectField{
					{Name: &ast.Name{Value: "name"}, Value: &ast.StringValue{Value: "test"}},
					{Name: &ast.Name{Value: "count"}, Value: &ast.IntValue{Value: "42"}},
				},
			},
			expected: map[string]interface{}{"name": "test", "count": int64(42)},
		},
		{
			name:     "empty object",
			input:    &ast.ObjectValue{Fields: []*ast.ObjectField{}},
			expected: map[string]interface{}{},
		},
		{
			name: "nested object",
			input: &ast.ObjectValue{
				Fields: []*ast.ObjectField{
					{
						Name: &ast.Name{Value: "nested"},
						Value: &ast.ObjectValue{
							Fields: []*ast.ObjectField{
								{Name: &ast.Name{Value: "inner"}, Value: &ast.StringValue{Value: "value"}},
							},
						},
					},
				},
			},
			expected: map[string]interface{}{
				"nested": map[string]interface{}{"inner": "value"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseObjectValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseListValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *ast.ListValue
		expected []interface{}
	}{
		{
			name: "string list",
			input: &ast.ListValue{
				Values: []ast.Value{
					&ast.StringValue{Value: "a"},
					&ast.StringValue{Value: "b"},
					&ast.StringValue{Value: "c"},
				},
			},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name: "mixed list",
			input: &ast.ListValue{
				Values: []ast.Value{
					&ast.StringValue{Value: "text"},
					&ast.IntValue{Value: "42"},
					&ast.BooleanValue{Value: true},
				},
			},
			expected: []interface{}{"text", int64(42), true},
		},
		{
			name:     "empty list",
			input:    &ast.ListValue{Values: []ast.Value{}},
			expected: []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseListValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseASTValue(t *testing.T) {
	tests := []struct {
		name     string
		input    ast.Value
		expected interface{}
	}{
		{
			name:     "string value",
			input:    &ast.StringValue{Value: "hello"},
			expected: "hello",
		},
		{
			name:     "int value",
			input:    &ast.IntValue{Value: "42"},
			expected: int64(42),
		},
		{
			name:     "float value",
			input:    &ast.FloatValue{Value: "3.14"},
			expected: float64(3.14),
		},
		{
			name:     "boolean true",
			input:    &ast.BooleanValue{Value: true},
			expected: true,
		},
		{
			name:     "boolean false",
			input:    &ast.BooleanValue{Value: false},
			expected: false,
		},
		{
			name: "object value",
			input: &ast.ObjectValue{
				Fields: []*ast.ObjectField{
					{Name: &ast.Name{Value: "key"}, Value: &ast.StringValue{Value: "value"}},
				},
			},
			expected: map[string]interface{}{"key": "value"},
		},
		{
			name: "list value",
			input: &ast.ListValue{
				Values: []ast.Value{&ast.IntValue{Value: "1"}, &ast.IntValue{Value: "2"}},
			},
			expected: []interface{}{int64(1), int64(2)},
		},
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseASTValue(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// =============================================================================
// PostgresTypeToGraphQL Tests
// =============================================================================

func TestPostgresTypeToGraphQL_StringTypes(t *testing.T) {
	stringTypes := []string{"text", "varchar", "character varying", "char", "character", "name", "citext"}
	for _, pgType := range stringTypes {
		t.Run(pgType+"_nullable", func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, graphql.String, result)
		})
		t.Run(pgType+"_non_nullable", func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, false)
			_, isNonNull := result.(*graphql.NonNull)
			assert.True(t, isNonNull)
		})
	}
}

func TestPostgresTypeToGraphQL_IntegerTypes(t *testing.T) {
	tests := []struct {
		pgType   string
		expected graphql.Output
	}{
		{"integer", graphql.Int},
		{"int", graphql.Int},
		{"int4", graphql.Int},
		{"smallint", graphql.Int},
		{"int2", graphql.Int},
		{"serial", graphql.Int},
		{"serial4", graphql.Int},
	}

	for _, tt := range tests {
		t.Run(tt.pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(tt.pgType, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPostgresTypeToGraphQL_BigIntTypes(t *testing.T) {
	bigIntTypes := []string{"bigint", "int8", "bigserial", "serial8"}
	for _, pgType := range bigIntTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, BigIntScalar, result)
		})
	}
}

func TestPostgresTypeToGraphQL_FloatTypes(t *testing.T) {
	floatTypes := []string{"real", "float4", "double precision", "float8", "numeric", "decimal", "money"}
	for _, pgType := range floatTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, graphql.Float, result)
		})
	}
}

func TestPostgresTypeToGraphQL_BooleanTypes(t *testing.T) {
	boolTypes := []string{"boolean", "bool"}
	for _, pgType := range boolTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, graphql.Boolean, result)
		})
	}
}

func TestPostgresTypeToGraphQL_UUIDType(t *testing.T) {
	t.Run("uuid", func(t *testing.T) {
		result := PostgresTypeToGraphQL("uuid", true)
		assert.Equal(t, UUIDScalar, result)
	})
}

func TestPostgresTypeToGraphQL_JSONTypes(t *testing.T) {
	jsonTypes := []string{"json", "jsonb"}
	for _, pgType := range jsonTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, JSONScalar, result)
		})
	}
}

func TestPostgresTypeToGraphQL_DateTimeTypes(t *testing.T) {
	dateTypes := []string{
		"timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz",
		"date", "time", "time without time zone", "time with time zone", "timetz", "interval",
	}
	for _, pgType := range dateTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, DateTimeScalar, result)
		})
	}
}

func TestPostgresTypeToGraphQL_ArrayTypes(t *testing.T) {
	t.Run("ARRAY keyword", func(t *testing.T) {
		result := PostgresTypeToGraphQL("ARRAY", true)
		assert.Equal(t, JSONScalar, result)
	})
	t.Run("array lowercase", func(t *testing.T) {
		result := PostgresTypeToGraphQL("array", true)
		assert.Equal(t, JSONScalar, result)
	})
	t.Run("array suffix notation", func(t *testing.T) {
		result := PostgresTypeToGraphQL("integer[]", true)
		assert.Equal(t, JSONScalar, result)
	})
	t.Run("text array suffix", func(t *testing.T) {
		result := PostgresTypeToGraphQL("text[]", true)
		assert.Equal(t, JSONScalar, result)
	})
}

func TestPostgresTypeToGraphQL_BinaryTypes(t *testing.T) {
	t.Run("bytea", func(t *testing.T) {
		result := PostgresTypeToGraphQL("bytea", true)
		assert.Equal(t, graphql.String, result)
	})
}

func TestPostgresTypeToGraphQL_NetworkTypes(t *testing.T) {
	networkTypes := []string{"inet", "cidr", "macaddr", "macaddr8"}
	for _, pgType := range networkTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, graphql.String, result)
		})
	}
}

func TestPostgresTypeToGraphQL_GeometricTypes(t *testing.T) {
	geoTypes := []string{"point", "line", "lseg", "box", "path", "polygon", "circle"}
	for _, pgType := range geoTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, JSONScalar, result)
		})
	}
}

func TestPostgresTypeToGraphQL_RangeTypes(t *testing.T) {
	rangeTypes := []string{"int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange"}
	for _, pgType := range rangeTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, JSONScalar, result)
		})
	}
}

func TestPostgresTypeToGraphQL_FullTextSearchTypes(t *testing.T) {
	ftsTypes := []string{"tsvector", "tsquery"}
	for _, pgType := range ftsTypes {
		t.Run(pgType, func(t *testing.T) {
			result := PostgresTypeToGraphQL(pgType, true)
			assert.Equal(t, graphql.String, result)
		})
	}
}

func TestPostgresTypeToGraphQL_VectorType(t *testing.T) {
	t.Run("vector (pgvector)", func(t *testing.T) {
		result := PostgresTypeToGraphQL("vector", true)
		assert.Equal(t, JSONScalar, result)
	})
}

func TestPostgresTypeToGraphQL_UnknownTypes(t *testing.T) {
	tests := []struct {
		name     string
		pgType   string
		expected graphql.Output
	}{
		{
			name:     "enum type defaults to string",
			pgType:   "user_status",
			expected: graphql.String,
		},
		{
			name:     "custom type defaults to string",
			pgType:   "custom_domain",
			expected: graphql.String,
		},
		{
			name:     "unknown type defaults to string",
			pgType:   "xyz_unknown",
			expected: graphql.String,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PostgresTypeToGraphQL(tt.pgType, true)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPostgresTypeToGraphQL_NonNullWrapping(t *testing.T) {
	t.Run("nullable returns base type", func(t *testing.T) {
		result := PostgresTypeToGraphQL("text", true)
		assert.Equal(t, graphql.String, result)
	})
	t.Run("non-nullable wraps in NonNull", func(t *testing.T) {
		result := PostgresTypeToGraphQL("text", false)
		nonNull, ok := result.(*graphql.NonNull)
		assert.True(t, ok)
		assert.Equal(t, graphql.String, nonNull.OfType)
	})
}

// =============================================================================
// GetFilterOperatorsForType Tests
// =============================================================================

func TestGetFilterOperatorsForType_StringTypes(t *testing.T) {
	stringTypes := []string{"text", "varchar", "character varying", "char", "character", "name", "citext"}
	for _, pgType := range stringTypes {
		t.Run(pgType, func(t *testing.T) {
			ops := GetFilterOperatorsForType(pgType)
			// Base operators
			assert.True(t, ops.Eq)
			assert.True(t, ops.Neq)
			assert.True(t, ops.IsNull)
			// String-specific operators
			assert.True(t, ops.Like)
			assert.True(t, ops.ILike)
			assert.True(t, ops.In)
			assert.True(t, ops.Gt)
			assert.True(t, ops.Gte)
			assert.True(t, ops.Lt)
			assert.True(t, ops.Lte)
			// Not for strings
			assert.False(t, ops.Contains)
			assert.False(t, ops.ContainedBy)
		})
	}
}

func TestGetFilterOperatorsForType_NumericTypes(t *testing.T) {
	numericTypes := []string{
		"integer", "int", "int4", "smallint", "int2", "bigint", "int8",
		"real", "float4", "double precision", "float8", "numeric", "decimal", "money",
	}
	for _, pgType := range numericTypes {
		t.Run(pgType, func(t *testing.T) {
			ops := GetFilterOperatorsForType(pgType)
			// Base operators
			assert.True(t, ops.Eq)
			assert.True(t, ops.Neq)
			assert.True(t, ops.IsNull)
			// Numeric operators
			assert.True(t, ops.Gt)
			assert.True(t, ops.Gte)
			assert.True(t, ops.Lt)
			assert.True(t, ops.Lte)
			assert.True(t, ops.In)
			// Not for numerics
			assert.False(t, ops.Like)
			assert.False(t, ops.ILike)
		})
	}
}

func TestGetFilterOperatorsForType_BooleanTypes(t *testing.T) {
	boolTypes := []string{"boolean", "bool"}
	for _, pgType := range boolTypes {
		t.Run(pgType, func(t *testing.T) {
			ops := GetFilterOperatorsForType(pgType)
			// Base operators only
			assert.True(t, ops.Eq)
			assert.True(t, ops.Neq)
			assert.True(t, ops.IsNull)
			// Not for booleans
			assert.False(t, ops.Gt)
			assert.False(t, ops.Gte)
			assert.False(t, ops.Lt)
			assert.False(t, ops.Lte)
			assert.False(t, ops.Like)
			assert.False(t, ops.ILike)
			assert.False(t, ops.In)
		})
	}
}

func TestGetFilterOperatorsForType_UUIDType(t *testing.T) {
	t.Run("uuid", func(t *testing.T) {
		ops := GetFilterOperatorsForType("uuid")
		assert.True(t, ops.Eq)
		assert.True(t, ops.Neq)
		assert.True(t, ops.IsNull)
		assert.True(t, ops.In)
		// Not for UUID
		assert.False(t, ops.Gt)
		assert.False(t, ops.Like)
	})
}

func TestGetFilterOperatorsForType_JSONTypes(t *testing.T) {
	jsonTypes := []string{"json", "jsonb"}
	for _, pgType := range jsonTypes {
		t.Run(pgType, func(t *testing.T) {
			ops := GetFilterOperatorsForType(pgType)
			assert.True(t, ops.Eq)
			assert.True(t, ops.Neq)
			assert.True(t, ops.IsNull)
			// JSON-specific operators
			assert.True(t, ops.Contains)
			assert.True(t, ops.ContainedBy)
			// Not for JSON
			assert.False(t, ops.Gt)
			assert.False(t, ops.Like)
			assert.False(t, ops.In)
		})
	}
}

func TestGetFilterOperatorsForType_DateTimeTypes(t *testing.T) {
	dateTypes := []string{
		"timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz",
		"date", "time", "time without time zone", "time with time zone", "timetz",
	}
	for _, pgType := range dateTypes {
		t.Run(pgType, func(t *testing.T) {
			ops := GetFilterOperatorsForType(pgType)
			assert.True(t, ops.Eq)
			assert.True(t, ops.Neq)
			assert.True(t, ops.IsNull)
			assert.True(t, ops.Gt)
			assert.True(t, ops.Gte)
			assert.True(t, ops.Lt)
			assert.True(t, ops.Lte)
			assert.True(t, ops.In)
		})
	}
}

func TestGetFilterOperatorsForType_UnknownType(t *testing.T) {
	t.Run("unknown type gets base operators only", func(t *testing.T) {
		ops := GetFilterOperatorsForType("custom_enum_type")
		// Only base operators
		assert.True(t, ops.Eq)
		assert.True(t, ops.Neq)
		assert.True(t, ops.IsNull)
		// No additional operators
		assert.False(t, ops.Gt)
		assert.False(t, ops.Like)
		assert.False(t, ops.In)
		assert.False(t, ops.Contains)
	})
}

// =============================================================================
// GraphQLFilterOperators Struct Tests
// =============================================================================

func TestGraphQLFilterOperators_Struct(t *testing.T) {
	t.Run("zero value is all false", func(t *testing.T) {
		var ops GraphQLFilterOperators
		assert.False(t, ops.Eq)
		assert.False(t, ops.Neq)
		assert.False(t, ops.Gt)
		assert.False(t, ops.Gte)
		assert.False(t, ops.Lt)
		assert.False(t, ops.Lte)
		assert.False(t, ops.Like)
		assert.False(t, ops.ILike)
		assert.False(t, ops.In)
		assert.False(t, ops.IsNull)
		assert.False(t, ops.Contains)
		assert.False(t, ops.ContainedBy)
	})

	t.Run("can set all fields", func(t *testing.T) {
		ops := GraphQLFilterOperators{
			Eq:          true,
			Neq:         true,
			Gt:          true,
			Gte:         true,
			Lt:          true,
			Lte:         true,
			Like:        true,
			ILike:       true,
			In:          true,
			IsNull:      true,
			Contains:    true,
			ContainedBy: true,
		}
		assert.True(t, ops.Eq)
		assert.True(t, ops.Neq)
		assert.True(t, ops.Gt)
		assert.True(t, ops.Gte)
		assert.True(t, ops.Lt)
		assert.True(t, ops.Lte)
		assert.True(t, ops.Like)
		assert.True(t, ops.ILike)
		assert.True(t, ops.In)
		assert.True(t, ops.IsNull)
		assert.True(t, ops.Contains)
		assert.True(t, ops.ContainedBy)
	})
}

// =============================================================================
// Benchmarks
// =============================================================================

func BenchmarkDateTimeScalar_Serialize(b *testing.B) {
	testTime := time.Date(2024, 6, 15, 10, 30, 0, 0, time.UTC)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		DateTimeScalar.Serialize(testTime)
	}
}

func BenchmarkUUIDScalar_Serialize(b *testing.B) {
	testUUID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		UUIDScalar.Serialize(testUUID)
	}
}

func BenchmarkJSONScalar_Serialize(b *testing.B) {
	testJSON := map[string]interface{}{"key": "value", "nested": map[string]interface{}{"a": 1}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		JSONScalar.Serialize(testJSON)
	}
}

func BenchmarkBigIntScalar_Serialize(b *testing.B) {
	testValue := int64(9223372036854775807)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		BigIntScalar.Serialize(testValue)
	}
}

func BenchmarkPostgresTypeToGraphQL(b *testing.B) {
	types := []string{"text", "integer", "boolean", "uuid", "jsonb", "timestamp", "vector"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range types {
			PostgresTypeToGraphQL(t, true)
		}
	}
}

func BenchmarkGetFilterOperatorsForType(b *testing.B) {
	types := []string{"text", "integer", "boolean", "uuid", "jsonb", "timestamp"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, t := range types {
			GetFilterOperatorsForType(t)
		}
	}
}

func BenchmarkParseASTValue(b *testing.B) {
	values := []ast.Value{
		&ast.StringValue{Value: "test"},
		&ast.IntValue{Value: "42"},
		&ast.BooleanValue{Value: true},
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, v := range values {
			parseASTValue(v)
		}
	}
}
