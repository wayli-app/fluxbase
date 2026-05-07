package routes

import (
	"github.com/gofiber/fiber/v3"
)

// AIAdminDeps contains dependencies for AI admin routes.
// Auth middleware is inherited from the parent admin route group.
//
// Role Access:
//   - instance_admin: Full access to all AI management operations
//   - tenant_admin: Can manage chatbots and view AI tables for their tenant
type AIAdminDeps struct {
	ListChatbots               fiber.Handler
	GetChatbot                 fiber.Handler
	ToggleChatbot              fiber.Handler
	UpdateChatbot              fiber.Handler
	DeleteChatbot              fiber.Handler
	SyncChatbots               fiber.Handler
	GetAIMetrics               fiber.Handler
	ListAIProviders            fiber.Handler
	CreateAIProvider           fiber.Handler
	UpdateAIProvider           fiber.Handler
	DeleteAIProvider           fiber.Handler
	SetDefaultAIProvider       fiber.Handler
	SetEmbeddingAIProvider     fiber.Handler
	ClearEmbeddingAIProvider   fiber.Handler
	ListAIConversations        fiber.Handler
	GetAIConversationMessages  fiber.Handler
	GetAIAuditLog              fiber.Handler
	ListExportableTables       fiber.Handler
	GetExportableTableDetails  fiber.Handler
	ExportTableToKnowledgeBase fiber.Handler
	ListKnowledgeBases         fiber.Handler
	GetKnowledgeBase           fiber.Handler
	CreateKnowledgeBase        fiber.Handler
	UpdateKnowledgeBase        fiber.Handler
	ListChatbotKnowledgeBases  fiber.Handler
	LinkKnowledgeBase          fiber.Handler
	UpdateChatbotKnowledgeBase fiber.Handler
	UnlinkKnowledgeBase        fiber.Handler
	DeleteKnowledgeBase        fiber.Handler
	ListDocuments              fiber.Handler
	GetDocument                fiber.Handler
	AddDocument                fiber.Handler
	UploadDocument             fiber.Handler
	DeleteDocument             fiber.Handler
	UpdateDocument             fiber.Handler
	DeleteDocumentsByFilter    fiber.Handler
	SearchKnowledgeBase        fiber.Handler
	DebugSearch                fiber.Handler
	ListEntities               fiber.Handler
	SearchEntities             fiber.Handler
	GetEntityRelationships     fiber.Handler
	GetKnowledgeGraph          fiber.Handler
	ListKBChatbots             fiber.Handler
	CreateTableExportSync      fiber.Handler
	ListTableExportSyncs       fiber.Handler
	UpdateTableExportSync      fiber.Handler
	DeleteTableExportSync      fiber.Handler
	TriggerTableExportSync     fiber.Handler
	GrantDocumentPermission    fiber.Handler
	ListDocumentPermissions    fiber.Handler
	RevokeDocumentPermission   fiber.Handler
}

