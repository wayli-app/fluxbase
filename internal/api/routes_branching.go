package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildBranchRouteDeps() *routes.BranchDeps {
	if s.Branching.Handler == nil || !s.config.Branching.Enabled {
		return nil
	}

	return &routes.BranchDeps{
		GetActiveBranch:    s.Branching.Handler.GetActiveBranch,
		SetActiveBranch:    s.Branching.Handler.SetActiveBranch,
		ResetActiveBranch:  s.Branching.Handler.ResetActiveBranch,
		GetPoolStats:       s.Branching.Handler.GetPoolStats,
		CreateBranch:       s.Branching.Handler.CreateBranch,
		ListBranches:       s.Branching.Handler.ListBranches,
		GetBranch:          s.Branching.Handler.GetBranch,
		DeleteBranch:       s.Branching.Handler.DeleteBranch,
		ResetBranch:        s.Branching.Handler.ResetBranch,
		GetBranchActivity:  s.Branching.Handler.GetBranchActivity,
		ListBranchAccess:   s.Branching.Handler.ListBranchAccess,
		GrantBranchAccess:  s.Branching.Handler.GrantBranchAccess,
		RevokeBranchAccess: s.Branching.Handler.RevokeBranchAccess,
		ListGitHubConfigs:  s.Branching.Handler.ListGitHubConfigs,
		UpsertGitHubConfig: s.Branching.Handler.UpsertGitHubConfig,
		DeleteGitHubConfig: s.Branching.Handler.DeleteGitHubConfig,
	}
}

func (s *Server) buildGitHubWebhookRouteDeps() *routes.GitHubWebhookDeps {
	if s.Branching.GitHub == nil {
		return nil
	}
	return &routes.GitHubWebhookDeps{
		GitHubWebhookLimiter: middleware.GitHubWebhookLimiter(s.sharedMiddlewareStorage),
		HandleWebhook:        s.Branching.GitHub.HandleWebhook,
	}
}
