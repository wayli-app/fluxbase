package jobs

import (
	"context"

	"github.com/google/uuid"

	syncframework "github.com/nimbleflux/fluxbase/internal/sync"
	"github.com/nimbleflux/fluxbase/internal/util"
)

type jobSyncItem struct {
	Name                   string
	Code                   string
	Description            *string
	Enabled                *bool
	Schedule               *string
	TimeoutSeconds         *int
	MemoryLimitMB          *int
	MaxRetries             *int
	ProgressTimeoutSeconds *int
	AllowNet               *bool
	AllowEnv               *bool
	AllowRead              *bool
	AllowWrite             *bool
	RequireRoles           []string
	IsBundled              *bool
	OriginalCode           *string

	bundledCode string
	parsedCode  string
	isBundled   bool
	bundleError *string
	annotations JobAnnotations
}

func (i jobSyncItem) GetName() string {
	return i.Name
}

type jobSyncer struct {
	handler      *Handler
	syncCtx      context.Context
	namespace    string
	tenantID     string
	createdBy    *uuid.UUID
	existing     map[string]*JobFunctionSummary
	preprocessed map[string]*jobSyncItem
}

func newJobSyncer(h *Handler, syncCtx context.Context, namespace, tenantID string, createdBy *uuid.UUID) *jobSyncer {
	return &jobSyncer{
		handler:      h,
		syncCtx:      syncCtx,
		namespace:    namespace,
		tenantID:     tenantID,
		createdBy:    createdBy,
		existing:     make(map[string]*JobFunctionSummary),
		preprocessed: make(map[string]*jobSyncItem),
	}
}

func (s *jobSyncer) ListExisting(ctx context.Context, opts syncframework.Options) (map[string]string, error) {
	existingFns, err := s.handler.storage.ListJobFunctionsForSync(s.syncCtx, opts.Namespace, s.tenantID)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string, len(existingFns))
	for _, fn := range existingFns {
		s.existing[fn.Name] = fn
		result[fn.Name] = fn.ID.String()
	}
	return result, nil
}

func (s *jobSyncer) IsChanged(ctx context.Context, existingID string, item jobSyncItem, opts syncframework.Options) (bool, error) {
	return true, nil
}

func (s *jobSyncer) Preprocess(ctx context.Context, item jobSyncItem) error {
	code := item.Code
	originalCode := item.Code
	isBundled := false
	var bundleError *string

	if item.OriginalCode != nil {
		originalCode = *item.OriginalCode
	}

	if item.IsBundled != nil && *item.IsBundled {
		isBundled = true
	} else {
		bundledCode, bundleErr := s.handler.loader.BundleCode(ctx, item.Code)
		if bundleErr != nil {
			errMsg := bundleErr.Error()
			bundleError = &errMsg
		} else {
			code = bundledCode
			isBundled = true
		}
	}

	annotations := s.handler.loader.ParseAnnotations(originalCode)

	item.bundledCode = code
	item.parsedCode = originalCode
	item.isBundled = isBundled
	item.bundleError = bundleError
	item.annotations = annotations

	s.preprocessed[item.Name] = &item

	return nil
}

func (s *jobSyncer) Create(ctx context.Context, item jobSyncItem, opts syncframework.Options) error {
	pp := s.preprocessed[item.Name]
	if pp == nil {
		pp = &item
	}

	fn := &JobFunction{
		ID:                     uuid.New(),
		Name:                   item.Name,
		Namespace:              s.namespace,
		Description:            item.Description,
		Code:                   &pp.bundledCode,
		OriginalCode:           &pp.parsedCode,
		IsBundled:              pp.isBundled,
		BundleError:            pp.bundleError,
		Enabled:                util.ValueOr(item.Enabled, true),
		Schedule:               item.Schedule,
		TimeoutSeconds:         util.ValueOr(item.TimeoutSeconds, util.ValueOr(&pp.annotations.TimeoutSeconds, 300)),
		MemoryLimitMB:          util.ValueOr(item.MemoryLimitMB, util.ValueOr(&pp.annotations.MemoryLimitMB, 256)),
		MaxRetries:             util.ValueOr(item.MaxRetries, pp.annotations.MaxRetries),
		ProgressTimeoutSeconds: util.ValueOr(item.ProgressTimeoutSeconds, util.ValueOr(&pp.annotations.ProgressTimeoutSeconds, 60)),
		AllowNet:               util.ValueOr(item.AllowNet, true),
		AllowEnv:               util.ValueOr(item.AllowEnv, true),
		AllowRead:              util.ValueOr(item.AllowRead, false),
		AllowWrite:             util.ValueOr(item.AllowWrite, false),
		RequireRoles:           item.RequireRoles,
		Version:                1,
		CreatedBy:              s.createdBy,
		Source:                 "api",
	}

	return s.handler.storage.CreateJobFunction(ctx, fn)
}

