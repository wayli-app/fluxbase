package cmd

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var servicekeysCmd = &cobra.Command{
	Use:     "servicekeys",
	Aliases: []string{"servicekey", "sk"},
	Short:   "Manage service keys",
	Long:    `Create, update, and manage service keys for API access.`,
}

var (
	skName               string
	skDescription        string
	skScopes             string
	skRateLimitPerMinute int
	skRateLimitPerHour   int
	skExpires            string
	skEnabled            bool
	skRevokeReason       string
	skGracePeriod        string
)

var servicekeysListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all service keys",
	Long: `List all service keys.

Examples:
  fluxbase servicekeys list`,
	PreRunE: requireAuth,
	RunE:    runServiceKeysList,
}

var servicekeysCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new service key",
	Long: `Create a new service key.

Examples:
  fluxbase servicekeys create --name "Migrations Key" --scopes "migrations:*"
  fluxbase servicekeys create --name "Production" --rate-limit-per-hour 100`,
	PreRunE: requireAuth,
	RunE:    runServiceKeysCreate,
}

var servicekeysGetCmd = &cobra.Command{
	Use:   "get [id]",
	Short: "Get service key details",
	Long: `Get details of a specific service key.

Examples:
  fluxbase servicekeys get abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysGet,
}

var servicekeysUpdateCmd = &cobra.Command{
	Use:   "update [id]",
	Short: "Update a service key",
	Long: `Update a service key's properties.

Examples:
  fluxbase servicekeys update abc123 --name "New Name"
  fluxbase servicekeys update abc123 --rate-limit-per-hour 200
  fluxbase servicekeys update abc123 --enabled=false`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysUpdate,
}

var servicekeysDisableCmd = &cobra.Command{
	Use:   "disable [id]",
	Short: "Disable a service key",
	Long: `Disable a service key (keeps the record but prevents use).

Examples:
  fluxbase servicekeys disable abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysDisable,
}

var servicekeysEnableCmd = &cobra.Command{
	Use:   "enable [id]",
	Short: "Enable a service key",
	Long: `Enable a previously disabled service key.

Examples:
  fluxbase servicekeys enable abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysEnable,
}

var servicekeysDeleteCmd = &cobra.Command{
	Use:     "delete [id]",
	Aliases: []string{"rm", "remove"},
	Short:   "Delete a service key",
	Long: `Delete a service key permanently.

Examples:
  fluxbase servicekeys delete abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysDelete,
}

var servicekeysRevokeCmd = &cobra.Command{
	Use:   "revoke [id]",
	Short: "Emergency revoke a service key",
	Long: `Emergency revoke a service key immediately.

This action is irreversible. The key will be permanently disabled and marked
as revoked with an audit trail. Use this for security incidents.

Examples:
  fluxbase servicekeys revoke abc123 --reason "Key compromised"
  fluxbase servicekeys revoke abc123 --reason "Employee departure"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysRevoke,
}

var servicekeysDeprecateCmd = &cobra.Command{
	Use:   "deprecate [id]",
	Short: "Deprecate a service key for rotation",
	Long: `Mark a service key as deprecated with a grace period.

The key continues working during the grace period, allowing time for
applications to migrate to a new key. After the grace period, the key
will stop working.

Examples:
  fluxbase servicekeys deprecate abc123 --grace-period 24h
  fluxbase servicekeys deprecate abc123 --grace-period 7d --reason "Scheduled rotation"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysDeprecate,
}

var servicekeysRotateCmd = &cobra.Command{
	Use:   "rotate [id]",
	Short: "Rotate a service key",
	Long: `Create a new service key as a replacement for an existing one.

This deprecates the old key with a grace period and creates a new key
with the same configuration. The old key is marked with a reference to
the new key.

Examples:
  fluxbase servicekeys rotate abc123 --grace-period 24h
  fluxbase servicekeys rotate abc123 --grace-period 7d`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysRotate,
}

var servicekeysRevocationsCmd = &cobra.Command{
	Use:   "revocations [id]",
	Short: "View revocation history for a service key",
	Long: `View the revocation audit log for a service key.

Shows all revocation events including emergency revocations, rotations,
and expirations.

Examples:
  fluxbase servicekeys revocations abc123`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runServiceKeysRevocations,
}

