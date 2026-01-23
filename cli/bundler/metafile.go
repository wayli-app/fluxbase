// Package bundler provides client-side bundling for edge functions and jobs.
package bundler

// Metafile represents the esbuild metafile JSON structure
type Metafile struct {
	Inputs  map[string]MetafileInput  `json:"inputs"`
	Outputs map[string]MetafileOutput `json:"outputs"`
}

// MetafileInput represents an input file in the metafile
type MetafileInput struct {
	Bytes   int              `json:"bytes"`
	Imports []MetafileImport `json:"imports"`
	Format  string           `json:"format,omitempty"` // "cjs" or "esm"
}

// MetafileImport represents an import in the metafile
type MetafileImport struct {
	Path     string `json:"path"`
	Kind     string `json:"kind"`
	External bool   `json:"external,omitempty"`
	Original string `json:"original,omitempty"`
}

// MetafileOutput represents an output file in the metafile
type MetafileOutput struct {
	Bytes      int                     `json:"bytes"`
	Inputs     map[string]InputContrib `json:"inputs"`
	Imports    []MetafileImport        `json:"imports"`
	Exports    []string                `json:"exports"`
	EntryPoint string                  `json:"entryPoint,omitempty"`
}

// InputContrib represents the contribution of an input to an output
type InputContrib struct {
	BytesInOutput int `json:"bytesInOutput"`
}

// AnalysisResult contains the analyzed bundle information
type AnalysisResult struct {
	FunctionName    string
	TotalBytes      int
	InputFiles      []FileAnalysis
	ExternalImports []string
	Warnings        []string
}

// FileAnalysis contains analysis for a single file
type FileAnalysis struct {
	Path          string
	Bytes         int
	BytesInOutput int
	Percentage    float64
	ImportCount   int
	IsExternal    bool
}
