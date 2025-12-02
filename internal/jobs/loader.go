package jobs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/wayli-app/fluxbase/internal/config"
	"github.com/wayli-app/fluxbase/internal/functions"
)

// Loader handles loading and syncing job functions
type Loader struct {
	storage *Storage
	bundler *functions.Bundler
	config  *config.JobsConfig
}

// NewLoader creates a new job loader
func NewLoader(storage *Storage, cfg *config.JobsConfig) (*Loader, error) {
	bundler, err := functions.NewBundler()
	if err != nil {
		return nil, fmt.Errorf("failed to create bundler: %w", err)
	}

	return &Loader{
		storage: storage,
		bundler: bundler,
		config:  cfg,
	}, nil
}

// LoadFromFilesystem loads job functions from the filesystem
// Similar to edge functions, supports:
// - Flat files: jobs/my-job.ts
// - Directory-based: jobs/my-job/index.ts
// - Shared modules: jobs/_shared/utils.ts
// Also deletes job functions that no longer exist on the filesystem.
func (l *Loader) LoadFromFilesystem(ctx context.Context, namespace string) error {
	jobsDir := l.config.JobsDir

	// Get existing job functions for this namespace to track deletions
	existingFunctions, err := l.storage.ListJobFunctions(ctx, namespace)
	if err != nil {
		log.Warn().Err(err).Str("namespace", namespace).Msg("Failed to list existing job functions, will not delete missing")
		existingFunctions = nil
	}

	// Build map of existing functions by name (to access Source field for deletion filtering)
	existingByName := make(map[string]*JobFunctionSummary)
	for _, fn := range existingFunctions {
		existingByName[fn.Name] = fn
	}

	// Track which job names we find on the filesystem
	foundNames := make(map[string]bool)

	// Check if jobs directory exists
	if _, err := os.Stat(jobsDir); os.IsNotExist(err) {
		log.Info().Str("dir", jobsDir).Msg("Jobs directory does not exist, skipping auto-load")
		// If directory doesn't exist but we have existing filesystem-sourced functions, delete them
		// Preserve API-created functions
		if len(existingFunctions) > 0 {
			deleteCount := 0
			for _, fn := range existingFunctions {
				if fn.Source == "filesystem" {
					if err := l.storage.DeleteJobFunction(ctx, namespace, fn.Name); err != nil {
						log.Error().Err(err).Str("job", fn.Name).Msg("Failed to delete missing job function")
					} else {
						log.Info().Str("job", fn.Name).Msg("Deleted job function (no longer on filesystem)")
						deleteCount++
					}
				}
			}
			if deleteCount > 0 {
				log.Info().Int("deleted", deleteCount).Msg("Deleted filesystem-sourced job functions that no longer exist")
			}
		}
		return nil
	}

	log.Info().
		Str("dir", jobsDir).
		Str("namespace", namespace).
		Msg("Loading job functions from filesystem")

	// Load shared modules
	sharedModules, err := l.loadSharedModules(jobsDir)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load shared modules, continuing without them")
		sharedModules = make(map[string]string)
	}

	// Load global deno.json from jobs directory if it exists
	// This provides shared import mappings for all jobs
	var globalJobsDenoJSON string
	globalDenoPath := filepath.Join(jobsDir, "deno.json")
	if content, err := os.ReadFile(globalDenoPath); err == nil {
		globalJobsDenoJSON = string(content)
		log.Info().Str("path", globalDenoPath).Msg("Loaded global deno.json for jobs")
	} else {
		// Try deno.jsonc
		globalDenoPath = filepath.Join(jobsDir, "deno.jsonc")
		if content, err := os.ReadFile(globalDenoPath); err == nil {
			globalJobsDenoJSON = string(content)
			log.Info().Str("path", globalDenoPath).Msg("Loaded global deno.jsonc for jobs")
		}
	}

	// Scan jobs directory
	entries, err := os.ReadDir(jobsDir)
	if err != nil {
		return fmt.Errorf("failed to read jobs directory: %w", err)
	}

	successCount := 0
	errorCount := 0

	for _, entry := range entries {
		// Skip hidden files and _shared directory
		if strings.HasPrefix(entry.Name(), ".") || entry.Name() == "_shared" {
			continue
		}

		// Get job name (without extension)
		jobName := entry.Name()
		if strings.HasSuffix(jobName, ".ts") || strings.HasSuffix(jobName, ".js") {
			jobName = strings.TrimSuffix(jobName, filepath.Ext(jobName))
		}

		// Track that we found this job on the filesystem
		foundNames[jobName] = true

		// Load job code
		code, supportingFiles, err := l.loadJobCode(jobsDir, entry.Name())
		if err != nil {
			log.Error().
				Err(err).
				Str("job", jobName).
				Msg("Failed to load job code")
			errorCount++
			continue
		}

		// Parse annotations from code
		annotations := parseAnnotations(code)

		// If job doesn't have its own deno.json but we have a global one, use it
		if _, hasLocalDenoJSON := supportingFiles["deno.json"]; !hasLocalDenoJSON {
			if _, hasLocalDenoJSONC := supportingFiles["deno.jsonc"]; !hasLocalDenoJSONC {
				if globalJobsDenoJSON != "" {
					supportingFiles["deno.json"] = globalJobsDenoJSON
					log.Debug().Str("job", jobName).Msg("Using global deno.json for job")
				}
			}
		} else {
			log.Debug().Str("job", jobName).Msg("Job has its own deno.json")
		}

		// Log supporting files for debugging
		if len(supportingFiles) > 0 {
			fileNames := make([]string, 0, len(supportingFiles))
			for name := range supportingFiles {
				fileNames = append(fileNames, name)
			}
			log.Debug().Str("job", jobName).Strs("files", fileNames).Msg("Supporting files for job")
		}

		// Bundle if needed
		bundledCode, bundleErr := l.bundleJob(ctx, code, supportingFiles, sharedModules)

		// Create or update job function
		jobFunction := &JobFunction{
			Name:                   jobName,
			Namespace:              namespace,
			OriginalCode:           &code,
			Enabled:                annotations.Enabled,
			TimeoutSeconds:         annotations.TimeoutSeconds,
			MemoryLimitMB:          annotations.MemoryLimitMB,
			MaxRetries:             annotations.MaxRetries,
			ProgressTimeoutSeconds: annotations.ProgressTimeoutSeconds,
			AllowNet:               annotations.AllowNet,
			AllowEnv:               annotations.AllowEnv,
			AllowRead:              annotations.AllowRead,
			AllowWrite:             annotations.AllowWrite,
			RequireRole:            annotations.RequireRole,
			Schedule:               annotations.Schedule,
			Version:                1,
			Source:                 "filesystem",
		}

		if bundleErr != nil {
			// Store bundle error but continue
			jobFunction.IsBundled = false
			errMsg := bundleErr.Error()
			jobFunction.BundleError = &errMsg
			log.Warn().
				Err(bundleErr).
				Str("job", jobName).
				Msg("Failed to bundle job, storing original code")
		} else {
			jobFunction.IsBundled = true
			jobFunction.Code = &bundledCode
		}

		// Upsert job function (atomic create or update)
		jobFunction.ID = uuid.New() // Will be overwritten by RETURNING if job already exists
		if err := l.storage.UpsertJobFunction(ctx, jobFunction); err != nil {
			log.Error().
				Err(err).
				Str("job", jobName).
				Msg("Failed to upsert job function")
			errorCount++
			continue
		}
		log.Info().Str("job", jobName).Msg("Loaded job function from filesystem")

		// Save supporting files if any
		if len(supportingFiles) > 0 {
			// Delete existing files first
			if err := l.storage.DeleteJobFunctionFiles(ctx, jobFunction.ID); err != nil {
				log.Warn().Err(err).Str("job", jobName).Msg("Failed to delete old supporting files")
			}

			// Insert new files
			for filePath, content := range supportingFiles {
				file := &JobFunctionFile{
					ID:            uuid.New(),
					JobFunctionID: jobFunction.ID,
					FilePath:      filePath,
					Content:       content,
				}
				if err := l.storage.CreateJobFunctionFile(ctx, file); err != nil {
					log.Warn().
						Err(err).
						Str("job", jobName).
						Str("file", filePath).
						Msg("Failed to save supporting file")
				}
			}
		}

		successCount++
	}

	// Delete job functions that no longer exist on the filesystem
	// Only delete filesystem-sourced functions, preserve API-created ones
	deleteCount := 0
	for name, fn := range existingByName {
		if !foundNames[name] && fn.Source == "filesystem" {
			if err := l.storage.DeleteJobFunction(ctx, namespace, name); err != nil {
				log.Error().
					Err(err).
					Str("job", name).
					Str("namespace", namespace).
					Msg("Failed to delete missing job function")
				errorCount++
			} else {
				log.Info().
					Str("job", name).
					Str("namespace", namespace).
					Msg("Deleted job function (no longer on filesystem)")
				deleteCount++
			}
		}
	}

	log.Info().
		Int("success", successCount).
		Int("deleted", deleteCount).
		Int("errors", errorCount).
		Msg("Finished loading job functions from filesystem")

	return nil
}

