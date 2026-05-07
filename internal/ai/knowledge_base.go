package ai

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// SystemUserID is used as the owner for resources created via service role authentication
const SystemUserID = "00000000-0000-0000-0000-000000000000"

// SupportedEmbeddingDimension is the vector dimension supported by the database schema.
// The ai.chunks.embedding column is defined as vector(1536). If additional dimensions
// are needed in the future, the schema must be altered accordingly.
const SupportedEmbeddingDimension = 1536

// KnowledgeBase represents a collection of documents for RAG retrieval
type KnowledgeBase struct {
	ID                  string  `json:"id"`
	Name                string  `json:"name"`
	Namespace           string  `json:"namespace"`
	Description         string  `json:"description,omitempty"`
	EmbeddingModel      string  `json:"embedding_model"`
	EmbeddingDimensions int     `json:"embedding_dimensions"`
	ChunkSize           int     `json:"chunk_size"`
	ChunkOverlap        int     `json:"chunk_overlap"`
	ChunkStrategy       string  `json:"chunk_strategy"`
	Enabled             bool    `json:"enabled"`
	DocumentCount       int     `json:"document_count"`
	TotalChunks         int     `json:"total_chunks"`
	Source              string  `json:"source"`
	CreatedBy           *string `json:"created_by,omitempty"`
	// Access control
	OwnerID    *string      `json:"owner_id,omitempty"`
	Visibility KBVisibility `json:"visibility"`
	// Quotas
	QuotaMaxDocuments    int   `json:"quota_max_documents"`
	QuotaMaxChunks       int   `json:"quota_max_chunks"`
	QuotaMaxStorageBytes int64 `json:"quota_max_storage_bytes"`
	// Pipeline configuration
	PipelineType           string                 `json:"pipeline_type"`
	PipelineConfig         map[string]interface{} `json:"pipeline_config,omitempty"`
	TransformationFunction *string                `json:"transformation_function,omitempty"`
	CreatedAt              time.Time              `json:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at"`
	// Tenant info (populated for instance admin cross-tenant queries)
	TenantID   *string `json:"tenant_id,omitempty"`
	TenantName *string `json:"tenant_name,omitempty"`
}

// KnowledgeBaseSummary is a lightweight version for listing
type KnowledgeBaseSummary struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Namespace      string  `json:"namespace"`
	Description    string  `json:"description,omitempty"`
	Enabled        bool    `json:"enabled"`
	DocumentCount  int     `json:"document_count"`
	TotalChunks    int     `json:"total_chunks"`
	UpdatedAt      string  `json:"updated_at"`
	Visibility     string  `json:"visibility,omitempty"`
	UserPermission string  `json:"user_permission,omitempty"`
	OwnerID        *string `json:"owner_id,omitempty"`
	// Tenant info (populated for instance admin cross-tenant queries)
	TenantID   *string `json:"tenant_id,omitempty"`
	TenantName *string `json:"tenant_name,omitempty"`
}

// ToSummary converts a KnowledgeBase to a summary
func (kb *KnowledgeBase) ToSummary() KnowledgeBaseSummary {
	return KnowledgeBaseSummary{
		ID:            kb.ID,
		Name:          kb.Name,
		Namespace:     kb.Namespace,
		Description:   kb.Description,
		Enabled:       kb.Enabled,
		DocumentCount: kb.DocumentCount,
		TotalChunks:   kb.TotalChunks,
		UpdatedAt:     kb.UpdatedAt.Format(time.RFC3339),
		Visibility:    string(kb.Visibility),
		OwnerID: func() *string {
			if kb.OwnerID != nil {
				id := *kb.OwnerID
				return &id
			}
			return nil
		}(),
		TenantID:   kb.TenantID,
		TenantName: kb.TenantName,
	}
}

// Document represents a source document in a knowledge base
type Document struct {
	ID              string          `json:"id"`
	KnowledgeBaseID string          `json:"knowledge_base_id"`
	Title           string          `json:"title,omitempty"`
	SourceURL       string          `json:"source_url,omitempty"`
	SourceType      string          `json:"source_type"`
	MimeType        string          `json:"mime_type,omitempty"`
	Content         string          `json:"content"`
	ContentHash     string          `json:"content_hash,omitempty"`
	Status          DocumentStatus  `json:"status"`
	ErrorMessage    string          `json:"error_message,omitempty"`
	ChunksCount     int             `json:"chunks_count"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	Tags            []string        `json:"tags,omitempty"`
	OwnerID         *string         `json:"owner_id,omitempty"` // Document owner
	CreatedBy       *string         `json:"created_by,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
	IndexedAt       *time.Time      `json:"indexed_at,omitempty"`
}

