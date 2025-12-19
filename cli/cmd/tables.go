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

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var tablesCmd = &cobra.Command{
	Use:     "tables",
	Aliases: []string{"table", "db"},
	Short:   "Query and manage database tables",
	Long:    `Query, insert, update, and delete records in database tables.`,
}

var (
	tableSchema  string
	tableSelect  string
	tableWhere   string
	tableOrderBy string
	tableLimit   int
	tableOffset  int
	tableData    string
	tableFile    string
)

var tablesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tables",
	Long: `List all tables in the database.

Examples:
  fluxbase tables list
  fluxbase tables list --schema public`,
	PreRunE: requireAuth,
	RunE:    runTablesList,
}

var tablesDescribeCmd = &cobra.Command{
	Use:   "describe [table]",
	Short: "Describe table structure",
	Long: `Show the structure of a table.

Examples:
  fluxbase tables describe users
  fluxbase tables describe public.users`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runTablesDescribe,
}

var tablesQueryCmd = &cobra.Command{
	Use:   "query [table]",
	Short: "Query table records",
	Long: `Query records from a table.

Examples:
  fluxbase tables query users
  fluxbase tables query users --select "id,email,created_at"
  fluxbase tables query users --where "role=eq.admin"
  fluxbase tables query orders --order-by "created_at.desc" --limit 10`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runTablesQuery,
}

var tablesInsertCmd = &cobra.Command{
	Use:   "insert [table]",
	Short: "Insert a record",
	Long: `Insert a new record into a table.

Examples:
  fluxbase tables insert users --data '{"email": "user@example.com", "name": "John"}'
  fluxbase tables insert users --file ./user.json`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runTablesInsert,
}

var tablesUpdateCmd = &cobra.Command{
	Use:   "update [table]",
	Short: "Update records",
	Long: `Update records in a table.

Examples:
  fluxbase tables update users --where "id=eq.123" --data '{"name": "Jane"}'`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runTablesUpdate,
}

var tablesDeleteCmd = &cobra.Command{
	Use:   "delete [table]",
	Short: "Delete records",
	Long: `Delete records from a table.

Examples:
  fluxbase tables delete users --where "id=eq.123"`,
	Args:    cobra.ExactArgs(1),
	PreRunE: requireAuth,
	RunE:    runTablesDelete,
}

func init() {
	// List flags
	tablesListCmd.Flags().StringVar(&tableSchema, "schema", "public", "Schema to list tables from")

	// Query flags
	tablesQueryCmd.Flags().StringVar(&tableSelect, "select", "*", "Columns to select")
	tablesQueryCmd.Flags().StringVar(&tableWhere, "where", "", "Filter conditions (PostgREST format)")
	tablesQueryCmd.Flags().StringVar(&tableOrderBy, "order-by", "", "Order by column")
	tablesQueryCmd.Flags().IntVar(&tableLimit, "limit", 100, "Maximum records to return")
	tablesQueryCmd.Flags().IntVar(&tableOffset, "offset", 0, "Offset for pagination")

	// Insert flags
	tablesInsertCmd.Flags().StringVar(&tableData, "data", "", "JSON data to insert")
	tablesInsertCmd.Flags().StringVar(&tableFile, "file", "", "File containing JSON data")

	// Update flags
	tablesUpdateCmd.Flags().StringVar(&tableWhere, "where", "", "Filter conditions (required)")
	tablesUpdateCmd.Flags().StringVar(&tableData, "data", "", "JSON data to update")
	tablesUpdateCmd.Flags().StringVar(&tableFile, "file", "", "File containing JSON data")
	_ = tablesUpdateCmd.MarkFlagRequired("where")

	// Delete flags
	tablesDeleteCmd.Flags().StringVar(&tableWhere, "where", "", "Filter conditions (required)")
	_ = tablesDeleteCmd.MarkFlagRequired("where")

	tablesCmd.AddCommand(tablesListCmd)
	tablesCmd.AddCommand(tablesDescribeCmd)
	tablesCmd.AddCommand(tablesQueryCmd)
	tablesCmd.AddCommand(tablesInsertCmd)
	tablesCmd.AddCommand(tablesUpdateCmd)
	tablesCmd.AddCommand(tablesDeleteCmd)
}

