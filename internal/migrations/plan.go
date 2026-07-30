package migrations

import "time"

// Plan represents a migration plan from pgschema
// pgschema outputs groups containing steps, not a flat changes array
type Plan struct {
	Version           string             `json:"version"`
	PgschemaVersion   string             `json:"pgschema_version"`
	CreatedAt         string             `json:"created_at"`
	SourceFingerprint *SourceFingerprint `json:"source_fingerprint,omitempty"`
	Groups            []PlanGroup        `json:"groups"`
	Changes           []Change           `json:"changes,omitempty"` // Legacy field, populated from Groups
	DDL               string             `json:"ddl"`
	Transaction       bool               `json:"transaction"`
	Summary           *PlanSummary       `json:"summary,omitempty"`
	Duration          time.Duration      `json:"-"`
}

// SourceFingerprint represents the source fingerprint in pgschema output
type SourceFingerprint struct {
	Hash string `json:"hash"`
}

// PlanGroup represents a group of steps in the plan
type PlanGroup struct {
	Steps []PlanStep `json:"steps"`
}

// PlanStep represents a single step in the plan
type PlanStep struct {
	SQL       string         `json:"sql"`
	Type      string         `json:"type"`
	Operation string         `json:"operation"` // "create", "drop", "alter"
	Path      string         `json:"path"`
	Directive *PlanDirective `json:"directive,omitempty"`
}

// PlanDirective represents a directive in a plan step (e.g., wait for index creation)
type PlanDirective struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// PlanSummary provides a summary of the plan
type PlanSummary struct {
	TotalChanges     int `json:"total_changes"`
	CreateCount      int `json:"create_count"`
	AlterCount       int `json:"alter_count"`
	DropCount        int `json:"drop_count"`
	DestructiveCount int `json:"destructive_count"`
}

// Change represents a single schema change (derived from PlanStep)
type Change struct {
	Type        ChangeType `json:"type"`
	ObjectType  string     `json:"object_type"`
	Schema      string     `json:"schema"`
	Name        string     `json:"name"`
	SQL         string     `json:"sql"`
	Destructive bool       `json:"destructive"`
	DependsOn   []string   `json:"depends_on,omitempty"`
}

// ChangeType represents the type of schema change
type ChangeType string

const (
	ChangeCreate ChangeType = "CREATE"
	ChangeAlter  ChangeType = "ALTER"
	ChangeDrop   ChangeType = "DROP"
)

// ApplyResult represents the result of applying a plan
type ApplyResult struct {
	Applied  []Change
	Duration time.Duration
	Error    error
	// Fallback is true when the apply used the direct-apply fallback path
	// (pgschema plan could not validate the schema). In that case Applied
	// reflects executed statement count, not individual pgschema changes.
	Fallback bool
}

// ValidationResult represents the result of schema validation
type ValidationResult struct {
	Valid  bool
	Drifts []Drift
	Error  error
}

// Drift represents a schema drift between declared and actual state
type Drift struct {
	Type        string `json:"type"`
	ObjectType  string `json:"object_type"`
	Schema      string `json:"schema"`
	Name        string `json:"name"`
	SQL         string `json:"sql"`
	Destructive bool   `json:"destructive"`
}

// MigrationState represents the current state of migrations
type MigrationState struct {
	HasImperativeMigrations bool
	HasDeclarativeState     bool
	LastAppliedVersion      int64
	SchemaFingerprint       string
	HasDirtyMigrations      bool // For informational logging only - not blocking
}

// DeclarativeState represents the state record in platform.declarative_state
type DeclarativeState struct {
	ID                int       `json:"id"`
	SchemaFingerprint string    `json:"schema_fingerprint"`
	AppliedAt         time.Time `json:"applied_at"`
	AppliedBy         string    `json:"applied_by"`
	Source            string    `json:"source"` // 'fresh_install' | 'transitioned' | 'schema_apply'
}
