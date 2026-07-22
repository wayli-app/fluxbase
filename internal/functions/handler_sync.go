package functions

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"

	"github.com/nimbleflux/fluxbase/internal/database"
	apperrors "github.com/nimbleflux/fluxbase/internal/errors"
	"github.com/nimbleflux/fluxbase/internal/middleware"

	syncframework "github.com/nimbleflux/fluxbase/internal/sync"
	"github.com/nimbleflux/fluxbase/internal/util"
)

// RegisterAdminRoutes registers admin-only routes for functions management
// These routes should be called with UnifiedAuthMiddleware and RequireRole("admin", "instance_admin")
func (h *Handler) RegisterAdminRoutes(app *fiber.App) {
	// Admin-only function reload endpoint
	app.Post("/api/v1/admin/functions/reload", h.ReloadFunctions)
}

// bundleFunctionFromFilesystem loads function code with supporting files and shared modules,
// then bundles it. Returns bundled code, original code, bundled status, and any error.
func (h *Handler) bundleFunctionFromFilesystem(ctx context.Context, functionName string) (bundledCode string, originalCode string, isBundled bool, bundleError *string, err error) {
	// Load main code and supporting files
	mainCode, supportingFiles, err := LoadFunctionCodeWithFiles(h.functionsDir, functionName)
	if err != nil {
		return "", "", false, nil, fmt.Errorf("failed to load code: %w", err)
	}

	// Load shared modules from filesystem
	sharedModules, err := LoadSharedModulesFromFilesystem(h.functionsDir)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load shared modules from filesystem, continuing without them")
		sharedModules = make(map[string]string)
	}

	// Create bundler
	bundler, bundlerErr := h.createBundler()
	if bundlerErr != nil {
		// No bundler available - return unbundled code
		return mainCode, mainCode, false, nil, nil
	}

	// Determine if we need to use BundleWithFiles (multi-file or shared imports)
	hasSharedImports := strings.Contains(mainCode, "from \"_shared/") || strings.Contains(mainCode, "from '_shared/")
	hasMultipleFiles := len(supportingFiles) > 0

	var result *BundleResult
	var bundleErr error

	if hasSharedImports || hasMultipleFiles {
		// Use BundleWithFiles for multi-file or shared module support
		result, bundleErr = bundler.BundleWithFiles(ctx, mainCode, supportingFiles, sharedModules)
	} else {
		// Simple single-file bundle
		result, bundleErr = bundler.Bundle(ctx, mainCode)
	}

	if bundleErr != nil {
		// Bundling failed - return unbundled code with error
		errMsg := fmt.Sprintf("bundle failed: %v", bundleErr)
		return mainCode, mainCode, false, &errMsg, nil
	}

	// Bundling succeeded
	var bundleErrPtr *string
	if result.Error != "" {
		bundleErrPtr = &result.Error
	}

	return result.BundledCode, mainCode, result.IsBundled, bundleErrPtr, nil
}

