package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fluxbase-eu/fluxbase/internal/config"
	"github.com/rs/zerolog/log"
)

// MCPVersion is the MCP protocol version supported by this server
const MCPVersion = "2024-11-05"

// FluxbaseVersion should be set at build time
var FluxbaseVersion = "unknown"

// Server handles MCP protocol operations
type Server struct {
	config    *config.MCPConfig
	transport *Transport
	tools     *ToolRegistry
	resources *ResourceRegistry
}

// NewServer creates a new MCP server
func NewServer(cfg *config.MCPConfig) *Server {
	return &Server{
		config:    cfg,
		transport: NewTransport(),
		tools:     NewToolRegistry(),
		resources: NewResourceRegistry(),
	}
}

// ToolRegistry returns the tool registry for registration
func (s *Server) ToolRegistry() *ToolRegistry {
	return s.tools
}

// ResourceRegistry returns the resource registry for registration
func (s *Server) ResourceRegistry() *ResourceRegistry {
	return s.resources
}

// HandleRequest processes a JSON-RPC request and returns a response
func (s *Server) HandleRequest(ctx context.Context, data []byte, authCtx *AuthContext) *Response {
	// Parse the request
	req, err := s.transport.ParseRequest(data)
	if err != nil {
		log.Debug().Err(err).Msg("MCP: Failed to parse request")
		return NewParseError(err.Error())
	}

	// Log the request
	log.Debug().
		Str("method", req.Method).
		Interface("id", req.ID).
		Msg("MCP: Handling request")

	// Dispatch based on method
	return s.dispatch(ctx, req, authCtx)
}

