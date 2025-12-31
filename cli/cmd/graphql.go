package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxbase-eu/fluxbase/cli/output"
)

var graphqlCmd = &cobra.Command{
	Use:     "graphql",
	Aliases: []string{"gql"},
	Short:   "Execute GraphQL queries and mutations",
	Long: `Execute GraphQL queries and mutations against the Fluxbase GraphQL API.

The GraphQL API is auto-generated from your PostgreSQL database schema,
providing type-safe queries and mutations for all your tables.

Examples:
  # Execute a simple query
  fluxbase graphql query '{ users { id email } }'

  # Execute a query from a file
  fluxbase graphql query --file ./query.graphql

  # Execute a query with variables
  fluxbase graphql query 'query($id: ID!) { user(id: $id) { email } }' --var 'id=123'

  # Execute a mutation
  fluxbase graphql mutation 'mutation { insert_users(objects: [{email: "test@example.com"}]) { returning { id } } }'

  # Introspect the schema
  fluxbase graphql introspect`,
}

var (
	graphqlFile      string
	graphqlVariables []string
	graphqlPretty    bool
)

// GraphQL request/response types
type graphqlRequest struct {
	Query         string                 `json:"query"`
	Variables     map[string]interface{} `json:"variables,omitempty"`
	OperationName string                 `json:"operationName,omitempty"`
}

type graphqlResponse struct {
	Data   interface{}    `json:"data,omitempty"`
	Errors []graphqlError `json:"errors,omitempty"`
}

type graphqlError struct {
	Message    string                 `json:"message"`
	Locations  []graphqlErrorLocation `json:"locations,omitempty"`
	Path       []interface{}          `json:"path,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

type graphqlErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

var graphqlQueryCmd = &cobra.Command{
	Use:   "query [query]",
	Short: "Execute a GraphQL query",
	Long: `Execute a GraphQL query against the Fluxbase GraphQL API.

The query can be provided as an argument or from a file using --file.
Variables can be passed using --var flags in the format 'name=value'.

Examples:
  # Simple query
  fluxbase graphql query '{ users { id email created_at } }'

  # Query with select fields
  fluxbase graphql query '{ users(limit: 10, order_by: {created_at: desc}) { id email } }'

  # Query with filtering
  fluxbase graphql query '{ users(where: {role: {_eq: "admin"}}) { id email } }'

  # Query from file
  fluxbase graphql query --file ./get-users.graphql

  # Query with variables
  fluxbase graphql query 'query GetUser($id: ID!) { user(id: $id) { id email } }' --var 'id=abc-123'

  # Multiple variables
  fluxbase graphql query 'query($limit: Int, $offset: Int) { users(limit: $limit, offset: $offset) { id } }' \
    --var 'limit=10' --var 'offset=20'`,
	PreRunE: requireAuth,
	RunE:    runGraphQLQuery,
}

var graphqlMutationCmd = &cobra.Command{
	Use:   "mutation [mutation]",
	Short: "Execute a GraphQL mutation",
	Long: `Execute a GraphQL mutation against the Fluxbase GraphQL API.

The mutation can be provided as an argument or from a file using --file.
Variables can be passed using --var flags in the format 'name=value'.

Examples:
  # Insert a record
  fluxbase graphql mutation 'mutation {
    insert_users(objects: [{email: "new@example.com", name: "New User"}]) {
      returning { id email }
    }
  }'

  # Update records
  fluxbase graphql mutation 'mutation {
    update_users(where: {id: {_eq: "user-id"}}, _set: {name: "Updated Name"}) {
      affected_rows
      returning { id name }
    }
  }'

  # Delete records
  fluxbase graphql mutation 'mutation {
    delete_users(where: {id: {_eq: "user-id"}}) {
      affected_rows
    }
  }'

  # Mutation with variables
  fluxbase graphql mutation 'mutation CreateUser($email: String!, $name: String!) {
    insert_users(objects: [{email: $email, name: $name}]) {
      returning { id }
    }
  }' --var 'email=test@example.com' --var 'name=Test User'`,
	PreRunE: requireAuth,
	RunE:    runGraphQLMutation,
}

