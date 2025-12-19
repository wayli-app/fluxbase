package cmd

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Manage system settings",
	Long:  `View and modify system settings.`,
}

var settingsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all settings",
	Long: `List all system settings.

Examples:
  fluxbase settings list`,
	PreRunE: requireAuth,
	RunE:    runSettingsList,
}

var settingsGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a setting value",
	Long: `Get a specific setting value.

Examples:
  fluxbase settings get auth.signup_enabled`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runSettingsGet,
}

var settingsSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a setting value",
	Long: `Set a system setting value.

Examples:
  fluxbase settings set auth.signup_enabled true
  fluxbase settings set api.rate_limit 100`,
	Args:    cobra.ExactArgs(2),
	PreRunE: requireAuth,
	RunE:    runSettingsSet,
}

func init() {
	settingsCmd.AddCommand(settingsListCmd)
	settingsCmd.AddCommand(settingsGetCmd)
	settingsCmd.AddCommand(settingsSetCmd)
}

func runSettingsList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var settings map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/system/settings", nil, &settings); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		// Flatten settings for table display
		var rows [][]string
		flattenSettings("", settings, &rows)

		data := output.TableData{
			Headers: []string{"KEY", "VALUE"},
			Rows:    rows,
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(settings)
	}

	return nil
}

func flattenSettings(prefix string, settings map[string]interface{}, rows *[][]string) {
	for key, value := range settings {
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case map[string]interface{}:
			flattenSettings(fullKey, v, rows)
		default:
			*rows = append(*rows, []string{fullKey, fmt.Sprintf("%v", v)})
		}
	}
}

func runSettingsGet(cmd *cobra.Command, args []string) error {
	key := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var settings map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/system/settings", nil, &settings); err != nil {
		return err
	}

	// Navigate to the key
	value := navigateSettings(settings, key)
	if value == nil {
		return fmt.Errorf("setting '%s' not found", key)
	}

	formatter := GetFormatter()
	return formatter.Print(value)
}

func navigateSettings(settings map[string]interface{}, key string) interface{} {
	parts := splitKey(key)
	current := interface{}(settings)

	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			if v, exists := m[part]; exists {
				current = v
			} else {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

func splitKey(key string) []string {
	var parts []string
	current := ""
	for _, c := range key {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func runSettingsSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Parse value
	var parsedValue interface{} = value
	if value == "true" {
		parsedValue = true
	} else if value == "false" {
		parsedValue = false
	}

	body := map[string]interface{}{
		"key":   key,
		"value": parsedValue,
	}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/system/settings", body, nil); err != nil {
		return err
	}

	fmt.Printf("Setting '%s' updated to '%v'.\n", key, value)
	return nil
}
