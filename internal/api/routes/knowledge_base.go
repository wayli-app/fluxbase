package routes

import (
	"github.com/gofiber/fiber/v3"
)

type KnowledgeBaseDeps struct {
	RequireAIEnabled fiber.Handler
	RequireAuth      fiber.Handler
	TenantMiddleware fiber.Handler

	ListKBs          fiber.Handler
	CreateKB         fiber.Handler
	GetKB            fiber.Handler
	ShareKB          fiber.Handler
	ListPermissions  fiber.Handler
	RevokePermission fiber.Handler

	ListDocuments      fiber.Handler
	GetDocument        fiber.Handler
	AddDocument        fiber.Handler
	UploadDocument     fiber.Handler
	DeleteDocument     fiber.Handler
	UpdateDocument     fiber.Handler
	DeleteDocsByFilter fiber.Handler
	SearchKB           fiber.Handler
	DebugSearch        fiber.Handler

	// Knowledge Graph / Entities
	ListEntities           fiber.Handler
	SearchEntities         fiber.Handler
	GetEntityRelationships fiber.Handler
	GetKnowledgeGraph      fiber.Handler

	// Chatbot links
	ListLinkedChatbots fiber.Handler
}

func BuildKnowledgeBaseRoutes(deps *KnowledgeBaseDeps) *RouteGroup {
	if deps == nil {
		return nil
	}

	routes := []Route{
		{Method: "GET", Path: "/api/v1/ai/knowledge-bases", Handler: deps.ListKBs, Summary: "List user's knowledge bases", Auth: AuthRequired},
		{Method: "POST", Path: "/api/v1/ai/knowledge-bases", Handler: deps.CreateKB, Summary: "Create knowledge base", Auth: AuthRequired},
		{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id", Handler: deps.GetKB, Summary: "Get knowledge base", Auth: AuthRequired},
		{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/share", Handler: deps.ShareKB, Summary: "Share knowledge base", Auth: AuthRequired},
		{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/permissions", Handler: deps.ListPermissions, Summary: "List KB permissions", Auth: AuthRequired},
		{Method: "DELETE", Path: "/api/v1/ai/knowledge-bases/:id/permissions/:user_id", Handler: deps.RevokePermission, Summary: "Revoke KB permission", Auth: AuthRequired},
	}

	if deps.ListDocuments != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/documents", Handler: deps.ListDocuments, Summary: "List KB documents", Auth: AuthRequired},
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.GetDocument, Summary: "Get KB document", Auth: AuthRequired},
			Route{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/documents", Handler: deps.AddDocument, Summary: "Add document to KB", Auth: AuthRequired},
			Route{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/documents/upload", Handler: deps.UploadDocument, Summary: "Upload document to KB", Auth: AuthRequired},
			Route{Method: "DELETE", Path: "/api/v1/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.DeleteDocument, Summary: "Delete KB document", Auth: AuthRequired},
			Route{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/search", Handler: deps.SearchKB, Summary: "Search knowledge base", Auth: AuthRequired},
		)
	}

	// Document management extended routes
	if deps.UpdateDocument != nil {
		routes = append(
			routes,
			Route{Method: "PATCH", Path: "/api/v1/ai/knowledge-bases/:id/documents/:doc_id", Handler: deps.UpdateDocument, Summary: "Update KB document", Auth: AuthRequired},
		)
	}
	if deps.DeleteDocsByFilter != nil {
		routes = append(
			routes,
			Route{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/documents/delete-by-filter", Handler: deps.DeleteDocsByFilter, Summary: "Delete KB documents by filter", Auth: AuthRequired},
		)
	}
	if deps.DebugSearch != nil {
		routes = append(
			routes,
			Route{Method: "POST", Path: "/api/v1/ai/knowledge-bases/:id/debug-search", Handler: deps.DebugSearch, Summary: "Debug search knowledge base", Auth: AuthRequired},
		)
	}

	// Knowledge Graph / Entities routes
	if deps.ListEntities != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/entities", Handler: deps.ListEntities, Summary: "List KB entities", Auth: AuthRequired},
		)
	}
	if deps.SearchEntities != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/entities/search", Handler: deps.SearchEntities, Summary: "Search KB entities", Auth: AuthRequired},
		)
	}
	if deps.GetEntityRelationships != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/entities/:entity_id/relationships", Handler: deps.GetEntityRelationships, Summary: "Get entity relationships", Auth: AuthRequired},
		)
	}
	if deps.GetKnowledgeGraph != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/graph", Handler: deps.GetKnowledgeGraph, Summary: "Get knowledge graph", Auth: AuthRequired},
		)
	}

	// Chatbot links
	if deps.ListLinkedChatbots != nil {
		routes = append(
			routes,
			Route{Method: "GET", Path: "/api/v1/ai/knowledge-bases/:id/chatbots", Handler: deps.ListLinkedChatbots, Summary: "List linked chatbots", Auth: AuthRequired},
		)
	}

	middlewares := []Middleware{
		{Name: "RequireAIEnabled", Handler: deps.RequireAIEnabled},
	}
	if deps.TenantMiddleware != nil {
		middlewares = append(middlewares, Middleware{
			Name: "TenantContext", Handler: deps.TenantMiddleware,
		})
	}

	return &RouteGroup{
		Name:        "knowledge_base",
		Routes:      routes,
		Middlewares: middlewares,
		AuthMiddlewares: &AuthMiddlewares{
			Required: deps.RequireAuth,
		},
	}
}
