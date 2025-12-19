package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	cliconfig "github.com/fluxbase-eu/fluxbase/cli/config"
	"github.com/fluxbase-eu/fluxbase/cli/util"
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authentication commands",
	Long:  `Manage authentication with Fluxbase servers.`,
}

var (
	loginServer   string
	loginEmail    string
	loginPassword string
	loginToken    string
	loginProfile  string
	useKeychain   bool
)

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with a Fluxbase server",
	Long: `Authenticate with a Fluxbase server using email/password or an API token.

Examples:
  # Interactive login (prompts for server, email, password)
  fluxbase auth login

  # Non-interactive login with email/password
  fluxbase auth login --server https://api.example.com --email user@example.com --password secret

  # Login with an API token
  fluxbase auth login --server https://api.example.com --token your-api-token

  # Save to a named profile
  fluxbase auth login --profile prod --server https://api.example.com`,
	RunE: runAuthLogin,
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and clear credentials",
	Long: `Clear stored credentials for the current or specified profile.

Examples:
  fluxbase auth logout
  fluxbase auth logout --profile prod`,
	RunE: runAuthLogout,
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current authentication status",
	Long:  `Display the current authentication status for all profiles.`,
	RunE:  runAuthStatus,
}

var authSwitchCmd = &cobra.Command{
	Use:   "switch [profile]",
	Short: "Switch to a different profile",
	Long: `Switch the active profile to a different one.

Examples:
  fluxbase auth switch prod
  fluxbase auth switch dev`,
	Args: cobra.ExactArgs(1),
	RunE: runAuthSwitch,
}

var authWhoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Display current user info",
	Long:  `Display information about the currently authenticated user.`,
	RunE:  runAuthWhoami,
}

func init() {
	// Login flags
	authLoginCmd.Flags().StringVar(&loginServer, "server", "", "Fluxbase server URL")
	authLoginCmd.Flags().StringVar(&loginEmail, "email", "", "Email address for login")
	authLoginCmd.Flags().StringVar(&loginPassword, "password", "", "Password for login")
	authLoginCmd.Flags().StringVar(&loginToken, "token", "", "API token for authentication")
	authLoginCmd.Flags().StringVar(&loginProfile, "profile", "default", "Profile name to save credentials")
	authLoginCmd.Flags().BoolVar(&useKeychain, "use-keychain", false, "Store credentials in system keychain")

	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authLogoutCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authSwitchCmd)
	authCmd.AddCommand(authWhoamiCmd)
}

func runAuthLogin(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	// Load or create config
	cfg, err := cliconfig.LoadOrCreate(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Interactive prompts if needed
	server := loginServer
	if server == "" {
		server, err = util.ReadLine("Fluxbase server URL: ")
		if err != nil {
			return err
		}
	}

	// Validate and normalize server URL
	server = strings.TrimSpace(server)
	if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
		server = "https://" + server
	}
	server = strings.TrimSuffix(server, "/")

	// Validate URL
	if _, err := url.Parse(server); err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}

	var creds *cliconfig.Credentials
	var userInfo *cliconfig.UserInfo

	if loginToken != "" {
		// Token-based authentication
		creds = &cliconfig.Credentials{
			APIKey: loginToken,
		}
		fmt.Println("Using API token for authentication")
	} else {
		// Email/password authentication
		email := loginEmail
		if email == "" {
			email, err = util.ReadLine("Email: ")
			if err != nil {
				return err
			}
		}

		password := loginPassword
		if password == "" {
			password, err = util.ReadPassword("Password: ")
			if err != nil {
				return err
			}
		}

		// Perform login
		creds, userInfo, err = performLogin(server, email, password)
		if err != nil {
			return fmt.Errorf("login failed: %w", err)
		}
	}

	// Create or update profile
	profile := &cliconfig.Profile{
		Name:   loginProfile,
		Server: server,
		User:   userInfo,
	}

	// Store credentials
	credManager := cliconfig.NewCredentialManager(cfg)
	if useKeychain {
		if !cliconfig.NewKeychainStore().IsAvailable() {
			fmt.Println("Warning: System keychain not available, storing in config file instead")
			useKeychain = false
		}
	}

	cfg.SetProfile(profile)

	if err := credManager.SaveCredentials(loginProfile, creds, useKeychain); err != nil {
		return fmt.Errorf("failed to save credentials: %w", err)
	}

	cfg.CurrentProfile = loginProfile

	// Save config
	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Successfully logged in to %s\n", server)
	if userInfo != nil {
		fmt.Printf("Logged in as: %s\n", userInfo.Email)
	}
	fmt.Printf("Profile saved as: %s\n", loginProfile)

	return nil
}

