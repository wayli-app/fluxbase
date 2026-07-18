package cmd

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/nimbleflux/fluxbase/cli/client"
	cliconfig "github.com/nimbleflux/fluxbase/cli/config"
	"github.com/nimbleflux/fluxbase/cli/output"
)

// setupTestEnvWithHandler creates a test environment with a custom API handler.
// Returns the server, output buffer, and cleanup function.
func setupTestEnvWithHandler(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *bytes.Buffer, func()) {
	t.Helper()

	var buf bytes.Buffer
	server := httptest.NewServer(handler)

	cfg := cliconfig.New()
	profile := &cliconfig.Profile{
		Name:   "test",
		Server: server.URL,
		Credentials: &cliconfig.Credentials{
			APIKey: "test-token",
		},
	}
	testClient := client.NewClient(cfg, profile)

	apiClient = testClient
	formatter = output.NewFormatter(output.FormatJSON, false, false)
	formatter.Writer = &buf

	cleanup := func() {
		server.Close()
		apiClient = nil
		formatter = nil
	}

	return server, &buf, cleanup
}

// respondJSON writes a JSON response.
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// respondError writes a JSON error response.
func respondError(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{
		"message": message,
	})
}

// readRequestBody reads and unmarshals the request body.
func readRequestBody(t *testing.T, r *http.Request, target interface{}) {
	t.Helper()
	body, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, target))
}

// --- Flag reset functions ---

func resetTableFlags() {
	tableSchema = "public"
	tableSelect = "*"
	tableWhere = ""
	tableOrderBy = ""
	tableLimit = 100
	tableOffset = 0
	tableData = ""
	tableFile = ""
}

func resetFunctionFlags() {
	fnNamespace = ""
	fnCodeFile = ""
	fnDescription = ""
	fnTimeout = 0
	fnMemory = 0
	fnInvokeData = ""
	fnInvokeFile = ""
	fnAsync = false
	fnTail = 0
	fnFollow = false
	fnSyncDir = ""
	fnDryRun = false
	fnKeep = false
}

func resetJobFlags() {
	jobNamespace = ""
	jobPayload = ""
	jobPayloadFile = ""
	jobPriority = 0
	jobSchedule = ""
	jobSyncDir = ""
	jobDryRun = false
	jobKeep = false
}

func resetWebhookFlags() {
	whURL = ""
	whEvents = ""
	whSecret = ""
	whEnabled = false
}

func resetClientKeyFlags() {
	ckName = ""
	ckScopes = ""
	ckRateLimit = 0
	ckExpires = ""
}

func resetRPCFlags() {
	rpcNamespace = ""
	rpcParams = ""
	rpcFile = ""
	rpcAsync = false
	rpcSyncDir = ""
	rpcDryRun = false
	rpcKeep = false
}

func resetSettingsSecretsFlags() {
	settingsSecretUser = false
	settingsSecretDescription = ""
}

func resetSecretsFlags() {
	secretScope = ""
	secretNamespace = ""
	secretDescription = ""
	secretExpires = ""
}

func resetStorageFlags() {
	bucketPublic = false
	bucketMaxSize = 0
	objectPrefix = ""
	objectContentType = ""
	urlExpires = 0
}

func resetMigrationsFlags() {
	migNamespace = ""
	migUpSQL = ""
	migDownSQL = ""
	migSyncDir = ""
	migNoApply = false
	migDryRun = false
}

func resetExtensionsFlags() {
	extSchema = ""
}

func resetChatbotFlags() {
	cbSystemPrompt = ""
	cbModel = ""
	cbTemperature = 0
	cbMaxTokens = 0
	cbKnowledgeBase = ""
	cbSyncDir = ""
	cbNamespace = ""
	cbListNamespace = ""
	cbDryRun = false
	cbDeleteMissing = false
}

func resetServiceKeyFlags() {
	skName = ""
	skDescription = ""
	skScopes = ""
	skRateLimitPerMinute = 0
	skRateLimitPerHour = 0
	skExpires = ""
	skEnabled = false
	skRevokeReason = ""
	skGracePeriod = ""
}

func resetGraphQLFlags() {
	graphqlFile = ""
	graphqlVariables = nil
	graphqlPretty = false
	introspectTypesOnly = false
}

func resetLogsFlags() {
	logsCategory = ""
	logsCustomCategory = ""
	logsLevel = ""
	logsComponent = ""
	logsRequestID = ""
	logsUserID = ""
	logsSearch = ""
	logsSince = ""
	logsUntil = ""
	logsLimit = 0
	logsTail = 0
	logsFollow = false
	logsSortAsc = false
}

func resetAdminUsersFlags() {
	adminUserRole = ""
	adminUserEmail = ""
	adminUserForce = false
}

func resetAdminInvitationsFlags() {
	invIncludeAccepted = false
	invIncludeExpired = false
	invForce = false
}

func resetAdminSessionsFlags() {
	sessionForce = false
}

func resetUsersFlags() {
	appUserEmail = ""
	appUserForce = false
	usersSearchQuery = ""
}

func resetBranchFlags() {
	branchDataCloneMode = ""
	branchListType = ""
	branchCreateType = ""
	branchExpiresIn = ""
	branchParent = ""
	branchGitHubPR = 0
	branchGitHubRepo = ""
	branchSeedsDir = ""
	branchForce = false
}

func resetRealtimeFlags() {
	rtMessage = ""
	rtEvent = ""
}
