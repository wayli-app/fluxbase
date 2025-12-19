package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	cliconfig "github.com/fluxbase-eu/fluxbase/cli/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage CLI configuration",
	Long:  `View and modify CLI configuration settings.`,
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new configuration file",
	Long: `Create a new configuration file with default settings.

Examples:
  fluxbase config init`,
	RunE: runConfigInit,
}

var configViewCmd = &cobra.Command{
	Use:   "view",
	Short: "Display current configuration",
	Long: `Show the current CLI configuration.

Examples:
  fluxbase config view
  fluxbase config view --output json`,
	RunE: runConfigView,
}

var configSetCmd = &cobra.Command{
	Use:   "set [key] [value]",
	Short: "Set a configuration value",
	Long: `Set a configuration value.

Available keys:
  defaults.output     - Default output format (table, json, yaml)
  defaults.namespace  - Default namespace for functions/jobs
  defaults.no_headers - Hide table headers by default
  defaults.quiet      - Quiet mode by default

Examples:
  fluxbase config set defaults.output json
  fluxbase config set defaults.namespace production`,
	Args: cobra.ExactArgs(2),
	RunE: runConfigSet,
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get a configuration value",
	Long: `Get a specific configuration value.

Examples:
  fluxbase config get defaults.output
  fluxbase config get current_profile`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigGet,
}

var configProfilesCmd = &cobra.Command{
	Use:   "profiles",
	Short: "List all profiles",
	Long:  `Display all configured profiles.`,
	RunE:  runConfigProfiles,
}

var configProfilesAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new profile",
	Long: `Add a new empty profile. Use 'fluxbase auth login --profile NAME' to configure it.

Examples:
  fluxbase config profiles add staging`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigProfilesAdd,
}

var configProfilesRemoveCmd = &cobra.Command{
	Use:     "remove [name]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove a profile",
	Long: `Remove a profile and its credentials.

Examples:
  fluxbase config profiles remove staging`,
	Args: cobra.ExactArgs(1),
	RunE: runConfigProfilesRemove,
}

func init() {
	configProfilesCmd.AddCommand(configProfilesAddCmd)
	configProfilesCmd.AddCommand(configProfilesRemoveCmd)

	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configViewCmd)
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	configCmd.AddCommand(configProfilesCmd)
}

func runConfigInit(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	// Check if config already exists
	if _, err := cliconfig.Load(configPath); err == nil {
		return fmt.Errorf("configuration file already exists at %s", configPath)
	}

	// Create new config
	cfg := cliconfig.New()

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to create config: %w", err)
	}

	fmt.Printf("Configuration file created at: %s\n", configPath)
	fmt.Println("Run 'fluxbase auth login' to add a profile.")
	return nil
}

func runConfigView(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return err
	}

	formatter := GetFormatter()

	// Mask credentials in output
	for _, profile := range cfg.Profiles {
		if profile.Credentials != nil {
			if profile.Credentials.AccessToken != "" {
				profile.Credentials.AccessToken = "****"
			}
			if profile.Credentials.RefreshToken != "" {
				profile.Credentials.RefreshToken = "****"
			}
			if profile.Credentials.APIKey != "" {
				profile.Credentials.APIKey = "****"
			}
		}
	}

	return formatter.Print(cfg)
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key := args[0]
	value := args[1]
	configPath := GetConfigPath()

	cfg, err := cliconfig.LoadOrCreate(configPath)
	if err != nil {
		return err
	}

	switch strings.ToLower(key) {
	case "defaults.output":
		cfg.Defaults.Output = value
	case "defaults.namespace":
		cfg.Defaults.Namespace = value
	case "defaults.no_headers":
		cfg.Defaults.NoHeaders = value == "true" || value == "1"
	case "defaults.quiet":
		cfg.Defaults.Quiet = value == "true" || value == "1"
	case "current_profile":
		if _, err := cfg.GetProfile(value); err != nil {
			return err
		}
		cfg.CurrentProfile = value
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Set %s = %s\n", key, value)
	return nil
}

func runConfigGet(cmd *cobra.Command, args []string) error {
	key := args[0]
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return err
	}

	var value string

	switch strings.ToLower(key) {
	case "version":
		value = cfg.Version
	case "current_profile":
		value = cfg.CurrentProfile
	case "defaults.output":
		value = cfg.Defaults.Output
	case "defaults.namespace":
		value = cfg.Defaults.Namespace
	case "defaults.no_headers":
		value = fmt.Sprintf("%v", cfg.Defaults.NoHeaders)
	case "defaults.quiet":
		value = fmt.Sprintf("%v", cfg.Defaults.Quiet)
	default:
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	fmt.Println(value)
	return nil
}

func runConfigProfiles(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		fmt.Println("No profiles configured. Run 'fluxbase auth login' to create one.")
		return nil
	}

	formatter := GetFormatter()

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}

	profiles := make([]map[string]interface{}, 0, len(cfg.Profiles))
	for name, profile := range cfg.Profiles {
		current := name == cfg.CurrentProfile
		profiles = append(profiles, map[string]interface{}{
			"name":    name,
			"server":  profile.Server,
			"current": current,
		})
	}

	if formatter.Format == "table" {
		for _, p := range profiles {
			current := ""
			if p["current"].(bool) {
				current = " *"
			}
			fmt.Printf("%s%s\n", p["name"], current)
			fmt.Printf("  Server: %s\n", p["server"])
		}
	} else {
		formatter.Print(profiles)
	}

	return nil
}

func runConfigProfilesAdd(cmd *cobra.Command, args []string) error {
	name := args[0]
	configPath := GetConfigPath()

	cfg, err := cliconfig.LoadOrCreate(configPath)
	if err != nil {
		return err
	}

	if _, exists := cfg.Profiles[name]; exists {
		return fmt.Errorf("profile '%s' already exists", name)
	}

	cfg.SetProfile(&cliconfig.Profile{
		Name:            name,
		CredentialStore: "file",
	})

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Profile '%s' created.\n", name)
	fmt.Printf("Run 'fluxbase auth login --profile %s' to configure it.\n", name)
	return nil
}

func runConfigProfilesRemove(cmd *cobra.Command, args []string) error {
	name := args[0]
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return err
	}

	// Delete credentials first
	credManager := cliconfig.NewCredentialManager(cfg)
	_ = credManager.DeleteCredentials(name)

	if err := cfg.DeleteProfile(name); err != nil {
		return err
	}

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Profile '%s' removed.\n", name)
	return nil
}
