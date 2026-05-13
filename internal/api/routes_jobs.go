package api

import (
	"github.com/nimbleflux/fluxbase/internal/api/routes"
	"github.com/nimbleflux/fluxbase/internal/middleware"
)

func (s *Server) buildJobsRouteDeps() *routes.JobsDeps {
	if s.Jobs.Handler == nil {
		return nil
	}
	return &routes.JobsDeps{
		RequireJobsEnabled: middleware.RequireJobsEnabled(s.Auth.SettingsCache),
		RequireAuth:        s.requireAuth,
		SubmitJob:          s.Jobs.Handler.SubmitJob,
		GetJob:             s.Jobs.Handler.GetJob,
		ListJobs:           s.Jobs.Handler.ListJobs,
		CancelJob:          s.Jobs.Handler.CancelJob,
		RetryJob:           s.Jobs.Handler.RetryJob,
		GetJobLogsUser:     s.Jobs.Handler.GetJobLogsUser,

		TenantMiddleware:   s.Middleware.Tenant,
		TenantDBMiddleware: s.Middleware.TenantDB,
	}
}