func (s *jobSyncer) Update(ctx context.Context, item jobSyncItem, existingID string, opts syncframework.Options) error {
	existing, ok := s.existing[item.Name]
	if !ok {
		return nil
	}

	pp := s.preprocessed[item.Name]
	if pp == nil {
		pp = &item
	}

	updatedFn := &JobFunction{
		ID:                     existing.ID,
		Name:                   existing.Name,
		Namespace:              existing.Namespace,
		Code:                   &pp.bundledCode,
		OriginalCode:           &pp.parsedCode,
		IsBundled:              pp.isBundled,
		BundleError:            pp.bundleError,
		Description:            existing.Description,
		Enabled:                existing.Enabled,
		Schedule:               existing.Schedule,
		TimeoutSeconds:         existing.TimeoutSeconds,
		MemoryLimitMB:          existing.MemoryLimitMB,
		MaxRetries:             existing.MaxRetries,
		ProgressTimeoutSeconds: existing.ProgressTimeoutSeconds,
		AllowNet:               existing.AllowNet,
		AllowEnv:               existing.AllowEnv,
		AllowRead:              existing.AllowRead,
		AllowWrite:             existing.AllowWrite,
		RequireRoles:           existing.RequireRoles,
		Source:                 existing.Source,
	}

	if item.Description != nil {
		updatedFn.Description = item.Description
	}
	if item.Enabled != nil {
		updatedFn.Enabled = *item.Enabled
	}
	if item.Schedule != nil {
		updatedFn.Schedule = item.Schedule
	}
	if item.TimeoutSeconds != nil {
		updatedFn.TimeoutSeconds = *item.TimeoutSeconds
	} else if pp.annotations.TimeoutSeconds > 0 {
		updatedFn.TimeoutSeconds = pp.annotations.TimeoutSeconds
	}
	if item.MemoryLimitMB != nil {
		updatedFn.MemoryLimitMB = *item.MemoryLimitMB
	} else if pp.annotations.MemoryLimitMB > 0 {
		updatedFn.MemoryLimitMB = pp.annotations.MemoryLimitMB
	}
	if item.MaxRetries != nil {
		updatedFn.MaxRetries = *item.MaxRetries
	} else if pp.annotations.MaxRetries > 0 {
		updatedFn.MaxRetries = pp.annotations.MaxRetries
	}
	if item.ProgressTimeoutSeconds != nil {
		updatedFn.ProgressTimeoutSeconds = *item.ProgressTimeoutSeconds
	} else if pp.annotations.ProgressTimeoutSeconds > 0 {
		updatedFn.ProgressTimeoutSeconds = pp.annotations.ProgressTimeoutSeconds
	}
	if item.AllowNet != nil {
		updatedFn.AllowNet = *item.AllowNet
	}
	if item.AllowEnv != nil {
		updatedFn.AllowEnv = *item.AllowEnv
	}
	if item.AllowRead != nil {
		updatedFn.AllowRead = *item.AllowRead
	}
	if item.AllowWrite != nil {
		updatedFn.AllowWrite = *item.AllowWrite
	}
	if len(item.RequireRoles) > 0 {
		updatedFn.RequireRoles = item.RequireRoles
	}

	return s.handler.storage.UpdateJobFunctionForSync(s.syncCtx, s.tenantID, updatedFn)
}

func (s *jobSyncer) Delete(ctx context.Context, name string, existingID string, opts syncframework.Options) (bool, error) {
	err := s.handler.storage.DeleteJobFunctionForSync(s.syncCtx, s.tenantID, s.namespace, name)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (s *jobSyncer) PostSync(ctx context.Context, result *syncframework.Result, opts syncframework.Options) error {
	s.handler.rescheduleJobsFromNamespace(ctx, opts.Namespace)
	return nil
}
