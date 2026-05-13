package api

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/config"
	"github.com/nimbleflux/fluxbase/internal/query"
)

// isValidIdentifier checks if a string is a valid SQL identifier
func isValidIdentifier(s string) bool {
	return validIdentifierRegex.MatchString(s)
}

// quoteIdentifier safely quotes an SQL identifier to prevent injection
// Returns empty string if the identifier is invalid
func quoteIdentifier(s string) string {
	if !isValidIdentifier(s) {
		return ""
	}
	// Escape any embedded double quotes and wrap in double quotes
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// QueryParams represents parsed query parameters for REST API
type QueryParams struct {
	Select         []string           // Fields to select
	Filters        []Filter           // WHERE conditions
	Order          []OrderBy          // ORDER BY clauses
	Limit          *int               // LIMIT clause
	Offset         *int               // OFFSET clause
	Cursor         *string            // Base64-encoded cursor for keyset pagination
	CursorColumn   *string            // Column to use for cursor (default: primary key)
	Embedded       []EmbeddedRelation // Relations to embed
	Count          CountType          // Count preference
	Aggregations   []Aggregation      // Aggregation functions
	GroupBy        []string           // GROUP BY columns
	TruncateLength *int               // Truncate text columns to this length (for table browsing)
	orGroupCounter int                // Counter for assigning OR group IDs
}

// Filter is an alias for query.Filter for backward compatibility
type Filter = query.Filter

// FilterOperator is an alias for query.FilterOperator for backward compatibility
type FilterOperator = query.FilterOperator

// Re-export filter operator constants for backward compatibility
const (
	OpEqual          = query.OpEqual
	OpNotEqual       = query.OpNotEqual
	OpGreaterThan    = query.OpGreaterThan
	OpGreaterOrEqual = query.OpGreaterOrEqual
	OpLessThan       = query.OpLessThan
	OpLessOrEqual    = query.OpLessOrEqual
	OpLike           = query.OpLike
	OpILike          = query.OpILike
	OpIn             = query.OpIn
	OpNotIn          = query.OpNotIn
	OpIs             = query.OpIs
	OpIsNot          = query.OpIsNot
	OpContains       = query.OpContains
	OpContained      = query.OpContained
	OpContainedBy    = query.OpContainedBy
	OpOverlap        = query.OpOverlap
	OpOverlaps       = query.OpOverlaps
	OpTextSearch     = query.OpTextSearch
	OpPhraseSearch   = query.OpPhraseSearch
	OpWebSearch      = query.OpWebSearch
	OpNot            = query.OpNot
	OpAdjacent       = query.OpAdjacent
	OpStrictlyLeft   = query.OpStrictlyLeft
	OpStrictlyRight  = query.OpStrictlyRight
	OpNotExtendRight = query.OpNotExtendRight
	OpNotExtendLeft  = query.OpNotExtendLeft

	// PostGIS spatial operators
	OpSTIntersects = query.OpSTIntersects
	OpSTContains   = query.OpSTContains
	OpSTWithin     = query.OpSTWithin
	OpSTDWithin    = query.OpSTDWithin
	OpSTDistance   = query.OpSTDistance
	OpSTTouches    = query.OpSTTouches
	OpSTCrosses    = query.OpSTCrosses
	OpSTOverlaps   = query.OpSTOverlaps

	// pgvector similarity operators
	OpVectorL2     = query.OpVectorL2
	OpVectorCosine = query.OpVectorCosine
	OpVectorIP     = query.OpVectorIP
)

// OrderBy is an alias for query.OrderBy for backward compatibility
type OrderBy = query.OrderBy

// EmbeddedRelation represents a relation to embed
type EmbeddedRelation struct {
	Name    string   // Relation name
	Select  []string // Fields to select from relation
	Filters []Filter // Filters for the relation
}

// CountType represents row count preferences
type CountType string

const (
	CountNone      CountType = "none"
	CountExact     CountType = "exact"
	CountPlanned   CountType = "planned"
	CountEstimated CountType = "estimated"
)

// Aggregation represents an aggregation function
type Aggregation struct {
	Function AggregateFunction
	Column   string
	Alias    string // Optional alias for the result
}

// AggregateFunction represents aggregation functions
type AggregateFunction string

const (
	AggCount    AggregateFunction = "count"
	AggSum      AggregateFunction = "sum"
	AggAvg      AggregateFunction = "avg"
	AggMin      AggregateFunction = "min"
	AggMax      AggregateFunction = "max"
	AggCountAll AggregateFunction = "count(*)"
)

// QueryParser parses PostgREST-compatible query parameters
type QueryParser struct {
	config *config.Config
}

// ParseOptions configures query parsing behavior
type ParseOptions struct {
	// BypassMaxTotalResults skips the max_total_results enforcement.
	// Use for admin/dashboard requests that should have unlimited access.
	BypassMaxTotalResults bool
}

// NewQueryParser creates a new query parser
func NewQueryParser(cfg *config.Config) *QueryParser {
	return &QueryParser{
		config: cfg,
	}
}

// Parse parses URL query parameters into QueryParams with default options
func (qp *QueryParser) Parse(values url.Values) (*QueryParams, error) {
	return qp.ParseWithOptions(values, ParseOptions{})
}

// ParseWithOptions parses URL query parameters into QueryParams with custom options
func (qp *QueryParser) ParseWithOptions(values url.Values, opts ParseOptions) (*QueryParams, error) {
	params := &QueryParams{
		Filters: []Filter{},
		Order:   []OrderBy{},
	}

	// Parse each parameter type
	for key, vals := range values {
		switch key {
		case "select":
			if err := qp.parseSelect(vals[0], params); err != nil {
				return nil, fmt.Errorf("invalid select parameter: %w", err)
			}

		case "order":
			if err := qp.parseOrder(vals[0], params); err != nil {
				return nil, fmt.Errorf("invalid order parameter: %w", err)
			}

		case "limit":
			limit, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, fmt.Errorf("invalid limit parameter: %w", err)
			}

			// Enforce max_page_size (unless it's -1 for unlimited)
			if qp.config.API.MaxPageSize > 0 && limit > qp.config.API.MaxPageSize {
				log.Debug().
					Int("requested", limit).
					Int("max", qp.config.API.MaxPageSize).
					Msg("Limit capped to max_page_size")
				limit = qp.config.API.MaxPageSize
			}

			params.Limit = &limit

		case "offset":
			offset, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, fmt.Errorf("invalid offset parameter: %w", err)
			}
			params.Offset = &offset

		case "cursor":
			// Base64-encoded cursor for keyset pagination
			cursor := vals[0]
			if cursor != "" {
				params.Cursor = &cursor
			}

		case "cursor_column":
			// Column to use for cursor (must be a valid identifier)
			col := vals[0]
			if col != "" {
				if !isValidIdentifier(col) {
					return nil, fmt.Errorf("invalid cursor_column: must be a valid column name")
				}
				params.CursorColumn = &col
			}

		case "count":
			params.Count = CountType(vals[0])

		case "truncate":
			// Truncate text columns to specified length (for table browsing)
			truncateLen, err := strconv.Atoi(vals[0])
			if err != nil {
				return nil, fmt.Errorf("invalid truncate parameter: %w", err)
			}
			if truncateLen < 0 {
				return nil, fmt.Errorf("truncate must be a non-negative integer")
			}
			params.TruncateLength = &truncateLen

		case "group_by":
			// Parse GROUP BY columns: group_by=category,status
			columns := strings.Split(vals[0], ",")
			for _, col := range columns {
				col = strings.TrimSpace(col)
				if col != "" {
					// Validate column name to prevent SQL injection
					if !isValidIdentifier(col) {
						return nil, fmt.Errorf("invalid group_by column name: %s", col)
					}
					params.GroupBy = append(params.GroupBy, col)
				}
			}

		default:
			// Check if it's a filter parameter
			// PostgREST format: column=operator.value (dot in value)
			// Old format: column.operator=value (dot in key)
			// Process ALL values for the same key to support range queries like:
			// ?recorded_at=gte.2025-01-01&recorded_at=lte.2025-12-31
			for _, val := range vals {
				if strings.Contains(key, ".") || strings.Contains(val, ".") || key == "or" || key == "and" {
					if err := qp.parseFilter(key, val, params); err != nil {
						return nil, fmt.Errorf("invalid filter parameter %s: %w", key, err)
					}
				}
			}
		}
	}

	// Apply default limit if none specified (unless default is -1)
	if params.Limit == nil && qp.config.API.DefaultPageSize > 0 {
		defaultLimit := qp.config.API.DefaultPageSize
		params.Limit = &defaultLimit
		log.Debug().
			Int("default", defaultLimit).
			Msg("Applied default_page_size")
	}

	// Validate total results limit (offset + limit <= max_total_results)
	// Skip this check if BypassMaxTotalResults is set (e.g., for admin users)
	if !opts.BypassMaxTotalResults && qp.config.API.MaxTotalResults > 0 {
		offset := 0
		if params.Offset != nil {
			offset = *params.Offset
		}

		limit := 0
		if params.Limit != nil {
			limit = *params.Limit
		}

		totalRows := offset + limit
		if totalRows > qp.config.API.MaxTotalResults {
			// Cap the limit so that offset + limit = max_total_results
			cappedLimit := qp.config.API.MaxTotalResults - offset
			if cappedLimit < 0 {
				cappedLimit = 0
			}

			log.Debug().
				Int("offset", offset).
				Int("requested_limit", limit).
				Int("capped_limit", cappedLimit).
				Int("max_total", qp.config.API.MaxTotalResults).
				Msg("Limit capped due to max_total_results")

			params.Limit = &cappedLimit
		}
	}

	return params, nil
}