func init() {
	// Create flags
	servicekeysCreateCmd.Flags().StringVar(&skName, "name", "", "Service key name (required)")
	servicekeysCreateCmd.Flags().StringVar(&skDescription, "description", "", "Service key description")
	servicekeysCreateCmd.Flags().StringVar(&skScopes, "scopes", "", "Comma-separated scopes (default: *)")
	servicekeysCreateCmd.Flags().IntVar(&skRateLimitPerMinute, "rate-limit-per-minute", 0, "Requests per minute (0 = no limit)")
	servicekeysCreateCmd.Flags().IntVar(&skRateLimitPerHour, "rate-limit-per-hour", 0, "Requests per hour (0 = no limit)")
	servicekeysCreateCmd.Flags().StringVar(&skExpires, "expires", "", "Expiration time (e.g., 2025-12-31T23:59:59Z)")
	_ = servicekeysCreateCmd.MarkFlagRequired("name")

	// Update flags
	servicekeysUpdateCmd.Flags().StringVar(&skName, "name", "", "New service key name")
	servicekeysUpdateCmd.Flags().StringVar(&skDescription, "description", "", "New service key description")
	servicekeysUpdateCmd.Flags().StringVar(&skScopes, "scopes", "", "New comma-separated scopes")
	servicekeysUpdateCmd.Flags().IntVar(&skRateLimitPerMinute, "rate-limit-per-minute", 0, "Requests per minute (0 = no limit)")
	servicekeysUpdateCmd.Flags().IntVar(&skRateLimitPerHour, "rate-limit-per-hour", 0, "Requests per hour (0 = no limit)")
	servicekeysUpdateCmd.Flags().BoolVar(&skEnabled, "enabled", true, "Enable or disable the key")

	// Revoke flags
	servicekeysRevokeCmd.Flags().StringVar(&skRevokeReason, "reason", "", "Reason for revocation (required)")
	_ = servicekeysRevokeCmd.MarkFlagRequired("reason")

	// Deprecate flags
	servicekeysDeprecateCmd.Flags().StringVar(&skGracePeriod, "grace-period", "24h", "Grace period before key stops working (e.g., 24h, 7d)")
	servicekeysDeprecateCmd.Flags().StringVar(&skRevokeReason, "reason", "", "Reason for deprecation")

	// Rotate flags
	servicekeysRotateCmd.Flags().StringVar(&skGracePeriod, "grace-period", "24h", "Grace period for old key (e.g., 24h, 7d)")

	servicekeysCmd.AddCommand(servicekeysListCmd)
	servicekeysCmd.AddCommand(servicekeysCreateCmd)
	servicekeysCmd.AddCommand(servicekeysGetCmd)
	servicekeysCmd.AddCommand(servicekeysUpdateCmd)
	servicekeysCmd.AddCommand(servicekeysDisableCmd)
	servicekeysCmd.AddCommand(servicekeysEnableCmd)
	servicekeysCmd.AddCommand(servicekeysDeleteCmd)
	servicekeysCmd.AddCommand(servicekeysRevokeCmd)
	servicekeysCmd.AddCommand(servicekeysDeprecateCmd)
	servicekeysCmd.AddCommand(servicekeysRotateCmd)
	servicekeysCmd.AddCommand(servicekeysRevocationsCmd)
}

func runServiceKeysList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var keys []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/service-keys", nil, &keys); err != nil {
		return err
	}

	if len(keys) == 0 {
		fmt.Println("No service keys found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"ID", "NAME", "PREFIX", "SCOPES", "STATUS", "RATE LIMIT", "CREATED"},
			Rows:    make([][]string, len(keys)),
		}

		for i, key := range keys {
			id := getStringValue(key, "id")
			name := getStringValue(key, "name")
			prefix := getStringValue(key, "key_prefix")
			scopes := formatScopes(key["scopes"])
			status := getServiceKeyStatus(key)
			rateLimit := formatRateLimit(key)
			created := formatDate(getStringValue(key, "created_at"))

			data.Rows[i] = []string{id, name, prefix, scopes, status, rateLimit, created}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(keys); err != nil {
			return err
		}
	}

	return nil
}

