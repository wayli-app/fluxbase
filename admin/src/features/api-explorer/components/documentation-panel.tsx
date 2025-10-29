import { ScrollArea } from '@/components/ui/scroll-area'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { cn } from '@/lib/utils'
import type { EndpointInfo, OpenAPISchema } from '../types'

interface DocumentationPanelProps {
  endpoint: EndpointInfo | null
}

const TYPE_COLORS = {
  string: 'text-green-600',
  number: 'text-blue-600',
  integer: 'text-blue-600',
  boolean: 'text-purple-600',
  array: 'text-orange-600',
  object: 'text-pink-600',
}

export function DocumentationPanel({ endpoint }: DocumentationPanelProps) {
  if (!endpoint) {
    return (
      <div className="flex items-center justify-center h-full text-muted-foreground">
        Select an endpoint to view documentation
      </div>
    )
  }

  const renderSchema = (schema: OpenAPISchema, indent = 0): React.ReactNode => {
    if (!schema) return null

    if (schema.$ref) {
      const refName = schema.$ref.split('/').pop()
      return (
        <span className="text-muted-foreground italic">
          Reference: {refName}
        </span>
      )
    }

    const padding = `pl-${Math.min(indent * 4, 12)}`

    if (schema.type === 'object' && schema.properties) {
      return (
        <div className={padding}>
          <div className="text-xs text-muted-foreground mb-2">Object</div>
          {Object.entries(schema.properties).map(([key, prop]) => (
            <div key={key} className="mb-2">
              <div className="flex items-start gap-2">
                <span className="font-mono text-sm">
                  {key}
                  {schema.required?.includes(key) && (
                    <span className="text-red-500 ml-1">*</span>
                  )}
                </span>
                <span className={cn(
                  "text-xs",
                  TYPE_COLORS[prop.type as keyof typeof TYPE_COLORS] || 'text-gray-600'
                )}>
                  {prop.type}
                  {prop.format && ` (${prop.format})`}
                </span>
              </div>
              {prop.description && (
                <div className="text-xs text-muted-foreground mt-1 ml-4">
                  {prop.description}
                </div>
              )}
              {prop.enum && (
                <div className="text-xs text-muted-foreground mt-1 ml-4">
                  Enum: {prop.enum.join(', ')}
                </div>
              )}
              {prop.example !== undefined && (
                <div className="text-xs text-muted-foreground mt-1 ml-4">
                  Example: <code className="bg-muted px-1 rounded">
                    {JSON.stringify(prop.example)}
                  </code>
                </div>
              )}
              {prop.type === 'object' && prop.properties && (
                <div className="ml-4 mt-2 border-l pl-4">
                  {renderSchema(prop, indent + 1)}
                </div>
              )}
              {prop.type === 'array' && prop.items && (
                <div className="ml-4 mt-2 border-l pl-4">
                  <span className="text-xs text-muted-foreground">Array of:</span>
                  {renderSchema(prop.items, indent + 1)}
                </div>
              )}
            </div>
          ))}
        </div>
      )
    }

    if (schema.type === 'array' && schema.items) {
      return (
        <div className={padding}>
          <div className="text-xs text-muted-foreground mb-2">Array</div>
          {renderSchema(schema.items, indent + 1)}
        </div>
      )
    }

    return (
      <div className={padding}>
        <span className={cn(
          "text-xs",
          TYPE_COLORS[schema.type as keyof typeof TYPE_COLORS] || 'text-gray-600'
        )}>
          {schema.type}
          {schema.format && ` (${schema.format})`}
        </span>
        {schema.description && (
          <div className="text-xs text-muted-foreground mt-1">
            {schema.description}
          </div>
        )}
        {schema.enum && (
          <div className="text-xs text-muted-foreground mt-1">
            Enum: {schema.enum.join(', ')}
          </div>
        )}
        {schema.example !== undefined && (
          <div className="text-xs text-muted-foreground mt-1">
            Example: <code className="bg-muted px-1 rounded">
              {JSON.stringify(schema.example)}
            </code>
          </div>
        )}
      </div>
    )
  }

  return (
    <ScrollArea className="h-full">
      <div className="p-6 space-y-6">
        {/* Endpoint Header */}
        <div>
          <div className="flex items-center gap-2 mb-2">
            <Badge variant="outline" className="text-xs">
              {endpoint.method}
            </Badge>
            <code className="text-sm font-mono">{endpoint.path}</code>
          </div>
          {endpoint.operationId && (
            <div className="text-xs text-muted-foreground">
              Operation ID: <code>{endpoint.operationId}</code>
            </div>
          )}
        </div>

        {/* Summary & Description */}
        {(endpoint.summary || endpoint.description) && (
          <div>
            {endpoint.summary && (
              <h3 className="font-semibold mb-2">{endpoint.summary}</h3>
            )}
            {endpoint.description && (
              <p className="text-sm text-muted-foreground whitespace-pre-wrap">
                {endpoint.description}
              </p>
            )}
          </div>
        )}

        <Separator />

        {/* Parameters */}
        {endpoint.parameters && endpoint.parameters.length > 0 && (
          <div>
            <h3 className="font-semibold mb-3">Parameters</h3>
            <div className="space-y-3">
              {endpoint.parameters.map((param, idx) => (
                <div key={`${param.name}-${idx}`} className="border rounded-lg p-3">
                  <div className="flex items-center gap-2 mb-1">
                    <code className="font-mono text-sm">{param.name}</code>
                    {param.required && (
                      <Badge variant="destructive" className="text-xs">Required</Badge>
                    )}
                    <Badge variant="secondary" className="text-xs">
                      in: {param.in}
                    </Badge>
                    {param.deprecated && (
                      <Badge variant="outline" className="text-xs">Deprecated</Badge>
                    )}
                  </div>
                  {param.description && (
                    <p className="text-xs text-muted-foreground mt-1">
                      {param.description}
                    </p>
                  )}
                  {param.schema && (
                    <div className="mt-2">
                      {renderSchema(param.schema)}
                    </div>
                  )}
                  {param.example !== undefined && (
                    <div className="text-xs text-muted-foreground mt-2">
                      Example: <code className="bg-muted px-1 rounded">
                        {JSON.stringify(param.example)}
                      </code>
                    </div>
                  )}
                </div>
              ))}
            </div>
          </div>
        )}

        {/* Request Body */}
        {endpoint.requestBody && (
          <div>
            <h3 className="font-semibold mb-3">
              Request Body
              {endpoint.requestBody.required && (
                <Badge variant="destructive" className="ml-2 text-xs">Required</Badge>
              )}
            </h3>
            {endpoint.requestBody.description && (
              <p className="text-sm text-muted-foreground mb-3">
                {endpoint.requestBody.description}
              </p>
            )}
            <Accordion type="single" collapsible defaultValue="application/json">
              {Object.entries(endpoint.requestBody.content).map(([mediaType, content]) => (
                <AccordionItem key={mediaType} value={mediaType}>
                  <AccordionTrigger className="text-sm">
                    <code>{mediaType}</code>
                  </AccordionTrigger>
                  <AccordionContent>
                    {content.schema && renderSchema(content.schema)}
                    {content.example !== undefined && (
                      <div className="mt-4">
                        <div className="text-xs font-semibold mb-2">Example:</div>
                        <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                          {JSON.stringify(content.example, null, 2)}
                        </pre>
                      </div>
                    )}
                    {content.examples && (
                      <div className="mt-4 space-y-3">
                        {Object.entries(content.examples).map(([name, example]) => (
                          <div key={name}>
                            <div className="text-xs font-semibold mb-2">
                              Example: {name}
                              {example.summary && (
                                <span className="font-normal text-muted-foreground ml-2">
                                  - {example.summary}
                                </span>
                              )}
                            </div>
                            <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                              {JSON.stringify(example.value, null, 2)}
                            </pre>
                          </div>
                        ))}
                      </div>
                    )}
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </div>
        )}

        {/* Responses */}
        {endpoint.responses && (
          <div>
            <h3 className="font-semibold mb-3">Responses</h3>
            <Accordion type="single" collapsible defaultValue="200">
              {Object.entries(endpoint.responses).map(([statusCode, response]) => (
                <AccordionItem key={statusCode} value={statusCode}>
                  <AccordionTrigger className="text-sm">
                    <div className="flex items-center gap-2">
                      <Badge
                        variant={statusCode.startsWith('2') ? 'default' :
                                statusCode.startsWith('4') ? 'destructive' : 'secondary'}
                        className="text-xs"
                      >
                        {statusCode}
                      </Badge>
                      <span className="text-sm">{response.description}</span>
                    </div>
                  </AccordionTrigger>
                  <AccordionContent>
                    {response.content && Object.entries(response.content).map(([mediaType, content]) => (
                      <div key={mediaType} className="space-y-3">
                        <div className="text-xs text-muted-foreground">
                          Content-Type: <code>{mediaType}</code>
                        </div>
                        {content.schema && renderSchema(content.schema)}
                        {content.example !== undefined && (
                          <div>
                            <div className="text-xs font-semibold mb-2">Example:</div>
                            <pre className="bg-muted p-3 rounded text-xs overflow-x-auto">
                              {JSON.stringify(content.example, null, 2)}
                            </pre>
                          </div>
                        )}
                      </div>
                    ))}
                    {!response.content && (
                      <div className="text-sm text-muted-foreground">
                        No response body
                      </div>
                    )}
                  </AccordionContent>
                </AccordionItem>
              ))}
            </Accordion>
          </div>
        )}
      </div>
    </ScrollArea>
  )
}