// ReloadFunctions scans the functions directory and syncs with database
// Admin-only endpoint - requires authentication and admin role
func (h *Handler) ReloadFunctions(c fiber.Ctx) error {
	ctx := middleware.CtxWithTenant(c)

	syncCtx := database.ContextWithTenant(ctx, "")
	currentTenantID := database.TenantFromContext(ctx)

	functionFiles, err := ListFunctionFiles(h.functionsDir)
	if err != nil {
		return apperrors.SendOperationFailed(c, "scan functions directory")
	}

	allFunctions, err := h.storage.ListFunctionsForSync(syncCtx, currentTenantID)
	if err != nil {
		return apperrors.SendOperationFailed(c, "list existing functions")
	}

	// Build set of function names on disk
	diskFunctionNames := make(map[string]bool)
	for _, fileInfo := range functionFiles {
		diskFunctionNames[fileInfo.Name] = true
	}

	// Track results
	created := []string{}
	updated := []string{}
	deleted := []string{}
	errors := []string{}

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunctionForSync(syncCtx, fileInfo.Name, currentTenantID)

		if err != nil {
			// Function doesn't exist in database - create it
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         &originalCode,
				IsBundled:            isBundled,
				BundleError:          bundleError,
				Enabled:              true,
				TimeoutSeconds:       30,
				MemoryLimitMB:        128,
				AllowNet:             true,
				AllowEnv:             true,
				AllowRead:            false,
				AllowWrite:           false,
				AllowedDomains:       config.AllowedDomains,
				AllowUnauthenticated: config.AllowUnauthenticated,
				IsPublic:             config.IsPublic,
				DisableExecutionLogs: config.DisableExecutionLogs,
				Source:               "filesystem",
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := originalCode
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated || existingFn.IsPublic != config.IsPublic || existingFn.DisableExecutionLogs != config.DisableExecutionLogs {
				updates := map[string]interface{}{
					"code":                   bundledCode,
					"original_code":          originalCode,
					"is_bundled":             isBundled,
					"bundle_error":           bundleError,
					"allow_unauthenticated":  config.AllowUnauthenticated,
					"is_public":              config.IsPublic,
					"disable_execution_logs": config.DisableExecutionLogs,
				}

				if err := h.storage.UpdateFunctionForSync(syncCtx, fileInfo.Name, currentTenantID, updates); err != nil {
					errors = append(errors, fmt.Sprintf("%s: failed to update: %v", fileInfo.Name, err))
					continue
				}

				updated = append(updated, fileInfo.Name)
			}
		}
	}

	// Delete functions that exist in database but not on disk
	// Only delete filesystem-sourced functions, preserve API-created ones
	for _, dbFunc := range allFunctions {
		if !diskFunctionNames[dbFunc.Name] && dbFunc.Source == "filesystem" {
			// Function exists in DB but not on disk - delete it
			if err := h.storage.DeleteFunctionForSync(syncCtx, dbFunc.Name, "default", currentTenantID); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to delete: %v", dbFunc.Name, err))
				continue
			}
			deleted = append(deleted, dbFunc.Name)
		}
	}

	return c.JSON(fiber.Map{
		"message": "Functions reloaded from filesystem",
		"created": created,
		"updated": updated,
		"deleted": deleted,
		"errors":  errors,
		"total":   len(functionFiles),
	})
}

