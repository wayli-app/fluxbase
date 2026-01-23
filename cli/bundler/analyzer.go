package bundler

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Analyzer provides bundle analysis using esbuild metafile
type Analyzer struct {
	sourceDir string
}

// NewAnalyzer creates a new bundle analyzer
func NewAnalyzer(sourceDir string) *Analyzer {
	return &Analyzer{sourceDir: sourceDir}
}

// AnalyzeBundle bundles code with esbuild and returns analysis
func (a *Analyzer) AnalyzeBundle(ctx context.Context, code string, functionName string, sharedModules map[string]string) (*AnalysisResult, error) {
	// Create temporary directory for bundle
	tmpDir, err := os.MkdirTemp("", "fluxbase-analyze-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Transform bare imports to npm: specifiers before writing
	code = transformBareImports(code)

	// Write main file
	mainPath := filepath.Join(tmpDir, "index.ts")
	if err := os.WriteFile(mainPath, []byte(code), 0600); err != nil {
		return nil, fmt.Errorf("failed to write main file: %w", err)
	}

	// Write shared modules
	for modulePath, content := range sharedModules {
		// Skip data files - they should already be inlined
		if isDataFile(modulePath) {
			continue
		}

		// Ensure path starts with _shared/
		if !strings.HasPrefix(modulePath, "_shared/") {
			modulePath = "_shared/" + modulePath
		}

		// Transform bare imports in shared modules too
		content = transformBareImports(content)

		fullPath := filepath.Join(tmpDir, modulePath)
		dir := filepath.Dir(fullPath)
		if err := os.MkdirAll(dir, 0750); err != nil {
			return nil, fmt.Errorf("failed to create directory for %s: %w", modulePath, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0600); err != nil {
			return nil, fmt.Errorf("failed to write shared module %s: %w", modulePath, err)
		}
	}

	// Build with esbuild, enabling metafile
	result := api.Build(api.BuildOptions{
		EntryPoints:       []string{mainPath},
		Bundle:            true,
		Write:             false, // Don't write to disk, just analyze
		Metafile:          true,  // Generate metafile
		Format:            api.FormatESModule,
		Platform:          api.PlatformNeutral, // Neutral platform for Deno
		Target:            api.ESNext,
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
		AbsWorkingDir:     tmpDir,
		Plugins: []api.Plugin{
			a.denoExternalPlugin(),
		},
	})

	// Check for errors
	if len(result.Errors) > 0 {
		var errMsgs []string
		for _, err := range result.Errors {
			errMsgs = append(errMsgs, err.Text)
		}
		return nil, fmt.Errorf("bundle analysis failed: %s", strings.Join(errMsgs, "; "))
	}

	// Parse metafile
	var metafile Metafile
	if err := json.Unmarshal([]byte(result.Metafile), &metafile); err != nil {
		return nil, fmt.Errorf("failed to parse metafile: %w", err)
	}

	// Analyze the metafile
	return a.analyzeMetafile(&metafile, functionName, tmpDir)
}

// denoExternalPlugin marks Deno-specific imports as external
func (a *Analyzer) denoExternalPlugin() api.Plugin {
	return api.Plugin{
		Name: "deno-external",
		Setup: func(build api.PluginBuild) {
			// Mark npm: imports as external
			build.OnResolve(api.OnResolveOptions{Filter: `^npm:`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{
						Path:     args.Path,
						External: true,
					}, nil
				})

			// Mark https:// and http:// imports as external
			build.OnResolve(api.OnResolveOptions{Filter: `^https?://`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{
						Path:     args.Path,
						External: true,
					}, nil
				})

			// Mark jsr: imports as external
			build.OnResolve(api.OnResolveOptions{Filter: `^jsr:`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{
						Path:     args.Path,
						External: true,
					}, nil
				})

			// Mark node: imports as external
			build.OnResolve(api.OnResolveOptions{Filter: `^node:`},
				func(args api.OnResolveArgs) (api.OnResolveResult, error) {
					return api.OnResolveResult{
						Path:     args.Path,
						External: true,
					}, nil
				})
		},
	}
}

// analyzeMetafile processes the metafile and returns analysis
func (a *Analyzer) analyzeMetafile(meta *Metafile, functionName string, tmpDir string) (*AnalysisResult, error) {
	result := &AnalysisResult{
		FunctionName: functionName,
	}

	// Find the main output (there should be only one for single entry point)
	for _, output := range meta.Outputs {
		result.TotalBytes = output.Bytes

		// Collect external imports
		for _, imp := range output.Imports {
			if imp.External {
				result.ExternalImports = append(result.ExternalImports, imp.Path)
			}
		}

		// Analyze input contributions
		for inputPath, contrib := range output.Inputs {
			// Clean up the path for display
			displayPath := inputPath
			// Remove temp directory prefix
			if strings.HasPrefix(displayPath, tmpDir) {
				displayPath = strings.TrimPrefix(displayPath, tmpDir)
				displayPath = strings.TrimPrefix(displayPath, "/")
			}
			// Replace index.ts with entry file indicator
			if displayPath == "index.ts" {
				displayPath = "<entry>"
			}

			// Get input file info
			inputInfo, ok := meta.Inputs[inputPath]
			if !ok {
				continue
			}

			percentage := 0.0
			if result.TotalBytes > 0 {
				percentage = float64(contrib.BytesInOutput) / float64(result.TotalBytes) * 100
			}

			result.InputFiles = append(result.InputFiles, FileAnalysis{
				Path:          displayPath,
				Bytes:         inputInfo.Bytes,
				BytesInOutput: contrib.BytesInOutput,
				Percentage:    percentage,
				ImportCount:   len(inputInfo.Imports),
				IsExternal:    false,
			})
		}

		// Only process first output
		break
	}

	// Sort by bytes in output (largest first)
	sort.Slice(result.InputFiles, func(i, j int) bool {
		return result.InputFiles[i].BytesInOutput > result.InputFiles[j].BytesInOutput
	})

	// Sort external imports alphabetically
	sort.Strings(result.ExternalImports)

	return result, nil
}
