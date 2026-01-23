package bundler

import (
	"fmt"
	"io"
	"sort"
	"strings"
)

// DisplayAnalysis prints the bundle analysis in a formatted way
func DisplayAnalysis(w io.Writer, result *AnalysisResult, showDetails bool) {
	_, _ = fmt.Fprintf(w, "\n=== Bundle Analysis: %s ===\n", result.FunctionName)
	_, _ = fmt.Fprintf(w, "Total bundle size: %s\n", formatBytesHuman(result.TotalBytes))

	if len(result.ExternalImports) > 0 {
		_, _ = fmt.Fprintln(w, "\nExternal imports (resolved at runtime):")
		for _, imp := range result.ExternalImports {
			_, _ = fmt.Fprintf(w, "  - %s\n", imp)
		}
	}

	if len(result.InputFiles) > 0 {
		_, _ = fmt.Fprintln(w, "\nBundle breakdown:")

		// Determine how many files to show
		maxFiles := 10
		if showDetails {
			maxFiles = len(result.InputFiles)
		}

		// Calculate max path length for alignment
		maxPathLen := 0
		for i, file := range result.InputFiles {
			if i >= maxFiles {
				break
			}
			displayPath := truncatePath(file.Path, 50)
			if len(displayPath) > maxPathLen {
				maxPathLen = len(displayPath)
			}
		}

		// Print file breakdown
		for i, file := range result.InputFiles {
			if i >= maxFiles {
				remaining := len(result.InputFiles) - maxFiles
				_, _ = fmt.Fprintf(w, "  ... and %d more files\n", remaining)
				break
			}

			displayPath := truncatePath(file.Path, 50)
			padding := strings.Repeat(" ", maxPathLen-len(displayPath))
			_, _ = fmt.Fprintf(w, "  %s%s  %8s  %5.1f%%\n",
				displayPath,
				padding,
				formatBytesHuman(file.BytesInOutput),
				file.Percentage,
			)
		}
	}

	if len(result.Warnings) > 0 {
		_, _ = fmt.Fprintln(w, "\nWarnings:")
		for _, warn := range result.Warnings {
			_, _ = fmt.Fprintf(w, "  - %s\n", warn)
		}
	}

	_, _ = fmt.Fprintln(w)
}

// DisplaySummary prints a compact summary of multiple analyses
func DisplaySummary(w io.Writer, results []*AnalysisResult) {
	if len(results) == 0 {
		return
	}

	_, _ = fmt.Fprintln(w, "\n=== Bundle Size Summary ===")

	// Sort by size (largest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].TotalBytes > results[j].TotalBytes
	})

	// Calculate max name length for alignment
	maxNameLen := 8 // minimum "FUNCTION" header length
	for _, r := range results {
		if len(r.FunctionName) > maxNameLen {
			maxNameLen = len(r.FunctionName)
		}
	}

	// Print header
	namePadding := strings.Repeat(" ", maxNameLen-8)
	_, _ = fmt.Fprintf(w, "FUNCTION%s  BUNDLE SIZE  FILES  EXTERNALS\n", namePadding)
	_, _ = fmt.Fprintf(w, "%s  -----------  -----  ---------\n", strings.Repeat("-", maxNameLen))

	var totalSize int
	for _, r := range results {
		totalSize += r.TotalBytes
		padding := strings.Repeat(" ", maxNameLen-len(r.FunctionName))
		_, _ = fmt.Fprintf(w, "%s%s  %11s  %5d  %9d\n",
			r.FunctionName,
			padding,
			formatBytesHuman(r.TotalBytes),
			len(r.InputFiles),
			len(r.ExternalImports),
		)
	}

	// Print total
	_, _ = fmt.Fprintf(w, "%s  -----------  -----  ---------\n", strings.Repeat("-", maxNameLen))
	totalPadding := strings.Repeat(" ", maxNameLen-5)
	_, _ = fmt.Fprintf(w, "TOTAL%s  %11s\n", totalPadding, formatBytesHuman(totalSize))
	_, _ = fmt.Fprintln(w)
}

// formatBytesHuman formats bytes in human-readable format
func formatBytesHuman(bytes int) string {
	const (
		KB = 1024
		MB = 1024 * KB
	)
	switch {
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// truncatePath shortens a path if it's too long
func truncatePath(path string, maxLen int) string {
	if len(path) <= maxLen {
		return path
	}
	return "..." + path[len(path)-maxLen+3:]
}
