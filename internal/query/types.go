// Package query provides shared query-related types used by both the API and MCP packages.
// This package breaks the import cycle between internal/api and internal/mcp/tools.
package query

// FilterOperator represents comparison operators
type FilterOperator string

const (
	OpEqual          FilterOperator = "eq"
	OpNotEqual       FilterOperator = "neq"
	OpGreaterThan    FilterOperator = "gt"
	OpGreaterOrEqual FilterOperator = "gte"
	OpLessThan       FilterOperator = "lt"
	OpLessOrEqual    FilterOperator = "lte"
	OpLike           FilterOperator = "like"
	OpILike          FilterOperator = "ilike"
	OpIn             FilterOperator = "in"
	OpNotIn          FilterOperator = "nin"
	OpIs             FilterOperator = "is"
	OpIsNot          FilterOperator = "isnot"
	OpContains       FilterOperator = "cs"    // contains (array/jsonb) @>
	OpContained      FilterOperator = "cd"    // contained by (array/jsonb) <@
	OpContainedBy    FilterOperator = "cd"    // alias for OpContained
	OpOverlap        FilterOperator = "ov"    // overlap (array) &&
	OpOverlaps       FilterOperator = "ov"    // alias for OpOverlap
	OpTextSearch     FilterOperator = "fts"   // full text search
	OpPhraseSearch   FilterOperator = "plfts" // phrase search
	OpWebSearch      FilterOperator = "wfts"  // web search
	OpNot            FilterOperator = "not"   // negation
	OpAdjacent       FilterOperator = "adj"   // adjacent range <<
	OpStrictlyLeft   FilterOperator = "sl"    // strictly left of <<
	OpStrictlyRight  FilterOperator = "sr"    // strictly right of >>
	OpNotExtendRight FilterOperator = "nxr"   // does not extend to right &<
	OpNotExtendLeft  FilterOperator = "nxl"   // does not extend to left &>

	// PostGIS spatial operators
	OpSTIntersects FilterOperator = "st_intersects" // ST_Intersects - geometries intersect
	OpSTContains   FilterOperator = "st_contains"   // ST_Contains - geometry A contains B
	OpSTWithin     FilterOperator = "st_within"     // ST_Within - geometry A is within B
	OpSTDWithin    FilterOperator = "st_dwithin"    // ST_DWithin - geometries within distance
	OpSTDistance   FilterOperator = "st_distance"   // ST_Distance - distance between geometries
	OpSTTouches    FilterOperator = "st_touches"    // ST_Touches - geometries touch
	OpSTCrosses    FilterOperator = "st_crosses"    // ST_Crosses - geometries cross
	OpSTOverlaps   FilterOperator = "st_overlaps"   // ST_Overlaps - geometries overlap

	// pgvector similarity operators
	OpVectorL2     FilterOperator = "vec_l2"  // L2/Euclidean distance <-> (lower = more similar)
	OpVectorCosine FilterOperator = "vec_cos" // Cosine distance <=> (lower = more similar)
	OpVectorIP     FilterOperator = "vec_ip"  // Negative inner product <#> (lower = more similar)
)

// Filter represents a WHERE condition
type Filter struct {
	Column    string
	Operator  FilterOperator
	Value     interface{}
	IsOr      bool // OR instead of AND
	OrGroupID int  // Groups OR filters together (filters with same non-zero ID are ORed)
}

// OrderBy represents an ORDER BY clause
type OrderBy struct {
	Column      string
	Desc        bool
	Nulls       string         // "first" or "last"
	NullsFirst  bool           // Deprecated: use Nulls field instead
	VectorOp    FilterOperator // Vector operator for similarity ordering (vec_l2, vec_cos, vec_ip)
	VectorValue interface{}    // Vector value for similarity ordering
}
