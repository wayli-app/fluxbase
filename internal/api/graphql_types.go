package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/ast"
)

// Custom GraphQL scalar types for PostgreSQL data types

// DateTime scalar for timestamp/timestamptz columns
var DateTimeScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "DateTime",
	Description: "DateTime scalar type represents a date and time in RFC3339 format",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case time.Time:
			return v.Format(time.RFC3339)
		case *time.Time:
			if v == nil {
				return nil
			}
			return v.Format(time.RFC3339)
		case string:
			return v
		default:
			return nil
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			t, err := time.Parse(time.RFC3339, v)
			if err != nil {
				return nil
			}
			return t
		default:
			return nil
		}
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch v := valueAST.(type) {
		case *ast.StringValue:
			t, err := time.Parse(time.RFC3339, v.Value)
			if err != nil {
				return nil
			}
			return t
		default:
			return nil
		}
	},
})

// UUID scalar for uuid columns
var UUIDScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "UUID",
	Description: "UUID scalar type represents a universally unique identifier",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case uuid.UUID:
			return v.String()
		case *uuid.UUID:
			if v == nil {
				return nil
			}
			return v.String()
		case string:
			return v
		case []byte:
			if len(v) == 16 {
				u, err := uuid.FromBytes(v)
				if err == nil {
					return u.String()
				}
			}
			return string(v)
		default:
			return fmt.Sprintf("%v", v)
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			u, err := uuid.Parse(v)
			if err != nil {
				return nil
			}
			return u
		default:
			return nil
		}
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch v := valueAST.(type) {
		case *ast.StringValue:
			u, err := uuid.Parse(v.Value)
			if err != nil {
				return nil
			}
			return u
		default:
			return nil
		}
	},
})

// JSON scalar for jsonb/json columns
var JSONScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "JSON",
	Description: "JSON scalar type represents arbitrary JSON data",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case map[string]interface{}:
			return v
		case []interface{}:
			return v
		case string:
			var result interface{}
			if err := json.Unmarshal([]byte(v), &result); err != nil {
				return v
			}
			return result
		case []byte:
			var result interface{}
			if err := json.Unmarshal(v, &result); err != nil {
				return string(v)
			}
			return result
		default:
			return v
		}
	},
	ParseValue: func(value interface{}) interface{} {
		return value
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch v := valueAST.(type) {
		case *ast.StringValue:
			var result interface{}
			if err := json.Unmarshal([]byte(v.Value), &result); err != nil {
				return v.Value
			}
			return result
		case *ast.ObjectValue:
			return parseObjectValue(v)
		case *ast.ListValue:
			return parseListValue(v)
		default:
			return nil
		}
	},
})

// BigInt scalar for bigint columns (represented as string to avoid JS precision issues)
var BigIntScalar = graphql.NewScalar(graphql.ScalarConfig{
	Name:        "BigInt",
	Description: "BigInt scalar type represents large integers as strings",
	Serialize: func(value interface{}) interface{} {
		switch v := value.(type) {
		case int64:
			return fmt.Sprintf("%d", v)
		case *int64:
			if v == nil {
				return nil
			}
			return fmt.Sprintf("%d", *v)
		case int:
			return fmt.Sprintf("%d", v)
		case string:
			return v
		default:
			return fmt.Sprintf("%v", v)
		}
	},
	ParseValue: func(value interface{}) interface{} {
		switch v := value.(type) {
		case string:
			var n int64
			if _, err := fmt.Sscanf(v, "%d", &n); err != nil {
				return nil
			}
			return n
		case int:
			return int64(v)
		case float64:
			return int64(v)
		default:
			return nil
		}
	},
	ParseLiteral: func(valueAST ast.Value) interface{} {
		switch v := valueAST.(type) {
		case *ast.StringValue:
			var n int64
			if _, err := fmt.Sscanf(v.Value, "%d", &n); err != nil {
				return nil
			}
			return n
		case *ast.IntValue:
			var n int64
			if _, err := fmt.Sscanf(v.Value, "%d", &n); err != nil {
				return nil
			}
			return n
		default:
			return nil
		}
	},
})

// Helper functions for parsing AST values
func parseObjectValue(v *ast.ObjectValue) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range v.Fields {
		result[field.Name.Value] = parseASTValue(field.Value)
	}
	return result
}

func parseListValue(v *ast.ListValue) []interface{} {
	result := make([]interface{}, len(v.Values))
	for i, val := range v.Values {
		result[i] = parseASTValue(val)
	}
	return result
}