// BuildAIAdminRoutes creates the AI admin route group.
func BuildAIAdminRoutes(deps *AIAdminDeps) *RouteGroup {
	if deps == nil {
		return nil
	}

	return &RouteGroup{
		Name:         "ai_admin",
		DefaultAuth:  AuthRequired,
		DefaultRoles: []string{"admin", "instance_admin", "tenant_admin"},
		Routes: []Route{
			// Chatbots (uses default roles)
			{Method: "GET", Path: "/ai/chatbots", Handler: deps.ListChatbots, Summary: "List chatbots"},
			{Method: "GET", Path: "/ai/chatbots/:id", Handler: deps.GetChatbot, Summary: "Get chatbot"},
			{Method: "POST", Path: "/ai/chatbots/:id/toggle", Handler: deps.ToggleChatbot, Summary: "Toggle chatbot"},
			{Method: "PUT", Path: "/ai/chatbots/:id", Handler: deps.UpdateChatbot, Summary: "Update chatbot"},
			{Method: "DELETE", Path: "/ai/chatbots/:id", Handler: deps.DeleteChatbot, Summary: "Delete chatbot"},
			// Sync/Metrics are instance_admin only (override roles)
			{Method: "POST", Path: "/ai/chatbots/sync", Handler: deps.SyncChatbots, Summary: "Sync chatbots", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/ai/metrics", Handler: deps.GetAIMetrics, Summary: "Get AI metrics", Roles: []string{"admin", "instance_admin"}},

			// AI Providers - uses default roles (admin, instance_admin, tenant_admin)
			// Tenant scoping is handled by the storage layer via X-FB-Tenant header
			{Method: "GET", Path: "/ai/providers", Handler: deps.ListAIProviders, Summary: "List AI providers"},
			{Method: "POST", Path: "/ai/providers", Handler: deps.CreateAIProvider, Summary: "Create AI provider"},
			{Method: "PUT", Path: "/ai/providers/:id", Handler: deps.UpdateAIProvider, Summary: "Update AI provider"},
			{Method: "DELETE", Path: "/ai/providers/:id", Handler: deps.DeleteAIProvider, Summary: "Delete AI provider"},
			{Method: "PUT", Path: "/ai/providers/:id/default", Handler: deps.SetDefaultAIProvider, Summary: "Set default AI provider"},
			{Method: "PUT", Path: "/ai/providers/:id/embedding", Handler: deps.SetEmbeddingAIProvider, Summary: "Set embedding AI provider"},
			{Method: "DELETE", Path: "/ai/providers/:id/embedding", Handler: deps.ClearEmbeddingAIProvider, Summary: "Clear embedding AI provider"},

			// Conversations & Audit - instance admin only (override roles)
			{Method: "GET", Path: "/ai/conversations", Handler: deps.ListAIConversations, Summary: "List AI conversations", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/ai/conversations/:id/messages", Handler: deps.GetAIConversationMessages, Summary: "Get AI conversation messages", Roles: []string{"admin", "instance_admin"}},
			{Method: "GET", Path: "/ai/audit", Handler: deps.GetAIAuditLog, Summary: "Get AI audit log", Roles: []string{"admin", "instance_admin"}},

			// AI Tables (uses default roles)
			{Method: "GET", Path: "/ai/tables", Handler: deps.ListExportableTables, Summary: "List exportable AI tables"},
			{Method: "GET", Path: "/ai/tables/:schema/:table", Handler: deps.GetExportableTableDetails, Summary: "Get exportable table details"},
			{Method: "POST", Path: "/ai/tables/:schema/:table/export", Handler: deps.ExportTableToKnowledgeBase, Summary: "Export table to knowledge base"},

			// Knowledge Bases (uses default roles)
			{Method: "GET", Path: "/ai/knowledge-bases", Handler: deps.ListKnowledgeBases, Summary: "List knowledge bases"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id", Handler: deps.GetKnowledgeBase, Summary: "Get knowledge base"},
			{Method: "POST", Path: "/ai/knowledge-bases", Handler: deps.CreateKnowledgeBase, Summary: "Create knowledge base"},
			{Method: "PUT", Path: "/ai/knowledge-bases/:id", Handler: deps.UpdateKnowledgeBase, Summary: "Update knowledge base"},
			{Method: "DELETE", Path: "/ai/knowledge-bases/:id", Handler: deps.DeleteKnowledgeBase, Summary: "Delete knowledge base"},

			// Chatbot Knowledge Base linking (uses default roles)
			{Method: "GET", Path: "/ai/chatbots/:id/knowledge-bases", Handler: deps.ListChatbotKnowledgeBases, Summary: "List chatbot knowledge bases"},
			{Method: "POST", Path: "/ai/chatbots/:id/knowledge-bases", Handler: deps.LinkKnowledgeBase, Summary: "Link knowledge base to chatbot"},
			{Method: "PUT", Path: "/ai/chatbots/:id/knowledge-bases/:kb_id", Handler: deps.UpdateChatbotKnowledgeBase, Summary: "Update chatbot knowledge base link"},
			{Method: "DELETE", Path: "/ai/chatbots/:id/knowledge-bases/:kb_id", Handler: deps.UnlinkKnowledgeBase, Summary: "Unlink knowledge base from chatbot"},

			// Knowledge Base Documents (uses default roles)
			{Method: "GET", Path: "/ai/knowledge-bases/:id/documents", Handler: deps.ListDocuments, Summary: "List KB documents"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.GetDocument, Summary: "Get KB document"},
			{Method: "POST", Path: "/ai/knowledge-bases/:id/documents", Handler: deps.AddDocument, Summary: "Add KB document"},
			{Method: "POST", Path: "/ai/knowledge-bases/:id/documents/upload", Handler: deps.UploadDocument, Summary: "Upload KB document"},
			{Method: "DELETE", Path: "/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.DeleteDocument, Summary: "Delete KB document"},
			{Method: "PATCH", Path: "/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.UpdateDocument, Summary: "Update KB document"},
			{Method: "POST", Path: "/ai/knowledge-bases/:id/documents/delete-by-filter", Handler: deps.DeleteDocumentsByFilter, Summary: "Delete KB documents by filter"},

			// Knowledge Base Search (uses default roles)
			{Method: "POST", Path: "/ai/knowledge-bases/:id/search", Handler: deps.SearchKnowledgeBase, Summary: "Search knowledge base"},
			{Method: "POST", Path: "/ai/knowledge-bases/:id/debug-search", Handler: deps.DebugSearch, Summary: "Debug search"},

			// Knowledge Base Knowledge Graph / Entities (uses default roles)
			{Method: "GET", Path: "/ai/knowledge-bases/:id/entities", Handler: deps.ListEntities, Summary: "List KB entities"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id/entities/search", Handler: deps.SearchEntities, Summary: "Search KB entities"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id/entities/:entity_id/relationships", Handler: deps.GetEntityRelationships, Summary: "Get entity relationships"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id/graph", Handler: deps.GetKnowledgeGraph, Summary: "Get knowledge graph"},

			// Knowledge Base Chatbots (reverse lookup, uses default roles)
			{Method: "GET", Path: "/ai/knowledge-bases/:id/chatbots", Handler: deps.ListKBChatbots, Summary: "List KB chatbots"},

			// Knowledge Base Sync Configs (uses default roles)
			{Method: "POST", Path: "/ai/knowledge-bases/:id/sync-configs", Handler: deps.CreateTableExportSync, Summary: "Create sync config"},
			{Method: "GET", Path: "/ai/knowledge-bases/:id/sync-configs", Handler: deps.ListTableExportSyncs, Summary: "List sync configs"},
			{Method: "PATCH", Path: "/ai/knowledge-bases/:id/sync-configs/:syncId", Handler: deps.UpdateTableExportSync, Summary: "Update sync config"},
			{Method: "DELETE", Path: "/ai/knowledge-bases/:id/sync-configs/:syncId", Handler: deps.DeleteTableExportSync, Summary: "Delete sync config"},
			{Method: "POST", Path: "/ai/knowledge-bases/:id/sync-configs/:syncId/trigger", Handler: deps.TriggerTableExportSync, Summary: "Trigger sync"},

			// Knowledge Base Document Permissions (uses default roles)
			{Method: "POST", Path: "/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions", Handler: deps.GrantDocumentPermission, Summary: "Grant document permission"},
			{Method: "GET", Path: "/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions", Handler: deps.ListDocumentPermissions, Summary: "List document permissions"},
			{Method: "DELETE", Path: "/ai/knowledge-bases/:kb_id/documents/:doc_id/permissions/:user_id", Handler: deps.RevokeDocumentPermission, Summary: "Revoke document permission"},
		},
	}
}
