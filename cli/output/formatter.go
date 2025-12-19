// Package output provides output formatting for the Fluxbase CLI.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// Format represents the output format
type Format string

const (
	FormatTable Format = "table"
	FormatJSON  Format = "json"
	FormatYAML  Format = "yaml"
)

// ParseFormat parses a format string
func ParseFormat(s string) (Format, error) {
	switch strings.ToLower(s) {
	case "table", "":
		return FormatTable, nil
	case "json":
		return FormatJSON, nil
	case "yaml", "yml":
		return FormatYAML, nil
	default:
		return "", fmt.Errorf("invalid output format: %s (valid: table, json, yaml)", s)
	}
}

// Formatter formats output in various formats
type Formatter struct {
	Format    Format
	NoHeaders bool
	Quiet     bool
	Writer    io.Writer
}

// NewFormatter creates a new formatter
func NewFormatter(format Format, noHeaders, quiet bool) *Formatter {
	return &Formatter{
		Format:    format,
		NoHeaders: noHeaders,
		Quiet:     quiet,
		Writer:    os.Stdout,
	}
}

// Print outputs data in the configured format
func (f *Formatter) Print(data interface{}) error {
	if f.Quiet {
		return nil
	}

	switch f.Format {
	case FormatJSON:
		return f.printJSON(data)
	case FormatYAML:
		return f.printYAML(data)
	default:
		return f.printGeneric(data)
	}
}

func (f *Formatter) printJSON(data interface{}) error {
	encoder := json.NewEncoder(f.Writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(data)
}

func (f *Formatter) printYAML(data interface{}) error {
	encoder := yaml.NewEncoder(f.Writer)
	encoder.SetIndent(2)
	defer func() { _ = encoder.Close() }()
	return encoder.Encode(data)
}

func (f *Formatter) printGeneric(data interface{}) error {
	// For generic data, fall back to JSON in table mode
	return f.printJSON(data)
}

// TableData represents tabular data for table output
type TableData struct {
	Headers []string
	Rows    [][]string
}

// PrintTable prints formatted table output
func (f *Formatter) PrintTable(data TableData) {
	if f.Quiet {
		return
	}

	// For non-table formats, convert to list of maps
	if f.Format != FormatTable {
		rows := make([]map[string]string, len(data.Rows))
		for i, row := range data.Rows {
			rowMap := make(map[string]string)
			for j, cell := range row {
				if j < len(data.Headers) {
					rowMap[data.Headers[j]] = cell
				}
			}
			rows[i] = rowMap
		}
		_ = f.Print(rows)
		return
	}

	table := tablewriter.NewWriter(f.Writer)

	if !f.NoHeaders && len(data.Headers) > 0 {
		table.SetHeader(data.Headers)
	}

	// Configure table style
	table.SetBorder(false)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)

	table.AppendBulk(data.Rows)
	table.Render()
}

// PrintSuccess prints a success message
func (f *Formatter) PrintSuccess(message string) {
	if f.Quiet {
		return
	}
	_, _ = fmt.Fprintln(f.Writer, message)
}

// PrintError prints an error message
func (f *Formatter) PrintError(message string) {
	fmt.Fprintln(os.Stderr, "Error:", message)
}

// PrintWarning prints a warning message
func (f *Formatter) PrintWarning(message string) {
	if f.Quiet {
		return
	}
	fmt.Fprintln(os.Stderr, "Warning:", message)
}

// PrintInfo prints an info message
func (f *Formatter) PrintInfo(message string) {
	if f.Quiet {
		return
	}
	_, _ = fmt.Fprintln(f.Writer, message)
}

// PrintKeyValue prints a key-value pair
func (f *Formatter) PrintKeyValue(key, value string) {
	if f.Quiet {
		return
	}

	switch f.Format {
	case FormatJSON:
		_ = f.printJSON(map[string]string{key: value})
	case FormatYAML:
		_ = f.printYAML(map[string]string{key: value})
	default:
		_, _ = fmt.Fprintf(f.Writer, "%s: %s\n", key, value)
	}
}

// PrintList prints a list of items
func (f *Formatter) PrintList(items []string) {
	if f.Quiet {
		return
	}

	switch f.Format {
	case FormatJSON:
		_ = f.printJSON(items)
	case FormatYAML:
		_ = f.printYAML(items)
	default:
		for _, item := range items {
			_, _ = fmt.Fprintln(f.Writer, item)
		}
	}
}