func performLogin(server, email, password string) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	loginURL := server + "/api/v1/auth/signin"

	body := map[string]string{
		"email":    email,
		"password": password,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL, strings.NewReader(string(jsonBody)))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fluxbase-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized {
		return nil, nil, fmt.Errorf("invalid email or password")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if errResp.Message != "" {
				return nil, nil, fmt.Errorf("%s", errResp.Message)
			}
			if errResp.Error != "" {
				return nil, nil, fmt.Errorf("%s", errResp.Error)
			}
		}
		return nil, nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var result struct {
		// Normal login response fields
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		User         struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			Role          string `json:"role"`
			EmailVerified bool   `json:"email_verified"`
		} `json:"user"`
		// 2FA response fields
		Requires2FA bool   `json:"requires_2fa"`
		UserID      string `json:"user_id"`
		Message     string `json:"message"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Check if 2FA is required
	if result.Requires2FA {
		return handle2FAVerification(server, result.UserID)
	}

	creds := &cliconfig.Credentials{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Unix() + result.ExpiresIn,
	}

	userInfo := &cliconfig.UserInfo{
		ID:            result.User.ID,
		Email:         result.User.Email,
		Role:          result.User.Role,
		EmailVerified: result.User.EmailVerified,
	}

	return creds, userInfo, nil
}

// handle2FAVerification prompts the user for a 2FA code and verifies it
func handle2FAVerification(serverURL, userID string) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	// Check if running interactively
	if !util.IsInteractive() {
		return nil, nil, fmt.Errorf("2FA required but running non-interactively. Use --token flag with an API key instead")
	}

	fmt.Println("Two-factor authentication required.")

	// Prompt for 2FA code
	code, err := util.ReadLine("Enter 2FA code (or backup code): ")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read 2FA code: %w", err)
	}
	code = strings.TrimSpace(code)

	if code == "" {
		return nil, nil, fmt.Errorf("2FA code cannot be empty")
	}

	// Call verify endpoint
	return verify2FACode(serverURL, userID, code)
}

// verify2FACode sends the 2FA code to the server for verification
func verify2FACode(serverURL, userID, code string) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	u, err := url.Parse(serverURL)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = "/api/v1/auth/2fa/verify"

	body := map[string]string{
		"user_id": userID,
		"code":    code,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "fluxbase-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("2FA verification request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest {
		return nil, nil, fmt.Errorf("invalid 2FA code")
	}

	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Message string `json:"message"`
			Error   string `json:"error"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err == nil {
			if errResp.Message != "" {
				return nil, nil, fmt.Errorf("%s", errResp.Message)
			}
			if errResp.Error != "" {
				return nil, nil, fmt.Errorf("%s", errResp.Error)
			}
		}
		return nil, nil, fmt.Errorf("2FA verification failed with status %d", resp.StatusCode)
	}

	// Parse response (same structure as normal login)
	var result struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int64  `json:"expires_in"`
		User         struct {
			ID            string `json:"id"`
			Email         string `json:"email"`
			Role          string `json:"role"`
			EmailVerified bool   `json:"email_verified"`
		} `json:"user"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse 2FA response: %w", err)
	}

	// Build credentials and user info
	creds := &cliconfig.Credentials{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Unix() + result.ExpiresIn,
	}

	userInfo := &cliconfig.UserInfo{
		ID:            result.User.ID,
		Email:         result.User.Email,
		Role:          result.User.Role,
		EmailVerified: result.User.EmailVerified,
	}

	return creds, userInfo, nil
}

func runAuthLogout(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	profileToLogout := profileName
	if profileToLogout == "" {
		profileToLogout = cfg.CurrentProfile
	}

	if profileToLogout == "" {
		return fmt.Errorf("no profile specified")
	}

	profile, err := cfg.GetProfile(profileToLogout)
	if err != nil {
		return err
	}

	// Clear credentials
	credManager := cliconfig.NewCredentialManager(cfg)
	if err := credManager.DeleteCredentials(profileToLogout); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	profile.Credentials = nil
	profile.User = nil

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Logged out from profile: %s\n", profileToLogout)
	return nil
}

func runAuthStatus(cmd *cobra.Command, args []string) error {
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		fmt.Println("Not logged in. Run 'fluxbase auth login' to authenticate.")
		return nil
	}

	formatter := GetFormatter()

	if len(cfg.Profiles) == 0 {
		fmt.Println("No profiles configured. Run 'fluxbase auth login' to authenticate.")
		return nil
	}

	fmt.Printf("Current profile: %s\n\n", cfg.CurrentProfile)

	for name, profile := range cfg.Profiles {
		current := ""
		if name == cfg.CurrentProfile {
			current = " (current)"
		}

		status := "not authenticated"
		if profile.HasCredentials() {
			if profile.IsTokenExpired() {
				status = "token expired"
			} else {
				status = "authenticated"
			}
		}

		user := "-"
		if profile.User != nil {
			user = profile.User.Email
		}

		if formatter.Format == "table" {
			fmt.Printf("Profile: %s%s\n", name, current)
			fmt.Printf("  Server: %s\n", profile.Server)
			fmt.Printf("  Status: %s\n", status)
			fmt.Printf("  User: %s\n", user)
			fmt.Printf("  Credential Store: %s\n", profile.CredentialStore)
			fmt.Println()
		}
	}

	return nil
}

func runAuthSwitch(cmd *cobra.Command, args []string) error {
	newProfile := args[0]
	configPath := GetConfigPath()

	cfg, err := cliconfig.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if _, err := cfg.GetProfile(newProfile); err != nil {
		return fmt.Errorf("profile '%s' not found", newProfile)
	}

	cfg.CurrentProfile = newProfile

	if err := cfg.Save(configPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Switched to profile: %s\n", newProfile)
	return nil
}

func runAuthWhoami(cmd *cobra.Command, args []string) error {
	if err := requireAuth(cmd, args); err != nil {
		return err
	}

	profile := apiClient.Profile
	formatter := GetFormatter()

	if profile.User == nil {
		fmt.Println("User information not available. Try logging in again.")
		return nil
	}

	if formatter.Format == "table" {
		fmt.Printf("Server: %s\n", profile.Server)
		fmt.Printf("Email: %s\n", profile.User.Email)
		fmt.Printf("ID: %s\n", profile.User.ID)
		fmt.Printf("Role: %s\n", profile.User.Role)
		fmt.Printf("Email Verified: %v\n", profile.User.EmailVerified)
	} else {
		formatter.Print(map[string]interface{}{
			"server":         profile.Server,
			"email":          profile.User.Email,
			"id":             profile.User.ID,
			"role":           profile.User.Role,
			"email_verified": profile.User.EmailVerified,
		})
	}

	return nil
}