// loadJobCode loads a job's code and supporting files
func (l *Loader) loadJobCode(jobsDir, entryName string) (string, map[string]string, error) {
	supportingFiles := make(map[string]string)

	// Check if it's a flat file
	if strings.HasSuffix(entryName, ".ts") || strings.HasSuffix(entryName, ".js") {
		filePath := filepath.Join(jobsDir, entryName)
		code, err := os.ReadFile(filePath)
		if err != nil {
			return "", nil, fmt.Errorf("failed to read job file: %w", err)
		}
		return string(code), supportingFiles, nil
	}

	// It's a directory - check for index.ts
	jobDir := filepath.Join(jobsDir, entryName)
	dirInfo, err := os.Stat(jobDir)
	if err != nil || !dirInfo.IsDir() {
		return "", nil, fmt.Errorf("not a file or directory: %s", entryName)
	}

	// Read index.ts
	indexPath := filepath.Join(jobDir, "index.ts")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		return "", nil, fmt.Errorf("directory-based job missing index.ts: %s", entryName)
	}

	code, err := os.ReadFile(indexPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read index.ts: %w", err)
	}

	// Load supporting files
	entries, err := os.ReadDir(jobDir)
	if err != nil {
		log.Warn().Err(err).Str("job", entryName).Msg("Failed to read job directory for supporting files")
		return string(code), supportingFiles, nil
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Handle nested directories
			nestedFiles, err := l.loadNestedFiles(filepath.Join(jobDir, entry.Name()), entry.Name())
			if err != nil {
				log.Warn().
					Str("job", entryName).
					Str("dir", entry.Name()).
					Err(err).
					Msg("Failed to read nested directory")
				continue
			}
			for path, content := range nestedFiles {
				supportingFiles[path] = content
			}
			continue
		}

		fileName := entry.Name()

		// Skip index.ts (that's the main file)
		if fileName == "index.ts" {
			continue
		}

		// Process TypeScript/JavaScript files and deno.json config
		if strings.HasSuffix(fileName, ".ts") || strings.HasSuffix(fileName, ".js") ||
			strings.HasSuffix(fileName, ".mts") || strings.HasSuffix(fileName, ".mjs") ||
			fileName == "deno.json" || fileName == "deno.jsonc" {

			filePath := filepath.Join(jobDir, fileName)
			content, err := os.ReadFile(filePath)
			if err != nil {
				log.Warn().
					Str("job", entryName).
					Str("file", fileName).
					Err(err).
					Msg("Failed to read supporting file")
				continue
			}

			supportingFiles[fileName] = string(content)
		}
	}

	return string(code), supportingFiles, nil
}

