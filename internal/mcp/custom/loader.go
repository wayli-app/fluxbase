package custom

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/rs/zerolog/log"
)

// Loader handles loading custom MCP tools from the filesystem.
type Loader struct {
	toolsDir string
}

// NewLoader creates a new Loader instance.
func NewLoader(toolsDir string) *Loader {
	return &Loader{toolsDir: toolsDir}
}

// LoadedTool represents a tool loaded from the filesystem.
type LoadedTool struct {
	Name           string
	Namespace      string
	Description    string
	Code           string
	TimeoutSeconds int
	MemoryLimitMB  int
	AllowNet       bool
	AllowEnv       bool
	AllowRead      bool
	AllowWrite     bool
	RequiredScopes []string
}

// LoadAll loads all MCP tools from the configured directory.
func (l *Loader) LoadAll() ([]*LoadedTool, error) {
	// Check if directory exists
	if _, err := os.Stat(l.toolsDir); os.IsNotExist(err) {
		log.Debug().Str("dir", l.toolsDir).Msg("MCP tools directory does not exist, skipping")
		return nil, nil
	}

	entries, err := os.ReadDir(l.toolsDir)
	if err != nil {
		return nil, err
	}

	var tools []*LoadedTool
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".ts") && !strings.HasSuffix(name, ".js") {
			continue
		}

		tool, err := l.loadTool(name)
		if err != nil {
			log.Warn().Err(err).Str("file", name).Msg("Failed to load MCP tool")
			continue
		}

		tools = append(tools, tool)
	}

	return tools, nil
}

// loadTool loads a single tool from the filesystem.
func (l *Loader) loadTool(filename string) (*LoadedTool, error) {
	filePath := filepath.Join(l.toolsDir, filename)
	content, err := os.ReadFile(filePath) //nolint:gosec // Intentional read from configured directory
	if err != nil {
		return nil, err
	}

	// Parse annotations from code
	toolName, annotations := parseAnnotations(string(content), filename)

	// Build tool from annotations
	tool := &LoadedTool{
		Name:           toolName,
		Namespace:      "default",
		Code:           string(content),
		TimeoutSeconds: 30,
		MemoryLimitMB:  128,
		AllowNet:       true, // Default to allowing network access
		AllowEnv:       false,
		AllowRead:      false,
		AllowWrite:     false,
	}

	// Apply annotations
	if ns, ok := annotations["namespace"]; ok {
		tool.Namespace = ns.(string)
	}
	if desc, ok := annotations["description"]; ok {
		tool.Description = desc.(string)
	}
	if timeout, ok := annotations["timeout"]; ok {
		if t, ok := timeout.(int); ok {
			tool.TimeoutSeconds = t
		}
	}
	if memory, ok := annotations["memory"]; ok {
		if m, ok := memory.(int); ok {
			tool.MemoryLimitMB = m
		}
	}
	if _, ok := annotations["allow-net"]; ok {
		tool.AllowNet = true
	}
	if _, ok := annotations["allow-env"]; ok {
		tool.AllowEnv = true
	}
	if _, ok := annotations["allow-read"]; ok {
		tool.AllowRead = true
	}
	if _, ok := annotations["allow-write"]; ok {
		tool.AllowWrite = true
	}
	if scopes, ok := annotations["scopes"]; ok {
		if s, ok := scopes.(string); ok {
			tool.RequiredScopes = strings.Split(s, ",")
			for i, scope := range tool.RequiredScopes {
				tool.RequiredScopes[i] = strings.TrimSpace(scope)
			}
		}
	}

	return tool, nil
}

// parseAnnotations parses @fluxbase: annotations from code.
func parseAnnotations(code, filename string) (name string, annotations map[string]interface{}) {
	annotations = make(map[string]interface{})

	// Default name from filename
	name = strings.TrimSuffix(filename, ".ts")
	name = strings.TrimSuffix(name, ".js")
	name = strings.ReplaceAll(name, "-", "_")

	lines := strings.Split(code, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "//") {
			continue
		}

		line = strings.TrimPrefix(line, "//")
		line = strings.TrimSpace(line)

		// Support @fluxbase: annotations
		if !strings.HasPrefix(line, "@fluxbase:") {
			continue
		}

		line = strings.TrimPrefix(line, "@fluxbase:")
		parts := strings.SplitN(line, " ", 2)

		key := parts[0]
		var value interface{} = true
		if len(parts) > 1 {
			valueStr := strings.TrimSpace(parts[1])
			// Try to parse as int for timeout/memory
			if key == "timeout" || key == "memory" {
				if intVal, err := strconv.Atoi(valueStr); err == nil {
					value = intVal
				} else {
					value = valueStr
				}
			} else {
				value = valueStr
			}
		}

		annotations[key] = value

		// Override name if specified
		if key == "name" {
			name = value.(string)
		}
	}

	return name, annotations
}