func (h *Handler) SyncFunctions(c fiber.Ctx) error {
	var req struct {
		Namespace string `json:"namespace"`
		Functions []struct {
			Name                 string  `json:"name"`
			Description          *string `json:"description"`
			Code                 string  `json:"code"`
			OriginalCode         *string `json:"original_code"`
			IsBundled            *bool   `json:"is_bundled"`
			Enabled              *bool   `json:"enabled"`
			TimeoutSeconds       *int    `json:"timeout_seconds"`
			MemoryLimitMB        *int    `json:"memory_limit_mb"`
			AllowNet             *bool   `json:"allow_net"`
			AllowEnv             *bool   `json:"allow_env"`
			AllowRead            *bool   `json:"allow_read"`
			AllowWrite           *bool   `json:"allow_write"`
			AllowUnauthenticated *bool   `json:"allow_unauthenticated"`
			IsPublic             *bool   `json:"is_public"`
			CronSchedule         *string `json:"cron_schedule"`
		} `json:"functions"`
		Options struct {
			DeleteMissing bool `json:"delete_missing"`
			DryRun        bool `json:"dry_run"`
		} `json:"options"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return apperrors.SendInvalidBody(c)
	}

	namespace := req.Namespace
	if namespace == "" {
		namespace = "default"
	}

	ctx := middleware.CtxWithTenant(c)
	syncCtx := database.ContextWithTenant(ctx, "")

	var createdBy *uuid.UUID
	if userID := c.Locals("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			parsed, err := uuid.Parse(uid)
			if err == nil {
				createdBy = &parsed
			}
		}
	}

	currentTenantID := database.TenantFromContext(ctx)

	items := make([]functionSyncItem, 0, len(req.Functions))

	for _, spec := range req.Functions {
		items = append(items, functionSyncItem{
			name:                 spec.Name,
			description:          spec.Description,
			code:                 spec.Code,
			originalCode:         spec.OriginalCode,
			isBundled:            spec.IsBundled,
			enabled:              spec.Enabled,
			timeoutSeconds:       spec.TimeoutSeconds,
			memoryLimitMB:        spec.MemoryLimitMB,
			allowNet:             spec.AllowNet,
			allowEnv:             spec.AllowEnv,
			allowRead:            spec.AllowRead,
			allowWrite:           spec.AllowWrite,
			allowUnauthenticated: spec.AllowUnauthenticated,
			isPublic:             spec.IsPublic,
			cronSchedule:         spec.CronSchedule,
		})
	}

	sem := make(chan struct{}, 10)
	var wg sync.WaitGroup
	resultsChan := make(chan *bundleResult, len(req.Functions))

	sharedModules, _ := h.storage.ListSharedModules(ctx)
	sharedModulesMap := make(map[string]string)
	for _, module := range sharedModules {
		sharedModulesMap[module.ModulePath] = module.Content
	}

	for i := range req.Functions {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			spec := req.Functions[i]
			bundledCode := spec.Code
			originalCode := spec.Code
			isBundled := false
			var bundleError *string

			if spec.IsBundled != nil && *spec.IsBundled {
				isBundled = true
				if spec.OriginalCode != nil && *spec.OriginalCode != "" {
					originalCode = *spec.OriginalCode
				}
				resultsChan <- &bundleResult{
					name:         spec.Name,
					bundledCode:  bundledCode,
					originalCode: originalCode,
					isBundled:    isBundled,
					bundleError:  bundleError,
				}
				return
			}

			bundler, err := h.createBundler()
			if err != nil {
				resultsChan <- &bundleResult{
					name: spec.Name,
					err:  fmt.Errorf("failed to create bundler: %w", err),
				}
				return
			}

			hasSharedImports := strings.Contains(spec.Code, "from \"_shared/") ||
				strings.Contains(spec.Code, "from '_shared/")

			var result *BundleResult
			var bundleErr error

			if hasSharedImports {
				supportingFiles := make(map[string]string)
				result, bundleErr = bundler.BundleWithFiles(context.Background(), spec.Code, supportingFiles, sharedModulesMap)
			} else {
				result, bundleErr = bundler.Bundle(context.Background(), spec.Code)
			}

			if bundleErr != nil {
				resultsChan <- &bundleResult{
					name: spec.Name,
					err:  fmt.Errorf("bundle error: %w", bundleErr),
				}
				return
			}

			if result != nil {
				bundledCode = result.BundledCode
				isBundled = result.IsBundled
				if result.Error != "" {
					bundleError = &result.Error
				}
			}

			resultsChan <- &bundleResult{
				name:         spec.Name,
				bundledCode:  bundledCode,
				originalCode: originalCode,
				isBundled:    isBundled,
				bundleError:  bundleError,
			}
		}(i)
	}

	wg.Wait()
	close(resultsChan)

	bundleRes := make(map[string]*bundleResult)
	for result := range resultsChan {
		bundleRes[result.name] = result
	}

	syncer := &functionSyncer{
		handler:       h,
		syncCtx:       syncCtx,
		namespace:     namespace,
		tenantID:      currentTenantID,
		createdBy:     createdBy,
		bundleResults: bundleRes,
	}

	opts := syncframework.Options{
		Namespace:     namespace,
		DeleteMissing: req.Options.DeleteMissing,
		DryRun:        req.Options.DryRun,
		TenantID:      currentTenantID,
	}

	result, err := syncframework.Execute[functionSyncItem](ctx, syncer, items, opts)
	if err != nil {
		return apperrors.SendErrorWithCode(c, 500, err.Error(), apperrors.ErrCodeOperationFailed)
	}

	return c.JSON(result)
}

// LoadFromFilesystem loads functions from filesystem at boot time
// This is called from main.go if auto_load_on_boot is enabled
func (h *Handler) LoadFromFilesystem(ctx context.Context) error {
	// Scan functions directory for all .ts files
	functionFiles, err := ListFunctionFiles(h.functionsDir)
	if err != nil {
		return fmt.Errorf("failed to scan functions directory: %w", err)
	}

	// Track results
	created := []string{}
	updated := []string{}
	errors := []string{}

	// Process each function file
	for _, fileInfo := range functionFiles {
		// Check if function exists in database
		existingFn, err := h.storage.GetFunction(ctx, fileInfo.Name)

		if err != nil {
			// Function doesn't exist in database - create it
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Create new function with default settings
			fn := &EdgeFunction{
				Name:                 fileInfo.Name,
				Code:                 bundledCode,
				OriginalCode:         &originalCode,
				IsBundled:            isBundled,
				BundleError:          bundleError,
				Enabled:              true,
				TimeoutSeconds:       30,
				MemoryLimitMB:        128,
				AllowNet:             true,
				AllowEnv:             true,
				AllowRead:            false,
				AllowWrite:           false,
				AllowedDomains:       config.AllowedDomains,
				AllowUnauthenticated: config.AllowUnauthenticated,
				IsPublic:             config.IsPublic,
				DisableExecutionLogs: config.DisableExecutionLogs,
				Source:               "filesystem",
			}

			if err := h.storage.CreateFunction(ctx, fn); err != nil {
				errors = append(errors, fmt.Sprintf("%s: failed to create: %v", fileInfo.Name, err))
				continue
			}

			created = append(created, fileInfo.Name)
		} else {
			// Function exists - update code from filesystem
			bundledCode, originalCode, isBundled, bundleError, err := h.bundleFunctionFromFilesystem(ctx, fileInfo.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", fileInfo.Name, err))
				continue
			}

			// Parse configuration from code comments
			config := ParseFunctionConfig(originalCode)

			// Update if code or config has changed
			// Compare with original_code if available, otherwise with code
			compareCode := originalCode
			if existingFn.OriginalCode != nil {
				compareCode = *existingFn.OriginalCode
			}

			if existingFn.Code != bundledCode || compareCode != originalCode || existingFn.AllowUnauthenticated != config.AllowUnauthenticated || existingFn.IsPublic != config.IsPublic || existingFn.DisableExecutionLogs != config.DisableExecutionLogs {
				updates := map[string]interface{}{
					"code":                   bundledCode,
					"original_code":          originalCode,
					"is_bundled":             isBundled,
					"bundle_error":           bundleError,
					"allow_unauthenticated":  config.AllowUnauthenticated,
					"is_public":              config.IsPublic,
					"disable_execution_logs": config.DisableExecutionLogs,
				}

				if err := h.storage.UpdateFunction(ctx, fileInfo.Name, updates); err != nil {
					errors = append(errors, fmt.Sprintf("%s: failed to update: %v", fileInfo.Name, err))
					continue
				}

				updated = append(updated, fileInfo.Name)
			}
		}
	}

	// Note: Auto-load does NOT delete functions missing from filesystem
	// This prevents data loss when UI-created functions exist alongside file-based functions
	// Use the manual reload endpoint to perform full sync including deletions

	// Log results
	if len(created) > 0 || len(updated) > 0 {
		fmt.Printf("Functions loaded from filesystem: %d created, %d updated\n", len(created), len(updated))
	}
	if len(errors) > 0 {
		fmt.Printf("Errors loading functions: %v\n", errors)
	}

	return nil
}

// CreateSharedModule creates a new shared module
func (h *Handler) CreateSharedModule(c fiber.Ctx) error {
	var req struct {
		ModulePath  string  `json:"module_path"`
		Content     string  `json:"content"`
		Description *string `json:"description"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return apperrors.SendInvalidBody(c)
	}

	// Validate module_path starts with _shared/
	if !strings.HasPrefix(req.ModulePath, "_shared/") {
		return apperrors.SendBadRequest(c, "Module path must start with '_shared/'", apperrors.ErrCodeInvalidInput)
	}

	// Get user ID from context (if authenticated). JWT locals store strings,
	// not uuid.UUID — so parse manually like the SyncFunctions handler does.
	var userID *uuid.UUID
	if uid := c.Locals("user_id"); uid != nil {
		if uidStr, ok := uid.(string); ok {
			if parsed, err := uuid.Parse(uidStr); err == nil {
				userID = &parsed
			}
		} else if parsedUID, ok := uid.(uuid.UUID); ok {
			userID = &parsedUID
		}
	}

	module := &SharedModule{
		ModulePath:  req.ModulePath,
		Content:     req.Content,
		Description: req.Description,
		CreatedBy:   userID,
	}

	if err := h.storage.CreateSharedModule(middleware.CtxWithTenant(c), module); err != nil {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "already exists") {
			return apperrors.SendConflict(c, "Shared module already exists", apperrors.ErrCodeDuplicateKey)
		}
		log.Error().Err(err).Str("module_path", req.ModulePath).Msg("Failed to create shared module")
		return apperrors.SendOperationFailed(c, "create shared module")
	}

	log.Info().
		Str("module_path", module.ModulePath).
		Str("user_id", util.ToString(userID)).
		Msg("Shared module created")

	return c.Status(201).JSON(module)
}