// loadNestedFiles recursively loads files from nested directories
func (l *Loader) loadNestedFiles(dirPath, relativePath string) (map[string]string, error) {
	files := make(map[string]string)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			nestedPath := filepath.Join(relativePath, entry.Name())
			nestedFiles, err := l.loadNestedFiles(filepath.Join(dirPath, entry.Name()), nestedPath)
			if err != nil {
				continue
			}
			for path, content := range nestedFiles {
				files[path] = content
			}
			continue
		}

		fileName := entry.Name()
		if !strings.HasSuffix(fileName, ".ts") && !strings.HasSuffix(fileName, ".js") {
			continue
		}

		filePath := filepath.Join(dirPath, fileName)
		content, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		relativeFilePath := filepath.Join(relativePath, fileName)
		files[relativeFilePath] = string(content)
	}

	return files, nil
}

// loadSharedModules loads shared modules from _shared directory
func (l *Loader) loadSharedModules(jobsDir string) (map[string]string, error) {
	sharedModules := make(map[string]string)

	sharedDir := filepath.Join(jobsDir, "_shared")
	if _, err := os.Stat(sharedDir); os.IsNotExist(err) {
		return sharedModules, nil
	}

	files, err := l.loadNestedFiles(sharedDir, "_shared")
	if err != nil {
		return nil, err
	}

	log.Info().Int("count", len(files)).Msg("Loaded shared modules for jobs")
	return files, nil
}