// DocumentStatus represents the processing status of a document
type DocumentStatus string

const (
	DocumentStatusPending    DocumentStatus = "pending"
	DocumentStatusProcessing DocumentStatus = "processing"
	DocumentStatusIndexed    DocumentStatus = "indexed"
	DocumentStatusFailed     DocumentStatus = "failed"
)

// DocumentSummary is a lightweight version for listing
type DocumentSummary struct {
	ID          string         `json:"id"`
	Title       string         `json:"title,omitempty"`
	SourceType  string         `json:"source_type"`
	Status      DocumentStatus `json:"status"`
	ChunksCount int            `json:"chunks_count"`
	Tags        []string       `json:"tags,omitempty"`
	UpdatedAt   string         `json:"updated_at"`
}

// ToSummary converts a Document to a summary
func (d *Document) ToSummary() DocumentSummary {
	return DocumentSummary{
		ID:          d.ID,
		Title:       d.Title,
		SourceType:  d.SourceType,
		Status:      d.Status,
		ChunksCount: d.ChunksCount,
		Tags:        d.Tags,
		UpdatedAt:   d.UpdatedAt.Format(time.RFC3339),
	}
}

// Chunk represents a document chunk with its embedding
type Chunk struct {
	ID              string          `json:"id"`
	DocumentID      string          `json:"document_id"`
	KnowledgeBaseID string          `json:"knowledge_base_id"`
	Content         string          `json:"content"`
	ChunkIndex      int             `json:"chunk_index"`
	StartOffset     *int            `json:"start_offset,omitempty"`
	EndOffset       *int            `json:"end_offset,omitempty"`
	TokenCount      *int            `json:"token_count,omitempty"`
	Embedding       []float32       `json:"embedding,omitempty"` // Not included in JSON by default
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ChatbotKnowledgeBase links a chatbot to a knowledge base
type ChatbotKnowledgeBase struct {
	ID                  string                 `json:"id"`
	ChatbotID           string                 `json:"chatbot_id"`
	KnowledgeBaseID     string                 `json:"knowledge_base_id"`
	AccessLevel         string                 `json:"access_level"` // full, filtered, tiered
	FilterExpression    map[string]interface{} `json:"filter_expression"`
	ContextWeight       float64                `json:"context_weight"`       // 0.0-1.0
	Priority            int                    `json:"priority"`             // For tiered access
	IntentKeywords      []string               `json:"intent_keywords"`      // For query routing
	MaxChunks           *int                   `json:"max_chunks"`           // NULL = use default
	SimilarityThreshold *float64               `json:"similarity_threshold"` // NULL = use default
	Enabled             bool                   `json:"enabled"`
	Metadata            map[string]interface{} `json:"metadata"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`

	// Joined fields (not in DB)
	KnowledgeBaseName string `json:"knowledge_base_name,omitempty"`
	ChatbotName       string `json:"chatbot_name,omitempty"`
}

// RetrievalResult represents a single chunk retrieved during RAG
type RetrievalResult struct {
	ChunkID           string          `json:"chunk_id"`
	DocumentID        string          `json:"document_id"`
	KnowledgeBaseID   string          `json:"knowledge_base_id"`
	KnowledgeBaseName string          `json:"knowledge_base_name,omitempty"`
	DocumentTitle     string          `json:"document_title,omitempty"`
	Content           string          `json:"content"`
	Similarity        float64         `json:"similarity"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	Tags              []string        `json:"tags,omitempty"`
}

// MetadataOperator represents the comparison operator for a metadata filter condition
type MetadataOperator string

const (
	MetadataOpEquals        MetadataOperator = "="           // Exact match
	MetadataOpNotEquals     MetadataOperator = "!="          // Not equal
	MetadataOpILike         MetadataOperator = "ILIKE"       // Case-insensitive pattern match
	MetadataOpLike          MetadataOperator = "LIKE"        // Case-sensitive pattern match
	MetadataOpIn            MetadataOperator = "IN"          // Value in list
	MetadataOpNotIn         MetadataOperator = "NOT IN"      // Value not in list
	MetadataOpGreaterThan   MetadataOperator = ">"           // Greater than
	MetadataOpGreaterThanOr MetadataOperator = ">="          // Greater than or equal
	MetadataOpLessThan      MetadataOperator = "<"           // Less than
	MetadataOpLessThanOr    MetadataOperator = "<="          // Less than or equal
	MetadataOpBetween       MetadataOperator = "BETWEEN"     // Value between two numbers
	MetadataOpIsNull        MetadataOperator = "IS NULL"     // Value is null
	MetadataOpIsNotNull     MetadataOperator = "IS NOT NULL" // Value is not null
)

// MetadataCondition represents a single metadata filter condition with an operator
type MetadataCondition struct {
	Key      string           `json:"key"`              // Metadata key
	Operator MetadataOperator `json:"operator"`         // Comparison operator
	Value    interface{}      `json:"value,omitempty"`  // Single value (for =, !=, >, <, etc.)
	Values   []interface{}    `json:"values,omitempty"` // Multiple values (for IN, NOT IN)
	Min      interface{}      `json:"min,omitempty"`    // Minimum value (for BETWEEN)
	Max      interface{}      `json:"max,omitempty"`    // Maximum value (for BETWEEN)
}

// LogicalOperator represents how conditions should be combined
type LogicalOperator string

const (
	LogicalOpAND LogicalOperator = "AND" // All conditions must match
	LogicalOpOR  LogicalOperator = "OR"  // Any condition must match
)

// MetadataFilterGroup represents a group of metadata conditions with logical operators
type MetadataFilterGroup struct {
	Conditions []MetadataCondition   `json:"conditions"`       // Filter conditions
	LogicalOp  LogicalOperator       `json:"logical_op"`       // How to combine conditions (AND/OR)
	Groups     []MetadataFilterGroup `json:"groups,omitempty"` // Nested groups for complex queries
}

// MetadataFilter for user isolation and tag filtering in vector search
// For advanced filtering, use AdvancedFilter field instead of Metadata map
type MetadataFilter struct {
	UserID         *string              // If set, filter to this user's content + global content
	Tags           []string             // Filter by tags (documents must have ALL these tags)
	IncludeGlobal  bool                 // Include content without user_id (default: true)
	Metadata       map[string]string    // Arbitrary key-value filters on document metadata (legacy, exact match only)
	AdvancedFilter *MetadataFilterGroup // Advanced filtering with operators and logical combinations
}

// VectorSearchResult represents a single search result from the vector_search tool
type VectorSearchResult struct {
	ChunkID           string   `json:"chunk_id"`
	DocumentID        string   `json:"document_id"`
	DocumentTitle     string   `json:"document_title,omitempty"`
	KnowledgeBaseName string   `json:"knowledge_base_name"`
	Content           string   `json:"content"`
	Similarity        float64  `json:"similarity"`
	Tags              []string `json:"tags,omitempty"`
}

// VectorSearchOptions contains options for explicit vector search via the tool
type VectorSearchOptions struct {
	ChatbotID        string
	Query            string
	KnowledgeBases   []string // Specific KB names, or empty for all linked
	Limit            int
	Threshold        float64
	Tags             []string
	Metadata         map[string]string    // Arbitrary key-value filters on document metadata (legacy)
	MetadataFilter   *MetadataFilterGroup // Advanced metadata filtering with operators
	UserID           *string              // For user isolation
	IsAdmin          bool                 // Admin can bypass user filter
	GraphBoostWeight float64              // How much to weight entity matches vs vector similarity (0.0-1.0, default 0)
}

// RetrievalLog records a RAG retrieval operation
type RetrievalLog struct {
	ID                  string    `json:"id"`
	ChatbotID           *string   `json:"chatbot_id,omitempty"`
	ConversationID      *string   `json:"conversation_id,omitempty"`
	KnowledgeBaseID     *string   `json:"knowledge_base_id,omitempty"`
	UserID              *string   `json:"user_id,omitempty"`
	QueryText           string    `json:"query_text"`
	QueryEmbeddingModel string    `json:"query_embedding_model,omitempty"`
	ChunksRetrieved     int       `json:"chunks_retrieved"`
	ChunkIDs            []string  `json:"chunk_ids,omitempty"`
	SimilarityScores    []float64 `json:"similarity_scores,omitempty"`
	RetrievalDurationMs int       `json:"retrieval_duration_ms"`
	CreatedAt           time.Time `json:"created_at"`
}

// CreateKnowledgeBaseRequest is the request to create a knowledge base
type CreateKnowledgeBaseRequest struct {
	Name                string        `json:"name"`
	Namespace           string        `json:"namespace,omitempty"`
	Description         string        `json:"description,omitempty"`
	Visibility          *KBVisibility `json:"visibility,omitempty"`
	EmbeddingModel      string        `json:"embedding_model,omitempty"`
	EmbeddingDimensions int           `json:"embedding_dimensions,omitempty"`
	ChunkSize           int           `json:"chunk_size,omitempty"`
	ChunkOverlap        int           `json:"chunk_overlap,omitempty"`
	ChunkStrategy       string        `json:"chunk_strategy,omitempty"`
	// InitialPermissions grants permissions to users upon creation
	InitialPermissions []KBInitialPermission `json:"initial_permissions,omitempty"`
	// OwnerID is set internally by the handler, not exposed in the API
	OwnerID *string `json:"-"`
}

// KBInitialPermission represents a permission to grant upon KB creation
type KBInitialPermission struct {
	UserID     string       `json:"user_id"`
	Permission KBPermission `json:"permission"`
}

// UpdateKnowledgeBaseRequest is the request to update a knowledge base
type UpdateKnowledgeBaseRequest struct {
	Name                *string       `json:"name,omitempty"`
	Description         *string       `json:"description,omitempty"`
	Visibility          *KBVisibility `json:"visibility,omitempty"`
	EmbeddingModel      *string       `json:"embedding_model,omitempty"`
	EmbeddingDimensions *int          `json:"embedding_dimensions,omitempty"`
	ChunkSize           *int          `json:"chunk_size,omitempty"`
	ChunkOverlap        *int          `json:"chunk_overlap,omitempty"`
	ChunkStrategy       *string       `json:"chunk_strategy,omitempty"`
	Enabled             *bool         `json:"enabled,omitempty"`
}

// CreateDocumentRequest is the request to add a document to a knowledge base
type CreateDocumentRequest struct {
	Title            string            `json:"title,omitempty"`
	Content          string            `json:"content"`
	SourceURL        string            `json:"source_url,omitempty"`
	SourceType       string            `json:"source_type,omitempty"`
	MimeType         string            `json:"mime_type,omitempty"`
	Metadata         map[string]string `json:"metadata,omitempty"`
	Tags             []string          `json:"tags,omitempty"`
	OriginalFilename string            `json:"original_filename,omitempty"`
}

// LinkKnowledgeBaseRequest is the request to link a knowledge base to a chatbot
type LinkKnowledgeBaseRequest struct {
	KnowledgeBaseID     string   `json:"knowledge_base_id"`
	MaxChunks           *int     `json:"max_chunks,omitempty"`
	SimilarityThreshold *float64 `json:"similarity_threshold,omitempty"`
	Priority            *int     `json:"priority,omitempty"`
}

// ChunkingStrategy defines the strategy for splitting documents
type ChunkingStrategy string

const (
	ChunkingStrategyRecursive ChunkingStrategy = "recursive"
	ChunkingStrategySentence  ChunkingStrategy = "sentence"
	ChunkingStrategyParagraph ChunkingStrategy = "paragraph"
	ChunkingStrategyFixed     ChunkingStrategy = "fixed"
)

// DefaultKnowledgeBaseConfig returns default configuration
func DefaultKnowledgeBaseConfig() CreateKnowledgeBaseRequest {
	return CreateKnowledgeBaseRequest{
		Namespace:           "default",
		EmbeddingModel:      "text-embedding-3-small",
		EmbeddingDimensions: 1536,
		ChunkSize:           512,
		ChunkOverlap:        50,
		ChunkStrategy:       string(ChunkingStrategyRecursive),
	}
}

// KBVisibility defines who can access the KB
type KBVisibility string

const (
	KBVisibilityPrivate KBVisibility = "private" // Owner only
	KBVisibilityShared  KBVisibility = "shared"  // Explicit permissions
	KBVisibilityPublic  KBVisibility = "public"  // All authenticated users
)

// KBPermission defines access level
type KBPermission string

const (
	KBPermissionViewer KBPermission = "viewer" // Read only
	KBPermissionEditor KBPermission = "editor" // Read + write
	KBPermissionOwner  KBPermission = "owner"  // Full control
)

// KBPermissionGrant represents a permission grant
type KBPermissionGrant struct {
	ID              string       `json:"id"`
	KnowledgeBaseID string       `json:"knowledge_base_id"`
	UserID          string       `json:"user_id"`
	Permission      KBPermission `json:"permission"`
	GrantedBy       *string      `json:"granted_by,omitempty"`
	GrantedAt       time.Time    `json:"granted_at"`
}

// ============================================================================
// DOCUMENT PERMISSIONS
// ============================================================================

// DocumentPermission defines access level for documents
type DocumentPermission string

const (
	DocumentPermissionViewer DocumentPermission = "viewer" // Read only
	DocumentPermissionEditor DocumentPermission = "editor" // Read + write
)

// DocumentPermissionGrant represents a permission grant on a document
type DocumentPermissionGrant struct {
	ID         string             `json:"id"`
	DocumentID string             `json:"document_id"`
	UserID     string             `json:"user_id"`
	Permission DocumentPermission `json:"permission"`
	GrantedBy  string             `json:"granted_by"`
	GrantedAt  time.Time          `json:"granted_at"`
}

// GrantDocumentPermissionRequest is the request to grant permission on a document
type GrantDocumentPermissionRequest struct {
	UserID     string             `json:"user_id"`
	Permission DocumentPermission `json:"permission"`
}

// UserQuota represents per-user resource quotas
type UserQuota struct {
	UserID           string    `json:"user_id"`
	MaxDocuments     int       `json:"max_documents"`
	MaxChunks        int       `json:"max_chunks"`
	MaxStorageBytes  int64     `json:"max_storage_bytes"`
	UsedDocuments    int       `json:"used_documents"`
	UsedChunks       int       `json:"used_chunks"`
	UsedStorageBytes int64     `json:"used_storage_bytes"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// QuotaUsage represents current quota usage
type QuotaUsage struct {
	UserID         string `json:"user_id"`
	DocumentsUsed  int    `json:"documents_used"`
	DocumentsLimit int    `json:"documents_limit"`
	ChunksUsed     int    `json:"chunks_used"`
	ChunksLimit    int    `json:"chunks_limit"`
	StorageUsed    int64  `json:"storage_used"`
	StorageLimit   int64  `json:"storage_limit"`
	CanAddDocument bool   `json:"can_add_document"`
	CanAddChunks   bool   `json:"can_add_chunks"`
}

// SetUserQuotaRequest is the request to set user quotas
type SetUserQuotaRequest struct {
	MaxDocuments    int   `json:"max_documents,omitempty"`
	MaxChunks       int   `json:"max_chunks,omitempty"`
	MaxStorageBytes int64 `json:"max_storage_bytes,omitempty"`
}

// QuotaError represents a quota violation error
type QuotaError struct {
	ResourceType string // "documents", "chunks", "storage"
	Used         int64  // Current usage
	Limit        int64  // Limit
	Requested    int64  // Amount requested
}

func (e *QuotaError) Error() string {
	return fmt.Sprintf("quota exceeded for %s: used=%d, limit=%d, requested=%d",
		e.ResourceType, e.Used, e.Limit, e.Requested)
}

// IsQuotaError checks if an error is a quota error
func IsQuotaError(err error) bool {
	var quotaErr *QuotaError
	return errors.As(err, &quotaErr)
}

// ============================================================================
// Knowledge Graph: Entities and Relationships
// ============================================================================

// EntityType represents the type of an entity
type EntityType string

const (
	EntityPerson        EntityType = "person"
	EntityOrganization  EntityType = "organization"
	EntityLocation      EntityType = "location"
	EntityConcept       EntityType = "concept"
	EntityProduct       EntityType = "product"
	EntityEvent         EntityType = "event"
	EntityTable         EntityType = "table"
	EntityURL           EntityType = "url"
	EntityAPIEndpoint   EntityType = "api_endpoint"
	EntityDateTime      EntityType = "datetime"
	EntityCodeReference EntityType = "code_reference"
	EntityError         EntityType = "error"
	EntityOther         EntityType = "other"
)

// Entity represents a named entity extracted from documents
type Entity struct {
	ID              string                 `json:"id"`
	KnowledgeBaseID string                 `json:"knowledge_base_id"`
	EntityType      EntityType             `json:"entity_type"`
	Name            string                 `json:"name"`
	CanonicalName   string                 `json:"canonical_name"`
	Aliases         []string               `json:"aliases"`
	Metadata        map[string]interface{} `json:"metadata"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// RelationshipType represents the type of relationship between entities
type RelationshipType string

const (
	RelWorksAt      RelationshipType = "works_at"
	RelLocatedIn    RelationshipType = "located_in"
	RelFoundedBy    RelationshipType = "founded_by"
	RelOwns         RelationshipType = "owns"
	RelPartOf       RelationshipType = "part_of"
	RelRelatedTo    RelationshipType = "related_to"
	RelKnows        RelationshipType = "knows"
	RelCustomerOf   RelationshipType = "customer_of"
	RelSupplierOf   RelationshipType = "supplier_of"
	RelInvestedIn   RelationshipType = "invested_in"
	RelAcquired     RelationshipType = "acquired"
	RelMergedWith   RelationshipType = "merged_with"
	RelCompetitorOf RelationshipType = "competitor_of"
	RelParentOf     RelationshipType = "parent_of"
	RelChildOf      RelationshipType = "child_of"
	RelSpouseOf     RelationshipType = "spouse_of"
	RelSiblingOf    RelationshipType = "sibling_of"
	RelForeignKey   RelationshipType = "foreign_key"
	RelDependsOn    RelationshipType = "depends_on"
	RelOther        RelationshipType = "other"
)

// RelationshipDirection represents the direction of a relationship
type RelationshipDirection string

const (
	DirectionForward       RelationshipDirection = "forward"
	DirectionBackward      RelationshipDirection = "backward"
	DirectionBidirectional RelationshipDirection = "bidirectional"
)

// EntityRelationship represents a connection between two entities
type EntityRelationship struct {
	ID               string                 `json:"id"`
	KnowledgeBaseID  string                 `json:"knowledge_base_id"`
	SourceEntityID   string                 `json:"source_entity_id"`
	TargetEntityID   string                 `json:"target_entity_id"`
	RelationshipType RelationshipType       `json:"relationship_type"`
	Direction        RelationshipDirection  `json:"direction"`
	Confidence       *float64               `json:"confidence,omitempty"`
	Metadata         map[string]interface{} `json:"metadata"`
	CreatedAt        time.Time              `json:"created_at"`

	// Joined fields (not in DB)
	SourceEntity *Entity `json:"source_entity,omitempty"`
	TargetEntity *Entity `json:"target_entity,omitempty"`
}

// DocumentEntity represents a mention of an entity in a document
type DocumentEntity struct {
	ID                 string    `json:"id"`
	DocumentID         string    `json:"document_id"`
	EntityID           string    `json:"entity_id"`
	MentionCount       int       `json:"mention_count"`
	FirstMentionOffset *int      `json:"first_mention_offset,omitempty"`
	Salience           float64   `json:"salience"`
	Context            string    `json:"context,omitempty"`
	CreatedAt          time.Time `json:"created_at"`

	// Joined fields (not in DB)
	Entity *Entity `json:"entity,omitempty"`
}

// RelatedEntity represents an entity found through graph traversal
type RelatedEntity struct {
	EntityID         string   `json:"entity_id"`
	EntityType       string   `json:"entity_type"`
	Name             string   `json:"name"`
	CanonicalName    string   `json:"canonical_name"`
	RelationshipType string   `json:"relationship_type"`
	Depth            int      `json:"depth"`
	Path             []string `json:"path"` // Entity IDs in traversal path
}

// EntityExtractionResult contains entities extracted from a document
type EntityExtractionResult struct {
	DocumentID       string               `json:"document_id"`
	Entities         []Entity             `json:"entities"`
	Relationships    []EntityRelationship `json:"relationships"`
	DocumentEntities []DocumentEntity     `json:"document_entities"`
}