func runTablesList(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if tableSchema != "" {
		query.Set("schema", tableSchema)
	}

	var tables []map[string]interface{}
	if err := apiClient.DoGet(ctx, "/api/v1/admin/tables", query, &tables); err != nil {
		return err
	}

	if len(tables) == 0 {
		fmt.Println("No tables found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"SCHEMA", "TABLE", "TYPE"},
			Rows:    make([][]string, len(tables)),
		}

		for i, table := range tables {
			schema := getStringValue(table, "schema")
			name := getStringValue(table, "name")
			tableType := getStringValue(table, "type")

			data.Rows[i] = []string{schema, name, tableType}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(tables)
	}

	return nil
}

func runTablesDescribe(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/v1/admin/tables/%s/%s/columns", url.PathEscape(schema), url.PathEscape(table))

	var columns []map[string]interface{}
	if err := apiClient.DoGet(ctx, path, nil, &columns); err != nil {
		return err
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		data := output.TableData{
			Headers: []string{"COLUMN", "TYPE", "NULLABLE", "DEFAULT"},
			Rows:    make([][]string, len(columns)),
		}

		for i, col := range columns {
			name := getStringValue(col, "name")
			dataType := getStringValue(col, "data_type")
			nullable := fmt.Sprintf("%v", col["is_nullable"])
			defaultVal := getStringValue(col, "default")
			if defaultVal == "" {
				defaultVal = "-"
			}

			data.Rows[i] = []string{name, dataType, nullable, defaultVal}
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(columns)
	}

	return nil
}

func runTablesQuery(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	if tableSelect != "" && tableSelect != "*" {
		query.Set("select", tableSelect)
	}
	if tableWhere != "" {
		// Parse where conditions
		conditions := strings.Split(tableWhere, ",")
		for _, cond := range conditions {
			parts := strings.SplitN(cond, "=", 2)
			if len(parts) == 2 {
				query.Set(parts[0], parts[1])
			}
		}
	}
	if tableOrderBy != "" {
		query.Set("order", tableOrderBy)
	}
	if tableLimit > 0 {
		query.Set("limit", fmt.Sprintf("%d", tableLimit))
	}
	if tableOffset > 0 {
		query.Set("offset", fmt.Sprintf("%d", tableOffset))
	}

	path := fmt.Sprintf("/api/v1/tables/%s/%s", url.PathEscape(schema), url.PathEscape(table))

	var records []map[string]interface{}
	if err := apiClient.DoGet(ctx, path, query, &records); err != nil {
		return err
	}

	if len(records) == 0 {
		fmt.Println("No records found.")
		return nil
	}

	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		// Build headers from first record
		var headers []string
		for key := range records[0] {
			headers = append(headers, key)
		}

		data := output.TableData{
			Headers: headers,
			Rows:    make([][]string, len(records)),
		}

		for i, record := range records {
			row := make([]string, len(headers))
			for j, header := range headers {
				if val, ok := record[header]; ok {
					row[j] = fmt.Sprintf("%v", val)
				}
			}
			data.Rows[i] = row
		}

		formatter.PrintTable(data)
	} else {
		formatter.Print(records)
	}

	return nil
}

func runTablesInsert(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	// Get data
	var data interface{}
	if tableFile != "" {
		content, err := os.ReadFile(tableFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("invalid JSON in file: %w", err)
		}
	} else if tableData != "" {
		if err := json.Unmarshal([]byte(tableData), &data); err != nil {
			return fmt.Errorf("invalid JSON data: %w", err)
		}
	} else {
		return fmt.Errorf("either --data or --file is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	path := fmt.Sprintf("/api/v1/tables/%s/%s", url.PathEscape(schema), url.PathEscape(table))

	var result map[string]interface{}
	if err := apiClient.DoPost(ctx, path, data, &result); err != nil {
		return err
	}

	fmt.Println("Record inserted successfully.")
	formatter := GetFormatter()
	return formatter.Print(result)
}

func runTablesUpdate(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	// Get data
	var data interface{}
	if tableFile != "" {
		content, err := os.ReadFile(tableFile)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		if err := json.Unmarshal(content, &data); err != nil {
			return fmt.Errorf("invalid JSON in file: %w", err)
		}
	} else if tableData != "" {
		if err := json.Unmarshal([]byte(tableData), &data); err != nil {
			return fmt.Errorf("invalid JSON data: %w", err)
		}
	} else {
		return fmt.Errorf("either --data or --file is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	// Parse where conditions
	conditions := strings.Split(tableWhere, ",")
	for _, cond := range conditions {
		parts := strings.SplitN(cond, "=", 2)
		if len(parts) == 2 {
			query.Set(parts[0], parts[1])
		}
	}

	path := fmt.Sprintf("/api/v1/tables/%s/%s", url.PathEscape(schema), url.PathEscape(table))

	if err := apiClient.DoRequestWithQuery(ctx, "PATCH", path, data, query); err != nil {
		return err
	}

	fmt.Println("Records updated successfully.")
	return nil
}

func runTablesDelete(cmd *cobra.Command, args []string) error {
	tableName := args[0]

	// Parse schema.table format
	schema := "public"
	table := tableName
	if strings.Contains(tableName, ".") {
		parts := strings.SplitN(tableName, ".", 2)
		schema = parts[0]
		table = parts[1]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := url.Values{}
	// Parse where conditions
	conditions := strings.Split(tableWhere, ",")
	for _, cond := range conditions {
		parts := strings.SplitN(cond, "=", 2)
		if len(parts) == 2 {
			query.Set(parts[0], parts[1])
		}
	}

	path := fmt.Sprintf("/api/v1/tables/%s/%s", url.PathEscape(schema), url.PathEscape(table))

	if err := apiClient.DoRequestWithQuery(ctx, "DELETE", path, nil, query); err != nil {
		return err
	}

	fmt.Println("Records deleted successfully.")
	return nil
}
