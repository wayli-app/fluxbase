package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// Loader handles loading chatbot definitions from the filesystem
type Loader struct {
	chatbotsDir string
}

// NewLoader creates a new chatbot loader
func NewLoader(chatbotsDir string) *Loader {
	return &Loader{
		chatbotsDir: chatbotsDir,
	}
}

// LoadAll loads all chatbots from the chatbots directory
func (l *Loader) LoadAll() ([]*Chatbot, error) {
	// Check if directory exists
	info, err := os.Stat(l.chatbotsDir)
	if err != nil {
		if os.IsNotExist(err) {
			log.Warn().Str("dir", l.chatbotsDir).Msg("Chatbots directory does not exist, skipping load")
			return nil, nil
		}
		return nil, fmt.Errorf("failed to stat chatbots directory: %w", err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("chatbots path is not a directory: %s", l.chatbotsDir)
	}

	var chatbots []*Chatbot

	// Walk the directory looking for chatbot definitions
	err = filepath.Walk(l.chatbotsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, but continue walking into them
		if info.IsDir() {
			// Skip hidden directories and node_modules
			if strings.HasPrefix(info.Name(), ".") || info.Name() == "node_modules" || info.Name() == "_shared" {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process index.ts files
		if info.Name() != "index.ts" {
			return nil
		}

		// Load the chatbot
		chatbot, err := l.loadChatbot(path)
		if err != nil {
			log.Warn().Err(err).Str("path", path).Msg("Failed to load chatbot")
			return nil // Continue with other chatbots
		}

		chatbots = append(chatbots, chatbot)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk chatbots directory: %w", err)
	}

	log.Info().Int("count", len(chatbots)).Str("dir", l.chatbotsDir).Msg("Loaded chatbots from filesystem")

	return chatbots, nil
}

// loadChatbot loads a single chatbot from a file
func (l *Loader) loadChatbot(path string) (*Chatbot, error) {
	// Read the file
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	code := string(content)

	// Extract chatbot name from directory name
	// e.g., /path/to/chatbots/location-assistant/index.ts -> location-assistant
	dir := filepath.Dir(path)
	name := filepath.Base(dir)

	// Determine namespace from parent directory (if nested)
	// e.g., /path/to/chatbots/analytics/reports/index.ts -> analytics
	relPath, err := filepath.Rel(l.chatbotsDir, dir)
	if err != nil {
		relPath = name
	}

	namespace := "default"
	parts := strings.Split(relPath, string(filepath.Separator))
	if len(parts) > 1 {
		namespace = parts[0]
		name = parts[len(parts)-1]
	}

	// Parse configuration from annotations
	config := ParseChatbotConfig(code)

	// Parse description from JSDoc
	description := ParseDescription(code)

	// Create chatbot
	chatbot := &Chatbot{
		ID:           uuid.New().String(),
		Name:         name,
		Namespace:    namespace,
		Description:  description,
		Code:         code,
		OriginalCode: code,
		IsBundled:    false, // Not bundled yet
		Enabled:      true,
		Source:       "filesystem",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Apply parsed configuration
	chatbot.ApplyConfig(config)

	log.Debug().
		Str("name", name).
		Str("namespace", namespace).
		Strs("allowed_tables", chatbot.AllowedTables).
		Strs("allowed_operations", chatbot.AllowedOperations).
		Int("max_tokens", chatbot.MaxTokens).
		Float64("temperature", chatbot.Temperature).
		Msg("Loaded chatbot from filesystem")

	return chatbot, nil
}

// LoadOne loads a single chatbot by name and namespace
func (l *Loader) LoadOne(namespace, name string) (*Chatbot, error) {
	var path string

	if namespace == "default" {
		path = filepath.Join(l.chatbotsDir, name, "index.ts")
	} else {
		path = filepath.Join(l.chatbotsDir, namespace, name, "index.ts")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, fmt.Errorf("chatbot not found: %s/%s", namespace, name)
	}

	return l.loadChatbot(path)
}

// GetChatbotsDir returns the chatbots directory
func (l *Loader) GetChatbotsDir() string {
	return l.chatbotsDir
}

// ChatbotExists checks if a chatbot exists in the filesystem
func (l *Loader) ChatbotExists(namespace, name string) bool {
	var path string

	if namespace == "default" {
		path = filepath.Join(l.chatbotsDir, name, "index.ts")
	} else {
		path = filepath.Join(l.chatbotsDir, namespace, name, "index.ts")
	}

	_, err := os.Stat(path)
	return err == nil
}

// WatchForChanges sets up a file watcher for the chatbots directory
// Returns a channel that receives updates when files change
// The caller is responsible for closing the returned channel
// NOTE: This is a placeholder - actual implementation would use fsnotify
func (l *Loader) WatchForChanges() (<-chan ChatbotChange, error) {
	// For now, return a nil channel - actual file watching can be added later
	// using github.com/fsnotify/fsnotify
	log.Warn().Msg("Chatbot file watching not yet implemented - changes require manual sync")
	return nil, nil
}

// ChatbotChange represents a change to a chatbot file
type ChatbotChange struct {
	Type      string // "created", "modified", "deleted"
	Namespace string
	Name      string
	Path      string
}

// ParseChatbotFromCode parses a chatbot from code string (for SDK-based syncing)
func (l *Loader) ParseChatbotFromCode(code string, namespace string) (*Chatbot, error) {
	// Parse configuration from annotations
	config := ParseChatbotConfig(code)

	// Parse description from JSDoc
	description := ParseDescription(code)

	// Create chatbot
	chatbot := &Chatbot{
		ID:           uuid.New().String(),
		Namespace:    namespace,
		Description:  description,
		Code:         code,
		OriginalCode: code,
		IsBundled:    false,
		Enabled:      true,
		Source:       "sdk",
		Version:      1,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Apply parsed configuration
	chatbot.ApplyConfig(config)

	return chatbot, nil
}
