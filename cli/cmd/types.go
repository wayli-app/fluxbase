package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var typesCmd = &cobra.Command{
	Use:   "types",
	Short: "Generate type definitions from database schema",
	Long:  `Generate TypeScript or other type definitions from your Fluxbase database schema.`,
}

var (
	typesSchemas          []string
	typesIncludeFunctions bool
	typesIncludeViews     bool
	typesOutput           string
	typesFormat           string
)

var typesGenerateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate TypeScript types from schema",
	Long: `Generate TypeScript type definitions from your database schema.

This command fetches your database schema and generates TypeScript interfaces
for all tables, views, and optionally RPC functions. The generated types can
be used with the Fluxbase TypeScript SDK for type-safe database queries.

Examples:
  # Generate types for the public schema and write to types.ts
  fluxbase types generate --output types.ts

  # Generate types for multiple schemas
  fluxbase types generate --schemas public,auth --output types.ts

  # Generate types including RPC function signatures
  fluxbase types generate --include-functions --output types.ts

  # Output to stdout (for piping)
  fluxbase types generate

  # Generate types without views
  fluxbase types generate --include-views=false --output types.ts`,
	PreRunE: requireAuth,
	RunE:    runTypesGenerate,
}

var typesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available schemas",
	Long: `List all available database schemas that can be used for type generation.

Examples:
  fluxbase types list`,
	PreRunE: requireAuth,
	RunE:    runTypesList,
}

func init() {
	// Generate flags
	typesGenerateCmd.Flags().StringSliceVar(&typesSchemas, "schemas", []string{"public"}, "Schemas to include in type generation")
	typesGenerateCmd.Flags().BoolVar(&typesIncludeFunctions, "include-functions", true, "Include RPC function types")
	typesGenerateCmd.Flags().BoolVar(&typesIncludeViews, "include-views", true, "Include view types")
	typesGenerateCmd.Flags().StringVarP(&typesOutput, "output", "o", "", "Output file path (default: stdout)")
	typesGenerateCmd.Flags().StringVar(&typesFormat, "format", "types", "Output format: 'types' (interfaces only) or 'full' (with helpers)")

	typesCmd.AddCommand(typesGenerateCmd)
	typesCmd.AddCommand(typesListCmd)
}

func runTypesGenerate(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Build request body
	reqBody := map[string]interface{}{
		"schemas":           typesSchemas,
		"include_functions": typesIncludeFunctions,
		"include_views":     typesIncludeViews,
		"format":            typesFormat,
	}

	// Make request
	var result map[string]interface{}

	if err := apiClient.DoPost(ctx, "/api/v1/admin/schema/typescript", reqBody, &result); err != nil {
		return fmt.Errorf("failed to generate types: %w", err)
	}

	// Extract TypeScript content
	typescript, ok := result["typescript"].(string)
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	// Output
	if typesOutput != "" {
		// Write to file
		if err := os.WriteFile(typesOutput, []byte(typescript), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Printf("TypeScript types written to %s\n", typesOutput)

		// Show schemas included
		if schemas, ok := result["schemas"].([]interface{}); ok {
			schemaStrs := make([]string, len(schemas))
			for i, s := range schemas {
				schemaStrs[i] = fmt.Sprintf("%v", s)
			}
			fmt.Printf("Schemas included: %s\n", strings.Join(schemaStrs, ", "))
		}
	} else {
		// Output to stdout
		fmt.Print(typescript)
	}

	return nil
}

func runTypesList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var schemas []string
	if err := apiClient.DoGet(ctx, "/api/v1/admin/schemas", url.Values{}, &schemas); err != nil {
		return fmt.Errorf("failed to list schemas: %w", err)
	}

	if len(schemas) == 0 {
		fmt.Println("No schemas found.")
		return nil
	}

	formatter := GetFormatter()

	if jsonOutput, _ := cmd.Flags().GetBool("json"); jsonOutput {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(schemas)
	}

	fmt.Println("Available schemas:")
	for _, schema := range schemas {
		fmt.Printf("  - %s\n", schema)
	}

	// Also show type generation command hint
	fmt.Println("\nGenerate types with:")
	fmt.Printf("  fluxbase types generate --schemas %s --output types.ts\n", strings.Join(schemas, ","))

	_ = formatter // Suppress unused warning
	return nil
}