var graphqlIntrospectCmd = &cobra.Command{
	Use:   "introspect",
	Short: "Introspect the GraphQL schema",
	Long: `Fetch and display the GraphQL schema via introspection.

This command queries the GraphQL schema to show available types,
fields, and operations. Useful for exploring the auto-generated
schema from your database tables.

Note: Introspection must be enabled on the server (default: enabled in
development, disabled in production).

Examples:
  # Full introspection query
  fluxbase graphql introspect

  # List only type names
  fluxbase graphql introspect --types

  # Output as JSON
  fluxbase graphql introspect -o json`,
	PreRunE: requireAuth,
	RunE:    runGraphQLIntrospect,
}

var (
	introspectTypesOnly bool
)

func init() {
	// Query flags
	graphqlQueryCmd.Flags().StringVarP(&graphqlFile, "file", "f", "", "File containing the GraphQL query")
	graphqlQueryCmd.Flags().StringArrayVar(&graphqlVariables, "var", nil, "Variables in format 'name=value' (can be repeated)")
	graphqlQueryCmd.Flags().BoolVar(&graphqlPretty, "pretty", true, "Pretty print JSON output")

	// Mutation flags
	graphqlMutationCmd.Flags().StringVarP(&graphqlFile, "file", "f", "", "File containing the GraphQL mutation")
	graphqlMutationCmd.Flags().StringArrayVar(&graphqlVariables, "var", nil, "Variables in format 'name=value' (can be repeated)")
	graphqlMutationCmd.Flags().BoolVar(&graphqlPretty, "pretty", true, "Pretty print JSON output")

	// Introspect flags
	graphqlIntrospectCmd.Flags().BoolVar(&introspectTypesOnly, "types", false, "List only type names")

	graphqlCmd.AddCommand(graphqlQueryCmd)
	graphqlCmd.AddCommand(graphqlMutationCmd)
	graphqlCmd.AddCommand(graphqlIntrospectCmd)
}

func runGraphQLQuery(cmd *cobra.Command, args []string) error {
	return executeGraphQL(cmd, args, "query")
}

func runGraphQLMutation(cmd *cobra.Command, args []string) error {
	return executeGraphQL(cmd, args, "mutation")
}

func executeGraphQL(cmd *cobra.Command, args []string, operationType string) error {
	// Get the query/mutation
	var query string
	if graphqlFile != "" {
		content, err := os.ReadFile(graphqlFile) //nolint:gosec // CLI tool reads user-provided file path
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		query = string(content)
	} else if len(args) > 0 {
		query = args[0]
	} else {
		return fmt.Errorf("either provide a %s as an argument or use --file", operationType)
	}

	// Parse variables
	variables, err := parseVariables(graphqlVariables)
	if err != nil {
		return err
	}

	// Execute the query
	return doGraphQLRequest(query, variables)
}

func parseVariables(vars []string) (map[string]interface{}, error) {
	if len(vars) == 0 {
		return nil, nil
	}

	result := make(map[string]interface{})
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid variable format: %q (expected 'name=value')", v)
		}
		name := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Try to parse as JSON first (for objects, arrays, numbers, booleans)
		var parsed interface{}
		if err := json.Unmarshal([]byte(value), &parsed); err == nil {
			result[name] = parsed
		} else {
			// Use as string if not valid JSON
			result[name] = value
		}
	}

	return result, nil
}

func doGraphQLRequest(query string, variables map[string]interface{}) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	reqBody := graphqlRequest{
		Query:     query,
		Variables: variables,
	}

	var response graphqlResponse
	if err := apiClient.DoPost(ctx, "/api/v1/graphql", reqBody, &response); err != nil {
		return err
	}

	// Check for errors
	if len(response.Errors) > 0 {
		return formatGraphQLErrors(response.Errors)
	}

	// Format output
	formatter := GetFormatter()

	if formatter.Format == output.FormatTable {
		// For table format, try to display data in a readable way
		if response.Data != nil {
			if graphqlPretty {
				data, err := json.MarshalIndent(response.Data, "", "  ")
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			} else {
				data, err := json.Marshal(response.Data)
				if err != nil {
					return err
				}
				fmt.Println(string(data))
			}
		}
	} else {
		// JSON/YAML output
		if err := formatter.Print(response); err != nil {
			return err
		}
	}

	return nil
}