func parseASTValue(v ast.Value) interface{} {
	switch val := v.(type) {
	case *ast.StringValue:
		return val.Value
	case *ast.IntValue:
		var n int64
		fmt.Sscanf(val.Value, "%d", &n)
		return n
	case *ast.FloatValue:
		var f float64
		fmt.Sscanf(val.Value, "%f", &f)
		return f
	case *ast.BooleanValue:
		return val.Value
	case *ast.ObjectValue:
		return parseObjectValue(val)
	case *ast.ListValue:
		return parseListValue(val)
	case *ast.NullValue:
		return nil
	default:
		return nil
	}
}

// PostgresTypeToGraphQL maps PostgreSQL data types to GraphQL types
func PostgresTypeToGraphQL(pgType string, isNullable bool) graphql.Output {
	var baseType graphql.Output

	switch pgType {
	// String types
	case "text", "varchar", "character varying", "char", "character", "name", "citext":
		baseType = graphql.String

	// Integer types
	case "integer", "int", "int4", "smallint", "int2":
		baseType = graphql.Int
	case "bigint", "int8":
		baseType = BigIntScalar
	case "serial", "serial4":
		baseType = graphql.Int
	case "bigserial", "serial8":
		baseType = BigIntScalar

	// Floating point types
	case "real", "float4", "double precision", "float8", "numeric", "decimal", "money":
		baseType = graphql.Float

	// Boolean type
	case "boolean", "bool":
		baseType = graphql.Boolean

	// UUID type
	case "uuid":
		baseType = UUIDScalar

	// JSON types
	case "json", "jsonb":
		baseType = JSONScalar

	// Date/time types
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz",
		"date", "time", "time without time zone", "time with time zone", "timetz", "interval":
		baseType = DateTimeScalar

	// Array types (return as JSON for simplicity)
	case "ARRAY", "array":
		baseType = JSONScalar

	// Binary types
	case "bytea":
		baseType = graphql.String

	// Network types
	case "inet", "cidr", "macaddr", "macaddr8":
		baseType = graphql.String

	// Geometric types
	case "point", "line", "lseg", "box", "path", "polygon", "circle":
		baseType = JSONScalar

	// Range types
	case "int4range", "int8range", "numrange", "tsrange", "tstzrange", "daterange":
		baseType = JSONScalar

	// Full text search types
	case "tsvector", "tsquery":
		baseType = graphql.String

	// Vector types (pgvector)
	case "vector":
		baseType = JSONScalar

	// Enum types and other custom types - treat as string
	default:
		// Check if it's an array type (ends with [])
		if len(pgType) > 2 && pgType[len(pgType)-2:] == "[]" {
			baseType = JSONScalar
		} else {
			// Default to string for unknown types (including enums)
			baseType = graphql.String
		}
	}

	// Wrap in NonNull if not nullable
	if !isNullable {
		return graphql.NewNonNull(baseType)
	}
	return baseType
}

// GraphQLFilterOperators defines available filter operators for each type
type GraphQLFilterOperators struct {
	Eq          bool // equals
	Neq         bool // not equals
	Gt          bool // greater than
	Gte         bool // greater than or equal
	Lt          bool // less than
	Lte         bool // less than or equal
	Like        bool // LIKE pattern match
	ILike       bool // case-insensitive LIKE
	In          bool // in array
	IsNull      bool // is null / is not null
	Contains    bool // JSON contains (@>)
	ContainedBy bool // JSON contained by (<@)
}

// GetFilterOperatorsForType returns the available filter operators for a PostgreSQL type
func GetFilterOperatorsForType(pgType string) GraphQLFilterOperators {
	ops := GraphQLFilterOperators{
		Eq:     true,
		Neq:    true,
		IsNull: true,
	}

	switch pgType {
	case "text", "varchar", "character varying", "char", "character", "name", "citext":
		ops.Like = true
		ops.ILike = true
		ops.In = true
		ops.Gt = true
		ops.Gte = true
		ops.Lt = true
		ops.Lte = true

	case "integer", "int", "int4", "smallint", "int2", "bigint", "int8",
		"real", "float4", "double precision", "float8", "numeric", "decimal", "money":
		ops.Gt = true
		ops.Gte = true
		ops.Lt = true
		ops.Lte = true
		ops.In = true

	case "boolean", "bool":
		// Only eq/neq/isNull for booleans

	case "uuid":
		ops.In = true

	case "json", "jsonb":
		ops.Contains = true
		ops.ContainedBy = true

	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz",
		"date", "time", "time without time zone", "time with time zone", "timetz":
		ops.Gt = true
		ops.Gte = true
		ops.Lt = true
		ops.Lte = true
		ops.In = true
	}

	return ops
}