// dispatch routes the request to the appropriate handler
func (s *Server) dispatch(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	switch req.Method {
	case MethodInitialize:
		return s.handleInitialize(ctx, req)
	case MethodPing:
		return s.handlePing(ctx, req)
	case MethodToolsList:
		return s.handleToolsList(ctx, req, authCtx)
	case MethodToolsCall:
		return s.handleToolsCall(ctx, req, authCtx)
	case MethodResourcesList:
		return s.handleResourcesList(ctx, req, authCtx)
	case MethodResourcesRead:
		return s.handleResourcesRead(ctx, req, authCtx)
	case MethodResourcesTemplates:
		return s.handleResourcesTemplates(ctx, req, authCtx)
	default:
		return NewMethodNotFound(req.ID, req.Method)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(ctx context.Context, req *Request) *Response {
	params, err := ParseParams[InitializeParams](req.Params)
	if err != nil {
		return NewInvalidParams(req.ID, err.Error())
	}

	log.Info().
		Str("client_name", params.ClientInfo.Name).
		Str("client_version", params.ClientInfo.Version).
		Str("protocol_version", params.ProtocolVersion).
		Msg("MCP: Client initializing")

	result := InitializeResult{
		ProtocolVersion: MCPVersion,
		ServerInfo: ServerInfo{
			Name:            "Fluxbase MCP Server",
			Version:         FluxbaseVersion,
			ProtocolVersion: MCPVersion,
		},
		Capabilities: ServerCapabilities{
			Tools: &ToolsCapability{
				ListChanged: false, // We don't support dynamic tool list changes yet
			},
			Resources: &ResourcesCapability{
				Subscribe:   false, // We don't support resource subscriptions yet
				ListChanged: false,
			},
		},
	}

	return NewResult(req.ID, result)
}

// handlePing handles the ping request
func (s *Server) handlePing(ctx context.Context, req *Request) *Response {
	return NewResult(req.ID, PingResult{})
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	// Get all tools that the user has access to
	tools := s.tools.ListTools(authCtx)

	// Filter by allowed tools config if specified
	if len(s.config.AllowedTools) > 0 {
		allowedSet := make(map[string]bool)
		for _, name := range s.config.AllowedTools {
			allowedSet[name] = true
		}

		var filteredTools []Tool
		for _, tool := range tools {
			if allowedSet[tool.Name] {
				filteredTools = append(filteredTools, tool)
			}
		}
		tools = filteredTools
	}

	return NewResult(req.ID, ToolsListResult{Tools: tools})
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	params, err := MustParseParams[ToolsCallParams](req.Params)
	if err != nil {
		return NewInvalidParams(req.ID, err.Error())
	}

	// Check if tool is allowed by config
	if len(s.config.AllowedTools) > 0 {
		allowed := false
		for _, name := range s.config.AllowedTools {
			if name == params.Name {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewToolNotFound(req.ID, params.Name)
		}
	}

	// Get the tool
	tool := s.tools.GetTool(params.Name)
	if tool == nil {
		return NewToolNotFound(req.ID, params.Name)
	}

	// Check if user has required scopes
	if !authCtx.HasScopes(tool.RequiredScopes()...) {
		return NewForbidden(req.ID, fmt.Sprintf("missing required scopes for tool %s", params.Name))
	}

	// Execute the tool
	log.Debug().
		Str("tool", params.Name).
		Interface("arguments", params.Arguments).
		Msg("MCP: Executing tool")

	result, err := tool.Execute(ctx, params.Arguments, authCtx)
	if err != nil {
		log.Error().
			Err(err).
			Str("tool", params.Name).
			Msg("MCP: Tool execution failed")
		return NewToolExecutionError(req.ID, err.Error())
	}

	return NewResult(req.ID, result)
}

// handleResourcesList handles the resources/list request
func (s *Server) handleResourcesList(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	// Get all resources that the user has access to
	resources := s.resources.ListResources(authCtx)

	// Filter by allowed resources config if specified
	if len(s.config.AllowedResources) > 0 {
		allowedSet := make(map[string]bool)
		for _, uri := range s.config.AllowedResources {
			allowedSet[uri] = true
		}

		var filteredResources []Resource
		for _, res := range resources {
			if allowedSet[res.URI] {
				filteredResources = append(filteredResources, res)
			}
		}
		resources = filteredResources
	}

	return NewResult(req.ID, ResourcesListResult{Resources: resources})
}

// handleResourcesRead handles the resources/read request
func (s *Server) handleResourcesRead(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	params, err := MustParseParams[ResourcesReadParams](req.Params)
	if err != nil {
		return NewInvalidParams(req.ID, err.Error())
	}

	// Check if resource is allowed by config
	if len(s.config.AllowedResources) > 0 {
		allowed := false
		for _, uri := range s.config.AllowedResources {
			if uri == params.URI {
				allowed = true
				break
			}
		}
		if !allowed {
			return NewResourceNotFound(req.ID, params.URI)
		}
	}

	// Get the resource provider
	provider := s.resources.GetProvider(params.URI)
	if provider == nil {
		return NewResourceNotFound(req.ID, params.URI)
	}

	// Check if user has required scopes
	if !authCtx.HasScopes(provider.RequiredScopes()...) {
		return NewForbidden(req.ID, fmt.Sprintf("missing required scopes for resource %s", params.URI))
	}

	// Read the resource
	log.Debug().
		Str("uri", params.URI).
		Msg("MCP: Reading resource")

	var contents []Content
	// Check if this is a template resource
	if tp, ok := provider.(TemplateResourceProvider); ok && tp.IsTemplate() {
		// Extract parameters from URI
		uriParams, _ := tp.MatchURI(params.URI)
		contents, err = tp.ReadWithParams(ctx, authCtx, uriParams)
	} else {
		contents, err = provider.Read(ctx, authCtx)
	}

	if err != nil {
		log.Error().
			Err(err).
			Str("uri", params.URI).
			Msg("MCP: Resource read failed")
		return NewInternalError(req.ID, err.Error())
	}

	// Convert Content to ResourceContents
	resourceContents := make([]ResourceContents, 0, len(contents))
	for _, c := range contents {
		resourceContents = append(resourceContents, ResourceContents{
			URI:      params.URI,
			MimeType: provider.MimeType(),
			Text:     c.Text,
		})
	}

	return NewResult(req.ID, ResourcesReadResult{Contents: resourceContents})
}

// handleResourcesTemplates handles the resources/templates request
func (s *Server) handleResourcesTemplates(ctx context.Context, req *Request, authCtx *AuthContext) *Response {
	templates := s.resources.ListTemplates(authCtx)
	return NewResult(req.ID, ResourcesTemplatesResult{ResourceTemplates: templates})
}

// SerializeResponse serializes a response to JSON
func (s *Server) SerializeResponse(resp *Response) ([]byte, error) {
	return json.Marshal(resp)
}
