package functions

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	syncframework "github.com/nimbleflux/fluxbase/internal/sync"
	"github.com/nimbleflux/fluxbase/internal/util"
)

type functionSyncItem struct {
	name                 string
	description          *string
	code                 string
	originalCode         *string
	isBundled            *bool
	enabled              *bool
	timeoutSeconds       *int
	memoryLimitMB        *int
	allowNet             *bool
	allowEnv             *bool
	allowRead            *bool
	allowWrite           *bool
	allowUnauthenticated *bool
	isPublic             *bool
	cronSchedule         *string
}

func (i functionSyncItem) GetName() string {
	return i.name
}

type bundleResult struct {
	name         string
	bundledCode  string
	originalCode string
	isBundled    bool
	bundleError  *string
	err          error
}

type functionSyncer struct {
	handler       *Handler
	syncCtx       context.Context
	namespace     string
	tenantID      string
	createdBy     *uuid.UUID
	existingFns   map[string]*EdgeFunctionSummary
	bundleResults map[string]*bundleResult
}

func (s *functionSyncer) ListExisting(ctx context.Context, opts syncframework.Options) (map[string]string, error) {
	existingFns, err := s.handler.storage.ListFunctionsByNamespaceForSync(s.syncCtx, opts.Namespace, s.tenantID)
	if err != nil {
		return nil, err
	}

	s.existingFns = make(map[string]*EdgeFunctionSummary, len(existingFns))
	result := make(map[string]string, len(existingFns))
	for i := range existingFns {
		s.existingFns[existingFns[i].Name] = &existingFns[i]
		result[existingFns[i].Name] = existingFns[i].ID.String()
	}
	return result, nil
}

func (s *functionSyncer) IsChanged(ctx context.Context, existingID string, item functionSyncItem, opts syncframework.Options) (bool, error) {
	return true, nil
}

func (s *functionSyncer) Preprocess(ctx context.Context, item functionSyncItem) error {
	br, ok := s.bundleResults[item.name]
	if !ok {
		return fmt.Errorf("bundle result not found for function %s", item.name)
	}
	if br.err != nil {
		return br.err
	}
	return nil
}

func (s *functionSyncer) Create(ctx context.Context, item functionSyncItem, opts syncframework.Options) error {
	br := s.bundleResults[item.name]
	config := ParseFunctionConfig(item.code)

	allowUnauthenticated := config.AllowUnauthenticated
	if item.allowUnauthenticated != nil {
		allowUnauthenticated = *item.allowUnauthenticated
	}

	isPublic := config.IsPublic
	if item.isPublic != nil {
		isPublic = *item.isPublic
	}

	fn := &EdgeFunction{
		Name:                 item.name,
		Namespace:            opts.Namespace,
		Description:          item.description,
		Code:                 br.bundledCode,
		OriginalCode:         &br.originalCode,
		IsBundled:            br.isBundled,
		BundleError:          br.bundleError,
		Enabled:              util.ValueOr(item.enabled, true),
		TimeoutSeconds:       util.ValueOr(item.timeoutSeconds, 30),
		MemoryLimitMB:        util.ValueOr(item.memoryLimitMB, 128),
		AllowNet:             util.ValueOr(item.allowNet, true),
		AllowEnv:             util.ValueOr(item.allowEnv, true),
		AllowRead:            util.ValueOr(item.allowRead, false),
		AllowWrite:           util.ValueOr(item.allowWrite, false),
		AllowedDomains:       config.AllowedDomains,
		AllowUnauthenticated: allowUnauthenticated,
		IsPublic:             isPublic,
		CronSchedule:         item.cronSchedule,
		CreatedBy:            s.createdBy,
	}

	return s.handler.storage.CreateFunction(ctx, fn)
}

func (s *functionSyncer) Update(ctx context.Context, item functionSyncItem, existingID string, opts syncframework.Options) error {
	br := s.bundleResults[item.name]
	config := ParseFunctionConfig(item.code)

	allowUnauthenticated := config.AllowUnauthenticated
	if item.allowUnauthenticated != nil {
		allowUnauthenticated = *item.allowUnauthenticated
	}

	isPublic := config.IsPublic
	if item.isPublic != nil {
		isPublic = *item.isPublic
	}

	updates := map[string]interface{}{
		"code":                  br.bundledCode,
		"original_code":         br.originalCode,
		"is_bundled":            br.isBundled,
		"bundle_error":          br.bundleError,
		"allow_unauthenticated": allowUnauthenticated,
		"is_public":             isPublic,
	}

	if item.description != nil {
		updates["description"] = item.description
	}
	if item.enabled != nil {
		updates["enabled"] = *item.enabled
	}
	if item.timeoutSeconds != nil {
		updates["timeout_seconds"] = *item.timeoutSeconds
	}
	if item.memoryLimitMB != nil {
		updates["memory_limit_mb"] = *item.memoryLimitMB
	}
	if item.allowNet != nil {
		updates["allow_net"] = *item.allowNet
	}
	if item.allowEnv != nil {
		updates["allow_env"] = *item.allowEnv
	}
	if item.allowRead != nil {
		updates["allow_read"] = *item.allowRead
	}
	if item.allowWrite != nil {
		updates["allow_write"] = *item.allowWrite
	}
	if item.cronSchedule != nil {
		updates["cron_schedule"] = *item.cronSchedule
	}

	return s.handler.storage.UpdateFunctionByNamespaceForSync(s.syncCtx, item.name, opts.Namespace, s.tenantID, updates)
}

func (s *functionSyncer) Delete(ctx context.Context, name string, existingID string, opts syncframework.Options) (bool, error) {
	err := s.handler.storage.DeleteFunctionForSync(s.syncCtx, name, opts.Namespace, s.tenantID)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *functionSyncer) PostSync(ctx context.Context, result *syncframework.Result, opts syncframework.Options) error {
	return nil
}