// getServiceKeyStatus returns the status string considering revocation/deprecation
func getServiceKeyStatus(key map[string]interface{}) string {
	// Check if revoked
	if revokedAt := getStringValue(key, "revoked_at"); revokedAt != "" {
		return "revoked"
	}

	// Check if deprecated (within grace period)
	if deprecatedAt := getStringValue(key, "deprecated_at"); deprecatedAt != "" {
		gracePeriodEnds := getStringValue(key, "grace_period_ends_at")
		if gracePeriodEnds != "" {
			t, err := time.Parse(time.RFC3339, gracePeriodEnds)
			if err == nil && time.Now().Before(t) {
				return "deprecated"
			}
		}
		return "expired"
	}

	// Check if expired
	if expiresAt := getStringValue(key, "expires_at"); expiresAt != "" {
		t, err := time.Parse(time.RFC3339, expiresAt)
		if err == nil && time.Now().After(t) {
			return "expired"
		}
	}

	// Check if disabled
	if key["enabled"] != true {
		return "disabled"
	}

	return "active"
}

func runServiceKeysCreate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"name": skName,
	}

	if skDescription != "" {
		body["description"] = skDescription
	}

	if skScopes != "" {
		scopeList := strings.Split(skScopes, ",")
		for i, s := range scopeList {
			scopeList[i] = strings.TrimSpace(s)
		}
		body["scopes"] = scopeList
	}

	if skRateLimitPerMinute > 0 {
		body["rate_limit_per_minute"] = skRateLimitPerMinute
	}

	if skRateLimitPerHour > 0 {
		body["rate_limit_per_hour"] = skRateLimitPerHour
	}

	if skExpires != "" {
		body["expires_at"] = skExpires
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys", body, &result); err != nil {
		return err
	}

	id := getStringValue(result, "id")
	key := getStringValue(result, "key")
	prefix := getStringValue(result, "key_prefix")

	fmt.Printf("Service key created:\n")
	fmt.Printf("  ID:     %s\n", id)
	fmt.Printf("  Name:   %s\n", skName)
	fmt.Printf("  Prefix: %s\n", prefix)
	fmt.Printf("  Key:    %s\n", key)
	fmt.Println()
	fmt.Println("IMPORTANT: Save this key now. You won't be able to see it again!")

	return nil
}

func runServiceKeysGet(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var key map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id), nil, &key); err != nil {
		return err
	}

	formatter := GetFormatter()
	return formatter.Print(key)
}

func runServiceKeysUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{}

	if cmd.Flags().Changed("name") {
		body["name"] = skName
	}

	if cmd.Flags().Changed("description") {
		body["description"] = skDescription
	}

	if cmd.Flags().Changed("scopes") {
		scopeList := strings.Split(skScopes, ",")
		for i, s := range scopeList {
			scopeList[i] = strings.TrimSpace(s)
		}
		body["scopes"] = scopeList
	}

	if cmd.Flags().Changed("rate-limit-per-minute") {
		body["rate_limit_per_minute"] = skRateLimitPerMinute
	}

	if cmd.Flags().Changed("rate-limit-per-hour") {
		body["rate_limit_per_hour"] = skRateLimitPerHour
	}

	if cmd.Flags().Changed("enabled") {
		body["enabled"] = skEnabled
	}

	if len(body) == 0 {
		return fmt.Errorf("no fields to update")
	}

	if err := apiClient.DoPatch(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id), body, nil); err != nil {
		return err
	}

	fmt.Printf("Service key '%s' updated.\n", id)
	return nil
}

func runServiceKeysDisable(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/disable", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Service key '%s' disabled.\n", id)
	return nil
}

func runServiceKeysEnable(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/enable", nil, nil); err != nil {
		return err
	}

	fmt.Printf("Service key '%s' enabled.\n", id)
	return nil
}

func runServiceKeysDelete(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := apiClient.DoDelete(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)); err != nil {
		return err
	}

	fmt.Printf("Service key '%s' deleted.\n", id)
	return nil
}

func runServiceKeysRevoke(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"reason": skRevokeReason,
	}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/revoke", body, nil); err != nil {
		return err
	}

	fmt.Printf("Service key '%s' has been revoked.\n", id)
	fmt.Printf("Reason: %s\n", skRevokeReason)
	fmt.Println("\nThis action is permanent. The key can no longer be used.")
	return nil
}

