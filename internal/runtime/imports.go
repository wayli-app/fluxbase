package runtime

import "strings"

// extractImports separates import/export statements from the rest of the code
// Import statements must be at the top level in ES modules
func extractImports(code string) (imports string, remaining string) {
	lines := strings.Split(code, "\n")
	var importLines []string
	var codeLines []string

	inMultilineDeclaration := false
	inMultilineExport := false
	braceCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Check if we're starting a multi-line type/interface declaration
		if !inMultilineDeclaration && !inMultilineExport &&
			(strings.HasPrefix(trimmed, "export type ") ||
				strings.HasPrefix(trimmed, "export interface ") ||
				strings.HasPrefix(trimmed, "export enum ")) {
			inMultilineDeclaration = true
			braceCount = 0
			importLines = append(importLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				inMultilineDeclaration = false
			}
			continue
		}

		// If we're in a multi-line declaration, continue collecting lines
		if inMultilineDeclaration {
			importLines = append(importLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount == 0 {
				inMultilineDeclaration = false
			}
			continue
		}

		// Check if we're starting a multi-line export { ... } statement
		if !inMultilineExport && strings.HasPrefix(trimmed, "export {") {
			braceCount = strings.Count(line, "{") - strings.Count(line, "}")
			importLines = append(importLines, line)
			if braceCount > 0 {
				// Opening brace without closing - multi-line export
				inMultilineExport = true
			}
			continue
		}

		// If we're in a multi-line export, continue collecting lines
		if inMultilineExport {
			importLines = append(importLines, line)
			braceCount += strings.Count(line, "{") - strings.Count(line, "}")
			if braceCount <= 0 {
				inMultilineExport = false
			}
			continue
		}

		// Extract single-line import/export statements
		if strings.HasPrefix(trimmed, "import ") ||
			strings.HasPrefix(trimmed, "import{") ||
			strings.HasPrefix(trimmed, "export * ") {
			importLines = append(importLines, line)
		} else {
			codeLines = append(codeLines, line)
		}
	}

	return strings.Join(importLines, "\n"), strings.Join(codeLines, "\n")
}
