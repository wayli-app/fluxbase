package api

// addRealtimeEndpoints adds realtime endpoints to the spec
func (h *OpenAPIHandler) addRealtimeEndpoints(spec *OpenAPISpec) {
	// Realtime schemas
	spec.Components.Schemas["RealtimeStats"] = map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"connections":      map[string]string{"type": "integer"},
			"channels":         map[string]string{"type": "integer"},
			"messages_per_sec": map[string]string{"type": "number"},
		},
	}

	spec.Components.Schemas["BroadcastRequest"] = map[string]interface{}{
		"type":     "object",
		"required": []string{"channel", "event", "payload"},
		"properties": map[string]interface{}{
			"channel": map[string]string{"type": "string"},
			"event":   map[string]string{"type": "string"},
			"payload": map[string]string{"type": "object"},
		},
	}

	// GET /realtime - WebSocket endpoint
	spec.Paths["/realtime"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "WebSocket connection",
			Description: "Establish a WebSocket connection for realtime updates. Upgrade to WebSocket protocol required.",
			OperationID: "realtime_connect",
			Tags:        []string{"Realtime"},
			Responses: map[string]OpenAPIResponse{
				"101": {
					Description: "Switching Protocols - WebSocket connection established",
				},
			},
		},
	}

	// GET /api/v1/realtime/stats
	spec.Paths["/api/v1/realtime/stats"] = OpenAPIPath{
		"get": OpenAPIOperation{
			Summary:     "Get realtime stats",
			Description: "Get current realtime connection statistics",
			OperationID: "realtime_stats",
			Tags:        []string{"Realtime"},
			Security: []map[string][]string{
				{"bearerAuth": {}},
			},
			Responses: map[string]OpenAPIResponse{
				"200": {
					Description: "Realtime statistics",
					Content: map[string]OpenAPIMedia{
						"application/json": {
							Schema: map[string]string{"$ref": "#/components/schemas/RealtimeStats"},
						},
					},
				},
			},
		},
	}
}
