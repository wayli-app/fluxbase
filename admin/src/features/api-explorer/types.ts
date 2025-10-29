// OpenAPI Types
export interface OpenAPISpec {
  openapi: string
  info: {
    title: string
    description: string
    version: string
  }
  servers: Array<{
    url: string
    description: string
  }>
  paths: {
    [path: string]: {
      [method: string]: OpenAPIOperation
    }
  }
  components?: {
    schemas?: {
      [name: string]: OpenAPISchema
    }
    securitySchemes?: {
      [name: string]: any
    }
  }
}

export interface OpenAPIOperation {
  summary?: string
  description?: string
  operationId?: string
  tags?: string[]
  parameters?: OpenAPIParameter[]
  requestBody?: OpenAPIRequestBody
  responses: {
    [statusCode: string]: OpenAPIResponse
  }
  security?: Array<{ [name: string]: string[] }>
}

export interface OpenAPIParameter {
  name: string
  in: 'query' | 'header' | 'path' | 'cookie'
  description?: string
  required?: boolean
  deprecated?: boolean
  schema?: OpenAPISchema
  example?: any
}

export interface OpenAPIRequestBody {
  description?: string
  required?: boolean
  content: {
    [mediaType: string]: {
      schema?: OpenAPISchema
      example?: any
      examples?: { [name: string]: { value: any; summary?: string } }
    }
  }
}

export interface OpenAPIResponse {
  description: string
  content?: {
    [mediaType: string]: {
      schema?: OpenAPISchema
      example?: any
    }
  }
}

export interface OpenAPISchema {
  type?: string
  format?: string
  description?: string
  properties?: { [key: string]: OpenAPISchema }
  required?: string[]
  items?: OpenAPISchema
  enum?: any[]
  example?: any
  $ref?: string
  oneOf?: OpenAPISchema[]
  anyOf?: OpenAPISchema[]
  allOf?: OpenAPISchema[]
  additionalProperties?: boolean | OpenAPISchema
}

// Endpoint Browser Types
export interface EndpointGroup {
  name: string
  endpoints: EndpointInfo[]
  expanded?: boolean
  children?: EndpointGroup[] // For nested hierarchy (tag -> resources)
}

export interface EndpointInfo {
  path: string
  method: string
  summary?: string
  description?: string
  operationId?: string
  tags?: string[]
  parameters?: OpenAPIParameter[]
  requestBody?: OpenAPIRequestBody
  responses?: { [statusCode: string]: OpenAPIResponse }
}

// Request Template Types
export interface RequestTemplate {
  id: string
  name: string
  category: string
  description?: string
  method: string
  endpoint: string
  headers?: Record<string, string>
  queryParams?: Record<string, string>
  body?: any
}

// Parameter Builder Types
export interface ParameterInput {
  name: string
  value: string | number | boolean | null
  type: string
  required: boolean
  description?: string
  example?: any
  in: 'query' | 'header' | 'path' | 'body'
}