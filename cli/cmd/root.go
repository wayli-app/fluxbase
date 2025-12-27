// Package cmd provides the Cobra commands for the Fluxbase CLI.
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/fluxbase-eu/fluxbase/cli/client"
	cliconfig "github.com/fluxbase-eu/fluxbase/cli/config"
	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var (
	// Version information (set via ldflags during build)
	Version   = "dev"
	Commit    = "unknown"
	BuildDate = "unknown"

	// Global flags
	cfgFile     string
	profileName string
	outputFmt   string
	noHeaders   bool
	quiet       bool
	debug       bool

	// Shared across commands
	cfg       *cliconfig.Config
	apiClient *client.Client
	formatter *output.Formatter
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "fluxbase",
	Short: "Fluxbase CLI - Manage your Fluxbase platform",
	Long: `Fluxbase CLI provides command-line access to manage your Fluxbase platform.

Features:
  - Functions: Deploy and manage edge functions
  - Jobs: Schedule and monitor background jobs
  - Storage: Manage file storage buckets and objects
  - AI: Configure chatbots and knowledge bases
  - Database: Query tables and run migrations

Get started:
  fluxbase auth login    Login to your Fluxbase server
  fluxbase --help        Show available commands

For more information, visit https://fluxbase.eu/docs/cli`,
	SilenceUsage: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Silence errors only when --quiet is used
		cmd.SilenceErrors = quiet
	},
}

// Execute runs the CLI
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global persistent flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is ~/.fluxbase/config.yaml)")
	rootCmd.PersistentFlags().StringVarP(&profileName, "profile", "p", "",
		"profile to use (default is current profile)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "table",
		"output format: table, json, yaml")
	rootCmd.PersistentFlags().BoolVar(&noHeaders, "no-headers", false,
		"hide table headers")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false,
		"minimal output")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false,
		"enable debug output")

	// Bind environment variables
	viper.SetEnvPrefix("FLUXBASE")
	_ = viper.BindEnv("server")  // FLUXBASE_SERVER
	_ = viper.BindEnv("token")   // FLUXBASE_TOKEN
	_ = viper.BindEnv("profile") // FLUXBASE_PROFILE
	_ = viper.BindEnv("debug")   // FLUXBASE_DEBUG

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(authCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(completionCmd)
	rootCmd.AddCommand(functionsCmd)
	rootCmd.AddCommand(jobsCmd)
	rootCmd.AddCommand(storageCmd)
	rootCmd.AddCommand(chatbotsCmd)
	rootCmd.AddCommand(kbCmd)
	rootCmd.AddCommand(tablesCmd)
	rootCmd.AddCommand(rpcCmd)
	rootCmd.AddCommand(webhooksCmd)
	rootCmd.AddCommand(apikeysCmd)
	rootCmd.AddCommand(migrationsCmd)
	rootCmd.AddCommand(extensionsCmd)
	rootCmd.AddCommand(realtimeCmd)
	rootCmd.AddCommand(settingsCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(secretsCmd)
	rootCmd.AddCommand(syncCmd)
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		viper.AddConfigPath(cliconfig.DefaultConfigDir())
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	// Read config file (ignore error if not found)
	_ = viper.ReadInConfig()
}

// initializeClient sets up the API client for commands that need it
func initializeClient(cmd *cobra.Command, args []string) error {
	// Determine config path
	configPath := cfgFile
	if configPath == "" {
		configPath = cliconfig.DefaultConfigPath()
	}

	// Load config (use LoadOrCreate to allow env-var-only usage without a config file)
	var err error
	cfg, err = cliconfig.LoadOrCreate(configPath)
	if err != nil {
		return err
	}

	// Get profile name
	pName := profileName
	if pName == "" {
		pName = viper.GetString("profile")
	}
	if pName == "" {
		pName = cfg.CurrentProfile
	}

	// Try to get the profile, or create an empty one if env vars will provide credentials
	profile, err := cfg.GetProfile(pName)
	if err != nil {
		// If env vars provide server and token, we can work without a config file profile
		envServer := viper.GetString("server")
		envToken := viper.GetString("token")
		if envServer != "" && envToken != "" {
			profile = &cliconfig.Profile{
				Server:      envServer,
				Credentials: &cliconfig.Credentials{APIKey: envToken},
			}
		} else {
			return err
		}
	}

	// Override server from environment if set
	if envServer := viper.GetString("server"); envServer != "" {
		profile.Server = envServer
	}

	// Override token from environment if set
	if envToken := viper.GetString("token"); envToken != "" {
		if profile.Credentials == nil {
			profile.Credentials = &cliconfig.Credentials{}
		}
		profile.Credentials.APIKey = envToken
	}

	// Override debug from environment if set
	if viper.GetBool("debug") {
		debug = true
	}

	// Create API client
	apiClient = client.NewClient(cfg, profile,
		client.WithDebug(debug),
		client.WithConfigPath(configPath),
	)

	// Create formatter
	format, err := output.ParseFormat(outputFmt)
	if err != nil {
		return err
	}
	formatter = output.NewFormatter(format, noHeaders, quiet)

	return nil
}

// requireAuth wraps initializeClient for use in PreRunE
func requireAuth(cmd *cobra.Command, args []string) error {
	return initializeClient(cmd, args)
}

// GetFormatter returns the output formatter (for use by subcommands)
func GetFormatter() *output.Formatter {
	if formatter == nil {
		format, _ := output.ParseFormat(outputFmt)
		formatter = output.NewFormatter(format, noHeaders, quiet)
	}
	return formatter
}

// GetClient returns the API client (for use by subcommands)
func GetClient() *client.Client {
	return apiClient
}

// GetConfig returns the CLI config (for use by subcommands)
func GetConfig() *cliconfig.Config {
	return cfg
}

// GetConfigPath returns the config file path
func GetConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	return cliconfig.DefaultConfigPath()
}

// IsDebug returns true if debug mode is enabled
func IsDebug() bool {
	return debug
}
