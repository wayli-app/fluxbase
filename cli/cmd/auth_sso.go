package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"

	cliconfig "github.com/fluxbase-eu/fluxbase/cli/config"
	"github.com/fluxbase-eu/fluxbase/cli/util"
)

// SSOProvider represents an SSO provider for dashboard login
type SSOProvider struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`     // "oauth" or "saml"
	Provider string `json:"provider"` // For OAuth: google, github, etc.
}

// SSOProvidersResponse is the response from the SSO providers endpoint
type SSOProvidersResponse struct {
	Providers             []SSOProvider `json:"providers"`
	PasswordLoginDisabled bool          `json:"password_login_disabled"`
}

// getSSOProviders fetches the available SSO providers from the server
func getSSOProviders(serverURL string) (*SSOProvidersResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	u := fmt.Sprintf("%s/dashboard/auth/sso/providers", serverURL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fluxbase-cli/1.0")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch SSO providers: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var result SSOProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse SSO providers response: %w", err)
	}

	return &result, nil
}

// performSSOLogin performs SSO login via browser-based flow
func performSSOLogin(serverURL string, provider *SSOProvider) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	// Check if running interactively
	if !util.IsInteractive() {
		return nil, nil, fmt.Errorf("SSO login requires an interactive terminal. Use --token flag with an API key instead")
	}

	// Find an available port for the callback server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start callback server: %w", err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	callbackURL := fmt.Sprintf("http://localhost:%d/callback", port)

	// Build the SSO login URL
	var loginURL string
	if provider.Type == "oauth" {
		loginURL = fmt.Sprintf("%s/dashboard/auth/sso/oauth/%s?redirect_to=%s",
			serverURL, provider.ID, url.QueryEscape(callbackURL))
	} else if provider.Type == "saml" {
		loginURL = fmt.Sprintf("%s/dashboard/auth/sso/saml/%s?redirect_to=%s",
			serverURL, provider.ID, url.QueryEscape(callbackURL))
	} else {
		return nil, nil, fmt.Errorf("unknown SSO provider type: %s", provider.Type)
	}

	// Create a channel to receive the result
	resultCh := make(chan *ssoCallbackResult, 1)
	errCh := make(chan error, 1)

	// Start the callback server
	server := &http.Server{
		Addr: fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleSSOCallback(w, r, resultCh, errCh)
		}),
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("callback server error: %w", err)
		}
	}()

	// Open the browser
	fmt.Printf("Opening browser for SSO login with %s...\n", provider.Name)
	fmt.Printf("If the browser doesn't open, visit: %s\n", loginURL)

	if err := openBrowser(loginURL); err != nil {
		fmt.Printf("Warning: Failed to open browser automatically: %v\n", err)
	}

	fmt.Println("Waiting for SSO callback...")

	// Wait for the callback with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	select {
	case result := <-resultCh:
		server.Shutdown(context.Background())
		return buildCredsFromSSOResult(result)
	case err := <-errCh:
		server.Shutdown(context.Background())
		return nil, nil, err
	case <-ctx.Done():
		server.Shutdown(context.Background())
		return nil, nil, fmt.Errorf("SSO login timed out")
	}
}

type ssoCallbackResult struct {
	AccessToken  string
	RefreshToken string
	Error        string
}

func handleSSOCallback(w http.ResponseWriter, r *http.Request, resultCh chan *ssoCallbackResult, errCh chan error) {
	accessToken := r.URL.Query().Get("access_token")
	refreshToken := r.URL.Query().Get("refresh_token")
	errorParam := r.URL.Query().Get("error")

	if errorParam != "" {
		errCh <- fmt.Errorf("SSO error: %s", errorParam)
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>SSO Failed</title></head>
<body style="font-family: system-ui; text-align: center; padding: 40px;">
<h1>SSO Login Failed</h1>
<p>Error: %s</p>
<p>You can close this window.</p>
</body>
</html>`, errorParam)
		return
	}

	if accessToken == "" {
		errCh <- fmt.Errorf("no access token received")
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>SSO Failed</title></head>
<body style="font-family: system-ui; text-align: center; padding: 40px;">
<h1>SSO Login Failed</h1>
<p>No access token received.</p>
<p>You can close this window.</p>
</body>
</html>`)
		return
	}

	resultCh <- &ssoCallbackResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!DOCTYPE html>
<html>
<head><title>SSO Success</title></head>
<body style="font-family: system-ui; text-align: center; padding: 40px;">
<h1>SSO Login Successful!</h1>
<p>You can close this window and return to the CLI.</p>
<script>window.close();</script>
</body>
</html>`)
}

func buildCredsFromSSOResult(result *ssoCallbackResult) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	// For now, we just have the tokens. User info can be fetched later if needed.
	creds := &cliconfig.Credentials{
		AccessToken:  result.AccessToken,
		RefreshToken: result.RefreshToken,
		ExpiresAt:    time.Now().Unix() + 86400, // Assume 24 hours
	}

	return creds, nil, nil
}

// openBrowser opens the specified URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}

// selectSSOProvider prompts the user to select an SSO provider
func selectSSOProvider(providers []SSOProvider) (*SSOProvider, error) {
	if len(providers) == 0 {
		return nil, fmt.Errorf("no SSO providers available")
	}

	if len(providers) == 1 {
		return &providers[0], nil
	}

	fmt.Println("\nAvailable SSO providers:")
	for i, p := range providers {
		fmt.Printf("  [%d] %s (%s)\n", i+1, p.Name, p.Type)
	}

	input, err := util.ReadLine(fmt.Sprintf("\nSelect provider (1-%d): ", len(providers)))
	if err != nil {
		return nil, err
	}

	var index int
	if _, err := fmt.Sscanf(strings.TrimSpace(input), "%d", &index); err != nil || index < 1 || index > len(providers) {
		return nil, fmt.Errorf("invalid selection")
	}

	return &providers[index-1], nil
}

// performSSOAuthentication handles the full SSO authentication flow
func performSSOAuthentication(serverURL string) (*cliconfig.Credentials, *cliconfig.UserInfo, error) {
	// Fetch available SSO providers
	ssoInfo, err := getSSOProviders(serverURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to fetch SSO providers: %w", err)
	}

	if len(ssoInfo.Providers) == 0 {
		return nil, nil, fmt.Errorf("no SSO providers configured on this server")
	}

	// Select provider
	provider, err := selectSSOProvider(ssoInfo.Providers)
	if err != nil {
		return nil, nil, err
	}

	// Perform SSO login
	return performSSOLogin(serverURL, provider)
}