func formatGraphQLErrors(errs []graphqlError) error {
	var sb strings.Builder
	sb.WriteString("GraphQL errors:\n")

	for i, e := range errs {
		sb.WriteString(fmt.Sprintf("  %d. %s", i+1, e.Message))

		if len(e.Locations) > 0 {
			loc := e.Locations[0]
			sb.WriteString(fmt.Sprintf(" (line %d, column %d)", loc.Line, loc.Column))
		}

		if len(e.Path) > 0 {
			pathParts := make([]string, len(e.Path))
			for j, p := range e.Path {
				pathParts[j] = fmt.Sprintf("%v", p)
			}
			sb.WriteString(fmt.Sprintf(" at path: %s", strings.Join(pathParts, ".")))
		}

		sb.WriteString("\n")
	}

	return errors.New(sb.String())
}

func runGraphQLIntrospect(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var query string
	if introspectTypesOnly {
		// Simple query for type names only
		query = `{
			__schema {
				types {
					name
					kind
				}
			}
		}`
	} else {
		// Full introspection query
		query = introspectionQuery
	}

	reqBody := graphqlRequest{
		Query: query,
	}

	var response graphqlResponse
	if err := apiClient.DoPost(ctx, "/api/v1/graphql", reqBody, &response); err != nil {
		return err
	}

	// Check for errors
	if len(response.Errors) > 0 {
		return formatGraphQLErrors(response.Errors)
	}

	formatter := GetFormatter()

	if introspectTypesOnly && formatter.Format == output.FormatTable {
		// Display types in a table
		return displayTypesTable(response.Data)
	}

	// JSON/YAML output or full introspection
	if graphqlPretty || formatter.Format != output.FormatTable {
		if formatter.Format == output.FormatTable {
			data, err := json.MarshalIndent(response.Data, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			if err := formatter.Print(response); err != nil {
				return err
			}
		}
	}

	return nil
}

func displayTypesTable(data interface{}) error {
	// Extract types from response
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected response format")
	}

	schema, ok := dataMap["__schema"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("unexpected schema format")
	}

	types, ok := schema["types"].([]interface{})
	if !ok {
		return fmt.Errorf("unexpected types format")
	}

	// Filter out internal types (starting with __)
	var userTypes []map[string]interface{}
	for _, t := range types {
		typeMap, ok := t.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := typeMap["name"].(string)
		if !strings.HasPrefix(name, "__") {
			userTypes = append(userTypes, typeMap)
		}
	}

	if len(userTypes) == 0 {
		fmt.Println("No types found.")
		return nil
	}

	formatter := GetFormatter()

	tableData := output.TableData{
		Headers: []string{"TYPE", "KIND"},
		Rows:    make([][]string, len(userTypes)),
	}

	for i, t := range userTypes {
		name, _ := t["name"].(string)
		kind, _ := t["kind"].(string)
		tableData.Rows[i] = []string{name, kind}
	}

	formatter.PrintTable(tableData)
	return nil
}

// Standard GraphQL introspection query
const introspectionQuery = `
query IntrospectionQuery {
  __schema {
    queryType { name }
    mutationType { name }
    subscriptionType { name }
    types {
      ...FullType
    }
    directives {
      name
      description
      locations
      args {
        ...InputValue
      }
    }
  }
}

fragment FullType on __Type {
  kind
  name
  description
  fields(includeDeprecated: true) {
    name
    description
    args {
      ...InputValue
    }
    type {
      ...TypeRef
    }
    isDeprecated
    deprecationReason
  }
  inputFields {
    ...InputValue
  }
  interfaces {
    ...TypeRef
  }
  enumValues(includeDeprecated: true) {
    name
    description
    isDeprecated
    deprecationReason
  }
  possibleTypes {
    ...TypeRef
  }
}

fragment InputValue on __InputValue {
  name
  description
  type { ...TypeRef }
  defaultValue
}

fragment TypeRef on __Type {
  kind
  name
  ofType {
    kind
    name
    ofType {
      kind
      name
      ofType {
        kind
        name
        ofType {
          kind
          name
          ofType {
            kind
            name
            ofType {
              kind
              name
              ofType {
                kind
                name
              }
            }
          }
        }
      }
    }
  }
}
`
