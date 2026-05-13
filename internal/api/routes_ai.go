package api

import (
	"github.com/gofiber/fiber/v3"

	"github.com/nimbleflux/fluxbase/internal/ai"
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildAIRouteDeps() *routes.AIDeps {
	if s.AI.Chat == nil || s.AI.Handler == nil {
		return nil
	}
	return &routes.AIDeps{
		RequireAIEnabled:       middleware.RequireAIEnabled(s.Auth.SettingsCache),
		OptionalAuth:           s.optionalAuth,
		RequireAuth:            s.requireAuth,
		TenantMiddleware:       s.Middleware.Tenant,
		HandleWebSocket:        s.AI.Chat.HandleWebSocket,
		ListPublicChatbots:     s.AI.Handler.ListPublicChatbots,
		LookupChatbotByName:    s.AI.Handler.LookupChatbotByName,
		GetPublicChatbot:       s.AI.Handler.GetPublicChatbot,
		ListUserConversations:  s.AI.Handler.ListUserConversations,
		GetUserConversation:    s.AI.Handler.GetUserConversation,
		DeleteUserConversation: s.AI.Handler.DeleteUserConversation,
		UpdateUserConversation: s.AI.Handler.UpdateUserConversation,
	}
}

func (s *Server) buildInternalAIRouteDeps() *routes.InternalAIDeps {
	if s.AI.Internal == nil {
		return nil
	}
	return &routes.InternalAIDeps{
		RequireInternal:     middleware.RequireInternal(),
		RequireAuth:         s.requireAuth,
		HandleChat:          s.AI.Internal.HandleChat,
		HandleEmbed:         s.AI.Internal.HandleEmbed,
		HandleListProviders: s.AI.Internal.HandleListProviders,
	}
}

func knowledgeBaseDisabledHandler(c fiber.Ctx) error {
	return SendFeatureDisabled(c, "AI features")
}

func (s *Server) buildKnowledgeBaseRouteDeps() *routes.KnowledgeBaseDeps {
	deps := &routes.KnowledgeBaseDeps{
		RequireAIEnabled: middleware.RequireAIEnabled(s.Auth.SettingsCache),
		RequireAuth:      s.requireAuth,
		TenantMiddleware: s.Middleware.Tenant,
	}

	if s.AI.KBStorage == nil {
		deps.ListKBs = knowledgeBaseDisabledHandler
		deps.CreateKB = knowledgeBaseDisabledHandler
		deps.GetKB = knowledgeBaseDisabledHandler
		deps.ShareKB = knowledgeBaseDisabledHandler
		deps.ListPermissions = knowledgeBaseDisabledHandler
		deps.RevokePermission = knowledgeBaseDisabledHandler
		deps.ListDocuments = knowledgeBaseDisabledHandler
		deps.GetDocument = knowledgeBaseDisabledHandler
		deps.AddDocument = knowledgeBaseDisabledHandler
		deps.UploadDocument = knowledgeBaseDisabledHandler
		deps.DeleteDocument = knowledgeBaseDisabledHandler
		deps.SearchKB = knowledgeBaseDisabledHandler
		return deps
	}

	handler := ai.NewUserKnowledgeBaseHandler(s.AI.KBStorage)
	if s.AI.DocProcessor != nil {
		handler = ai.NewUserKnowledgeBaseHandlerWithProcessor(s.AI.KBStorage, s.AI.DocProcessor)
	}

	deps.ListKBs = handler.ListMyKnowledgeBases
	deps.CreateKB = handler.CreateMyKnowledgeBase
	deps.GetKB = handler.GetMyKnowledgeBase
	deps.ShareKB = handler.ShareKnowledgeBase
	deps.ListPermissions = handler.ListPermissions
	deps.RevokePermission = handler.RevokePermission

	if s.AI.DocProcessor != nil {
		deps.ListDocuments = handler.ListMyDocuments
		deps.GetDocument = handler.GetMyDocument
		deps.AddDocument = handler.AddMyDocument
		deps.UploadDocument = handler.UploadMyDocument
		deps.DeleteDocument = handler.DeleteMyDocument
		deps.SearchKB = handler.SearchMyKB
	}

	return deps
}