// BundleCode bundles job code without supporting files or shared modules
// Used by the sync API when code is submitted directly
func (l *Loader) BundleCode(ctx context.Context, code string) (string, error) {
	return l.bundleJob(ctx, code, nil, nil)
}

// ParseAnnotations parses @fluxbase: annotations from job code (public wrapper)
func (l *Loader) ParseAnnotations(code string) JobAnnotations {
	return parseAnnotations(code)
}

// bundleJob bundles a job's code with its dependencies
func (l *Loader) bundleJob(
	ctx context.Context,
	code string,
	supportingFiles map[string]string,
	sharedModules map[string]string,
) (string, error) {
	// Check if bundling is needed
	if !l.bundler.NeedsBundle(code) && len(supportingFiles) == 0 && len(sharedModules) == 0 {
		return code, nil
	}

	// Use the functions bundler with supporting files and shared modules
	result, err := l.bundler.BundleWithFiles(ctx, code, supportingFiles, sharedModules)
	if err != nil {
		return "", err
	}

	return result.BundledCode, nil
}

// JobAnnotations represents parsed annotations from job code
type JobAnnotations struct {
	Schedule               *string
	TimeoutSeconds         int
	MemoryLimitMB          int
	MaxRetries             int
	ProgressTimeoutSeconds int
	Enabled                bool
	AllowNet               bool
	AllowEnv               bool
	AllowRead              bool
	AllowWrite             bool
	RequireRole            *string
}

// parseAnnotations parses @fluxbase: annotations from job code
func parseAnnotations(code string) JobAnnotations {
	annotations := JobAnnotations{
		TimeoutSeconds:         300,  // 5 minutes default
		MemoryLimitMB:          256,  // 256MB default
		MaxRetries:             0,    // No retries by default
		ProgressTimeoutSeconds: 60,   // 1 minute default
		Enabled:                true, // Enabled by default
		AllowNet:               true,
		AllowEnv:               true,
		AllowRead:              false,
		AllowWrite:             false,
	}

	// Parse schedule
	if match := regexp.MustCompile(`@fluxbase:schedule\s+(.+)`).FindStringSubmatch(code); match != nil {
		schedule := strings.TrimSpace(match[1])
		annotations.Schedule = &schedule
	}

	// Parse timeout
	if match := regexp.MustCompile(`@fluxbase:timeout\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if timeout, err := strconv.Atoi(match[1]); err == nil {
			annotations.TimeoutSeconds = timeout
		}
	}

	// Parse memory limit
	if match := regexp.MustCompile(`@fluxbase:memory\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if memory, err := strconv.Atoi(match[1]); err == nil {
			annotations.MemoryLimitMB = memory
		}
	}

	// Parse max retries
	if match := regexp.MustCompile(`@fluxbase:max-retries\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if retries, err := strconv.Atoi(match[1]); err == nil {
			annotations.MaxRetries = retries
		}
	}

	// Parse progress timeout
	if match := regexp.MustCompile(`@fluxbase:progress-timeout\s+(\d+)`).FindStringSubmatch(code); match != nil {
		if timeout, err := strconv.Atoi(match[1]); err == nil {
			annotations.ProgressTimeoutSeconds = timeout
		}
	}

	// Parse enabled
	if regexp.MustCompile(`@fluxbase:enabled\s+false`).MatchString(code) {
		annotations.Enabled = false
	}

	// Parse permissions
	if regexp.MustCompile(`@fluxbase:allow-read\s+true`).MatchString(code) {
		annotations.AllowRead = true
	}
	if regexp.MustCompile(`@fluxbase:allow-write\s+true`).MatchString(code) {
		annotations.AllowWrite = true
	}
	if regexp.MustCompile(`@fluxbase:allow-net\s+false`).MatchString(code) {
		annotations.AllowNet = false
	}
	if regexp.MustCompile(`@fluxbase:allow-env\s+false`).MatchString(code) {
		annotations.AllowEnv = false
	}

	// Parse require-role
	if match := regexp.MustCompile(`@fluxbase:require-role\s+(\w+)`).FindStringSubmatch(code); match != nil {
		role := strings.TrimSpace(match[1])
		annotations.RequireRole = &role
	}

	return annotations
}