// ListSharedModules returns all shared modules
func (h *Handler) ListSharedModules(c fiber.Ctx) error {
	modules, err := h.storage.ListSharedModules(middleware.CtxWithTenant(c))
	if err != nil {
		log.Error().Err(err).Msg("Failed to list shared modules")
		return apperrors.SendOperationFailed(c, "list shared modules")
	}

	return c.JSON(modules)
}

// GetSharedModule retrieves a shared module by path
func (h *Handler) GetSharedModule(c fiber.Ctx) error {
	// Get full path from wildcard (e.g., "cors.ts" from "/shared/cors.ts")
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	module, err := h.storage.GetSharedModule(middleware.CtxWithTenant(c), modulePath)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.SendResourceNotFound(c, "Shared module")
		}
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to get shared module")
		return apperrors.SendOperationFailed(c, "get shared module")
	}

	return c.JSON(module)
}

// UpdateSharedModule updates an existing shared module
func (h *Handler) UpdateSharedModule(c fiber.Ctx) error {
	// Get full path from wildcard
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	var req struct {
		Content     string  `json:"content"`
		Description *string `json:"description"`
	}

	if err := c.Bind().Body(&req); err != nil {
		return apperrors.SendInvalidBody(c)
	}

	if err := h.storage.UpdateSharedModule(middleware.CtxWithTenant(c), modulePath, req.Content, req.Description); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return apperrors.SendResourceNotFound(c, "Shared module")
		}
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to update shared module")
		return apperrors.SendOperationFailed(c, "update shared module")
	}

	// Get updated module
	module, err := h.storage.GetSharedModule(middleware.CtxWithTenant(c), modulePath)
	if err != nil {
		return apperrors.SendErrorWithCode(c, 500, "Module updated but failed to retrieve", apperrors.ErrCodeOperationFailed)
	}

	log.Info().
		Str("module_path", modulePath).
		Int("version", module.Version).
		Msg("Shared module updated")

	return c.JSON(module)
}

// DeleteSharedModule deletes a shared module
func (h *Handler) DeleteSharedModule(c fiber.Ctx) error {
	// Get full path from wildcard
	modulePath := strings.TrimPrefix(c.Params("*"), "/")

	// Ensure it starts with _shared/
	if !strings.HasPrefix(modulePath, "_shared/") {
		modulePath = "_shared/" + modulePath
	}

	if err := h.storage.DeleteSharedModule(middleware.CtxWithTenant(c), modulePath); err != nil {
		log.Error().Err(err).Str("module_path", modulePath).Msg("Failed to delete shared module")
		return apperrors.SendOperationFailed(c, "delete shared module")
	}

	log.Info().Str("module_path", modulePath).Msg("Shared module deleted")

	return c.JSON(fiber.Map{"message": "Shared module deleted successfully"})
}
