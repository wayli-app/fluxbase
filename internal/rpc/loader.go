package rpc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

// Loader handles loading RPC procedures from the filesystem
type Loader struct {
	proceduresDir string
}

// NewLoader creates a new loader for the given directory
func NewLoader(proceduresDir string) *Loader {
	return &Loader{
		proceduresDir: proceduresDir,
	}
}

// LoadedProcedure represents a procedure loaded from the filesystem
type LoadedProcedure struct {
	Name        string
	Namespace   string
	FilePath    string
	Code        string
	SQLQuery    string
	Annotations *Annotations
}

// LoadProcedures loads all RPC procedures from the filesystem
// Directory structure: <proceduresDir>/<namespace>/<name>.sql
// or: <proceduresDir>/<name>.sql (for default namespace)
func (l *Loader) LoadProcedures() ([]*LoadedProcedure, error) {
	if l.proceduresDir == "" {
		return nil, nil
	}

	// Check if directory exists
	if _, err := os.Stat(l.proceduresDir); os.IsNotExist(err) {
		log.Debug().Str("dir", l.proceduresDir).Msg("RPC procedures directory does not exist")
		return nil, nil
	}

	var procedures []*LoadedProcedure

	err := filepath.Walk(l.proceduresDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".sql") {
			return nil
		}

		// Load the procedure
		proc, err := l.loadProcedure(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to load RPC procedure")
			return nil // Continue loading other procedures
		}

		procedures = append(procedures, proc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk procedures directory: %w", err)
	}

	log.Info().Int("count", len(procedures)).Str("dir", l.proceduresDir).Msg("Loaded RPC procedures from filesystem")
	return procedures, nil
}

// loadProcedure loads a single procedure from a file
func (l *Loader) loadProcedure(filePath string) (*LoadedProcedure, error) {
	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	code := string(content)

	// Parse annotations
	annotations, sqlQuery, err := ParseAnnotations(code)
	if err != nil {
		return nil, fmt.Errorf("failed to parse annotations: %w", err)
	}

	// Determine namespace and name from path
	relPath, err := filepath.Rel(l.proceduresDir, filePath)
	if err != nil {
		relPath = filePath
	}

	namespace, name := l.extractNamespaceName(relPath, annotations)

	// If name is still empty, use the annotation name or file name
	if name == "" {
		if annotations.Name != "" {
			name = annotations.Name
		} else {
			name = strings.TrimSuffix(filepath.Base(filePath), ".sql")
		}
	}

	return &LoadedProcedure{
		Name:        name,
		Namespace:   namespace,
		FilePath:    filePath,
		Code:        code,
		SQLQuery:    sqlQuery,
		Annotations: annotations,
	}, nil
}

// extractNamespaceName extracts namespace and name from relative path
// Path format: namespace/name.sql or just name.sql (default namespace)
func (l *Loader) extractNamespaceName(relPath string, annotations *Annotations) (string, string) {
	// Normalize path separators
	relPath = filepath.ToSlash(relPath)

	parts := strings.Split(relPath, "/")

	var namespace, name string

	if len(parts) >= 2 {
		// Has namespace subdirectory
		namespace = parts[0]
		// Name is the file name without extension
		name = strings.TrimSuffix(parts[len(parts)-1], ".sql")
	} else {
		// No namespace, use default
		namespace = "default"
		name = strings.TrimSuffix(parts[0], ".sql")
	}

	// Override with annotation name if present
	if annotations.Name != "" {
		name = annotations.Name
	}

	return namespace, name
}

// LoadProceduresFromNamespace loads procedures from a specific namespace directory
func (l *Loader) LoadProceduresFromNamespace(namespace string) ([]*LoadedProcedure, error) {
	if l.proceduresDir == "" {
		return nil, nil
	}

	nsDir := filepath.Join(l.proceduresDir, namespace)

	// Check if namespace directory exists
	if _, err := os.Stat(nsDir); os.IsNotExist(err) {
		return nil, nil
	}

	var procedures []*LoadedProcedure

	err := filepath.Walk(nsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only process .sql files
		if !strings.HasSuffix(strings.ToLower(info.Name()), ".sql") {
			return nil
		}

		// Load the procedure
		proc, err := l.loadProcedure(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to load RPC procedure")
			return nil
		}

		// Ensure correct namespace
		proc.Namespace = namespace

		procedures = append(procedures, proc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk namespace directory: %w", err)
	}

	return procedures, nil
}

// ToProcedure converts a LoadedProcedure to a Procedure for database storage
func (lp *LoadedProcedure) ToProcedure() *Procedure {
	proc := &Procedure{
		Name:         lp.Name,
		Namespace:    lp.Namespace,
		SQLQuery:     lp.SQLQuery,
		OriginalCode: lp.Code,
		Source:       "filesystem",
		Enabled:      true,
		// Default values
		AllowedSchemas:          []string{"public"},
		AllowedTables:           []string{},
		MaxExecutionTimeSeconds: 30,
	}

	// Apply annotations
	if lp.Annotations != nil {
		ApplyAnnotations(proc, lp.Annotations)
	}

	return proc
}