func runServiceKeysDeprecate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"grace_period": skGracePeriod,
	}

	if skRevokeReason != "" {
		body["reason"] = skRevokeReason
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/deprecate", body, &result); err != nil {
		return err
	}

	gracePeriodEnds := getStringValue(result, "grace_period_ends_at")

	fmt.Printf("Service key '%s' has been deprecated.\n", id)
	fmt.Printf("Grace period: %s\n", skGracePeriod)
	if gracePeriodEnds != "" {
		fmt.Printf("Key will stop working at: %s\n", gracePeriodEnds)
	}
	fmt.Println("\nThe key continues to work during the grace period.")
	return nil
}

func runServiceKeysRotate(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	body := map[string]interface{}{
		"grace_period": skGracePeriod,
	}

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/rotate", body, &result); err != nil {
		return err
	}

	newKeyID := getStringValue(result, "id")
	newKey := getStringValue(result, "key")
	newKeyPrefix := getStringValue(result, "key_prefix")
	gracePeriodEnds := getStringValue(result, "grace_period_ends_at")

	fmt.Println("Service key rotated successfully!")
	fmt.Println()
	fmt.Println("New key created:")
	fmt.Printf("  ID:     %s\n", newKeyID)
	fmt.Printf("  Prefix: %s\n", newKeyPrefix)
	fmt.Printf("  Key:    %s\n", newKey)
	fmt.Println()
	fmt.Println("IMPORTANT: Save this key now. You won't be able to see it again!")
	fmt.Println()
	fmt.Printf("Old key '%s' has been deprecated.\n", id)
	fmt.Printf("Grace period: %s\n", skGracePeriod)
	if gracePeriodEnds != "" {
		fmt.Printf("Old key will stop working at: %s\n", gracePeriodEnds)
	}

	return nil
}

func runServiceKeysRevocations(cmd *cobra.Command, args []string) error {
	id := args[0]

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var revocations []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/service-keys/"+url.PathEscape(id)+"/revocations", nil, &revocations); err != nil {
		return err
	}

	if len(revocations) == 0 {
		fmt.Printf("No revocation history found for service key '%s'.\n", id)
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"TYPE", "REASON", "REVOKED BY", "TIMESTAMP"},
			Rows:    make([][]string, len(revocations)),
		}

		for i, rev := range revocations {
			revType := getStringValue(rev, "revocation_type")
			reason := getStringValue(rev, "reason")
			revokedBy := getStringValue(rev, "revoked_by")
			timestamp := formatDate(getStringValue(rev, "created_at"))

			// Truncate reason if too long
			if len(reason) > 40 {
				reason = reason[:40] + "..."
			}

			data.Rows[i] = []string{revType, reason, revokedBy, timestamp}
		}

		formatter.PrintTable(data)
	} else {
		if err := formatter.Print(revocations); err != nil {
			return err
		}
	}

	return nil
}

// formatScopes formats the scopes array for display
func formatScopes(scopes interface{}) string {
	if scopes == nil {
		return "*"
	}
	switch v := scopes.(type) {
	case []interface{}:
		strs := make([]string, len(v))
		for i, s := range v {
			strs[i] = fmt.Sprintf("%v", s)
		}
		result := strings.Join(strs, ",")
		if len(result) > 20 {
			return result[:20] + "..."
		}
		return result
	case string:
		if len(v) > 20 {
			return v[:20] + "..."
		}
		return v
	default:
		return "*"
	}
}

// formatRateLimit formats the rate limit for display
func formatRateLimit(key map[string]interface{}) string {
	perMin := key["rate_limit_per_minute"]
	perHour := key["rate_limit_per_hour"]

	if perMin == nil && perHour == nil {
		return "unlimited"
	}

	var parts []string
	if perMin != nil {
		parts = append(parts, fmt.Sprintf("%v/min", perMin))
	}
	if perHour != nil {
		parts = append(parts, fmt.Sprintf("%v/hr", perHour))
	}

	return strings.Join(parts, ", ")
}

// formatDate formats a date string for display
func formatDate(date string) string {
	if date == "" {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, date)
	if err != nil {
		return date
	}
	return t.Format("2006-01-02")
}
