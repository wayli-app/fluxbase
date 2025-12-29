import type * as Monaco from 'monaco-editor'
import type { TableInfo } from '@/lib/api'

// GraphQL keywords
const GRAPHQL_KEYWORDS = [
  'query',
  'mutation',
  'subscription',
  'fragment',
  'on',
  'type',
  'interface',
  'union',
  'enum',
  'input',
  'scalar',
  'directive',
  'extend',
  'schema',
  'implements',
  'true',
  'false',
  'null',
]

// GraphQL built-in directives
const GRAPHQL_DIRECTIVES = [
  {
    name: '@skip',
    detail: 'Skip field if condition is true',
    insertText: '@skip(if: ${1:true})',
  },
  {
    name: '@include',
    detail: 'Include field if condition is true',
    insertText: '@include(if: ${1:true})',
  },
  {
    name: '@deprecated',
    detail: 'Mark as deprecated',
    insertText: '@deprecated(reason: "${1:No longer supported}")',
  },
]

// GraphQL built-in scalars
const GRAPHQL_SCALARS = ['Int', 'Float', 'String', 'Boolean', 'ID']

// Filter operators that match the backend
const FILTER_OPERATORS = [
  { name: 'eq', detail: 'Equal to' },
  { name: 'neq', detail: 'Not equal to' },
  { name: 'gt', detail: 'Greater than' },
  { name: 'gte', detail: 'Greater than or equal' },
  { name: 'lt', detail: 'Less than' },
  { name: 'lte', detail: 'Less than or equal' },
  { name: 'like', detail: 'SQL LIKE pattern match' },
  { name: 'ilike', detail: 'Case-insensitive LIKE' },
  { name: 'is', detail: 'IS NULL check' },
  { name: 'in', detail: 'In array of values' },
]

// Order by directions
const ORDER_DIRECTIONS = [
  { name: 'ASC', detail: 'Ascending order' },
  { name: 'DESC', detail: 'Descending order' },
  { name: 'ASC_NULLS_FIRST', detail: 'Ascending, nulls first' },
  { name: 'ASC_NULLS_LAST', detail: 'Ascending, nulls last' },
  { name: 'DESC_NULLS_FIRST', detail: 'Descending, nulls first' },
  { name: 'DESC_NULLS_LAST', detail: 'Descending, nulls last' },
]

export interface GraphQLSchemaMetadata {
  tables: TableInfo[]
}

// Convert PostgreSQL type to GraphQL type name
function postgresTypeToGraphQL(pgType: string): string {
  const typeMap: Record<string, string> = {
    integer: 'Int',
    smallint: 'Int',
    bigint: 'BigInt',
    serial: 'Int',
    bigserial: 'BigInt',
    numeric: 'Float',
    decimal: 'Float',
    real: 'Float',
    'double precision': 'Float',
    boolean: 'Boolean',
    text: 'String',
    'character varying': 'String',
    varchar: 'String',
    char: 'String',
    character: 'String',
    uuid: 'ID',
    json: 'JSON',
    jsonb: 'JSON',
    'timestamp with time zone': 'DateTime',
    'timestamp without time zone': 'DateTime',
    timestamptz: 'DateTime',
    timestamp: 'DateTime',
    date: 'DateTime',
    time: 'String',
    interval: 'String',
    bytea: 'String',
  }

  // Handle arrays
  if (pgType.endsWith('[]')) {
    const baseType = pgType.slice(0, -2)
    const mappedType = typeMap[baseType.toLowerCase()] || 'String'
    return `[${mappedType}]`
  }

  return typeMap[pgType.toLowerCase()] || 'String'
}

// Convert table name to PascalCase type name
function toPascalCase(str: string): string {
  return str
    .split('_')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join('')
}

// Convert to camelCase for field names
function toCamelCase(str: string): string {
  const pascal = toPascalCase(str)
  return pascal.charAt(0).toLowerCase() + pascal.slice(1)
}

// Determine context from cursor position
function getGraphQLContext(
  model: Monaco.editor.ITextModel,
  position: Monaco.Position
): {
  isInsideQuery: boolean
  isInsideMutation: boolean
  isInsideArguments: boolean
  isInsideFilter: boolean
  isInsideOrderBy: boolean
  currentType: string | null
  wordBefore: string
} {
  const fullText = model.getValue()
  const offset = model.getOffsetAt(position)
  const textBeforeCursor = fullText.substring(0, offset)

  // Check if we're inside query or mutation
  const lastQuery = textBeforeCursor.lastIndexOf('query')
  const lastMutation = textBeforeCursor.lastIndexOf('mutation')
  const isInsideQuery = lastQuery > lastMutation
  const isInsideMutation = lastMutation > lastQuery

  // Check if we're inside arguments (parentheses)
  let parenDepth = 0
  let braceDepth = 0
  for (let i = textBeforeCursor.length - 1; i >= 0; i--) {
    const char = textBeforeCursor[i]
    if (char === ')') parenDepth++
    else if (char === '(') {
      parenDepth--
      if (parenDepth < 0) break
    } else if (char === '}') braceDepth++
    else if (char === '{') {
      braceDepth--
      if (braceDepth < 0) break
    }
  }
  const isInsideArguments = parenDepth < 0

  // Check if we're inside a filter or orderBy argument
  const recentText = textBeforeCursor.slice(-100)
  const isInsideFilter = /filter\s*:\s*\{[^}]*$/i.test(recentText)
  const isInsideOrderBy =
    /orderBy\s*:\s*\{[^}]*$/i.test(recentText) ||
    /orderBy\s*:\s*$/i.test(recentText)

  // Get word before cursor
  const lineText = model.getLineContent(position.lineNumber)
  const textBefore = lineText.substring(0, position.column - 1).trim()
  const wordMatch = textBefore.match(/(\w+)\s*$/)
  const wordBefore = wordMatch ? wordMatch[1] : ''

  // Try to determine current type context
  let currentType: string | null = null
  const typeMatch = textBeforeCursor.match(/(\w+)\s*\{[^{}]*$/i)
  if (typeMatch) {
    currentType = typeMatch[1]
  }

  return {
    isInsideQuery,
    isInsideMutation,
    isInsideArguments,
    isInsideFilter,
    isInsideOrderBy,
    currentType,
    wordBefore,
  }
}

export function createGraphQLCompletionProvider(
  monaco: typeof Monaco,
  metadata: GraphQLSchemaMetadata
): Monaco.languages.CompletionItemProvider {
  return {
    triggerCharacters: ['{', '(', ':', ' ', '@', '.'],

    provideCompletionItems(
      model: Monaco.editor.ITextModel,
      position: Monaco.Position
    ): Monaco.languages.ProviderResult<Monaco.languages.CompletionList> {
      const context = getGraphQLContext(model, position)
      const suggestions: Monaco.languages.CompletionItem[] = []

      const word = model.getWordUntilPosition(position)
      const range: Monaco.IRange = {
        startLineNumber: position.lineNumber,
        startColumn: word.startColumn,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      }

      // Inside filter argument
      if (context.isInsideFilter) {
        // Suggest filter operators
        for (const op of FILTER_OPERATORS) {
          suggestions.push({
            label: op.name,
            kind: monaco.languages.CompletionItemKind.Property,
            detail: op.detail,
            insertText: `${op.name}: `,
            range,
          })
        }

        // Suggest column names from current table context
        if (context.currentType) {
          const tableName = context.currentType.toLowerCase()
          const table = metadata.tables.find(
            (t) =>
              t.name.toLowerCase() === tableName ||
              toCamelCase(t.name).toLowerCase() === tableName ||
              t.name.toLowerCase() + 's' === tableName || // plural
              toCamelCase(t.name + 's').toLowerCase() === tableName
          )
          if (table) {
            for (const col of table.columns) {
              suggestions.push({
                label: toCamelCase(col.name),
                kind: monaco.languages.CompletionItemKind.Field,
                detail: postgresTypeToGraphQL(col.data_type),
                insertText: `${toCamelCase(col.name)}: { `,
                range,
              })
            }
          }
        }

        return { suggestions }
      }

      // Inside orderBy argument
      if (context.isInsideOrderBy) {
        // Suggest order directions
        for (const dir of ORDER_DIRECTIONS) {
          suggestions.push({
            label: dir.name,
            kind: monaco.languages.CompletionItemKind.EnumMember,
            detail: dir.detail,
            insertText: dir.name,
            range,
          })
        }

        // Suggest column names
        if (context.currentType) {
          const tableName = context.currentType.toLowerCase()
          const table = metadata.tables.find(
            (t) =>
              t.name.toLowerCase() === tableName ||
              toCamelCase(t.name).toLowerCase() === tableName ||
              t.name.toLowerCase() + 's' === tableName ||
              toCamelCase(t.name + 's').toLowerCase() === tableName
          )
          if (table) {
            for (const col of table.columns) {
              suggestions.push({
                label: toCamelCase(col.name),
                kind: monaco.languages.CompletionItemKind.Field,
                detail: `Order by ${col.name}`,
                insertText: `${toCamelCase(col.name)}: `,
                range,
              })
            }
          }
        }

        return { suggestions }
      }

      // Inside query arguments
      if (context.isInsideArguments) {
        // Common arguments
        suggestions.push(
          {
            label: 'filter',
            kind: monaco.languages.CompletionItemKind.Property,
            detail: 'Filter results',
            insertText: 'filter: { ${1} }',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          },
          {
            label: 'orderBy',
            kind: monaco.languages.CompletionItemKind.Property,
            detail: 'Order results',
            insertText: 'orderBy: { ${1:column}: ${2:ASC} }',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          },
          {
            label: 'limit',
            kind: monaco.languages.CompletionItemKind.Property,
            detail: 'Limit number of results',
            insertText: 'limit: ${1:10}',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          },
          {
            label: 'offset',
            kind: monaco.languages.CompletionItemKind.Property,
            detail: 'Skip first N results',
            insertText: 'offset: ${1:0}',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          }
        )

        // For mutations
        if (context.isInsideMutation) {
          suggestions.push({
            label: 'input',
            kind: monaco.languages.CompletionItemKind.Property,
            detail: 'Input data for mutation',
            insertText: 'input: { ${1} }',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          })
        }

        return { suggestions }
      }

      // Inside selection set (after {)
      const lineText = model.getLineContent(position.lineNumber)
      const textBeforeCursor = lineText.substring(0, position.column - 1)

      // Check if we should suggest fields based on parent type
      if (
        context.currentType &&
        context.currentType !== 'query' &&
        context.currentType !== 'mutation'
      ) {
        const typeName = context.currentType.toLowerCase()

        // Try to find matching table
        const table = metadata.tables.find(
          (t) =>
            t.name.toLowerCase() === typeName ||
            toCamelCase(t.name).toLowerCase() === typeName ||
            t.name.toLowerCase() + 's' === typeName ||
            toCamelCase(t.name + 's').toLowerCase() === typeName
        )

        if (table) {
          // Suggest columns as fields
          for (const col of table.columns) {
            suggestions.push({
              label: toCamelCase(col.name),
              kind: monaco.languages.CompletionItemKind.Field,
              detail: postgresTypeToGraphQL(col.data_type),
              documentation: `Column: ${col.name}${col.is_nullable ? ' (nullable)' : ''}`,
              insertText: toCamelCase(col.name),
              range,
            })
          }

          return { suggestions }
        }
      }

      // Root level suggestions or inside query/mutation block
      if (
        context.isInsideQuery ||
        context.isInsideMutation ||
        !context.currentType
      ) {
        // For queries - suggest collection queries
        if (context.isInsideQuery || !context.isInsideMutation) {
          for (const table of metadata.tables) {
            const typeName = toCamelCase(table.name)
            const pluralName = typeName + 's'
            const pascalName = toPascalCase(table.name)

            // Collection query
            suggestions.push({
              label: pluralName,
              kind: monaco.languages.CompletionItemKind.Method,
              detail: `Query: [${pascalName}]`,
              documentation: `Fetch multiple ${table.name} records`,
              insertText: `${pluralName}(limit: \${1:10}) {\n  \${2:id}\n}`,
              insertTextRules:
                monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })

            // Single record query
            suggestions.push({
              label: typeName,
              kind: monaco.languages.CompletionItemKind.Method,
              detail: `Query: ${pascalName}`,
              documentation: `Fetch single ${table.name} by ID`,
              insertText: `${typeName}(id: "\${1:uuid}") {\n  \${2:id}\n}`,
              insertTextRules:
                monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })
          }
        }

        // For mutations - suggest insert/update/delete mutations
        if (context.isInsideMutation) {
          for (const table of metadata.tables) {
            const pascalName = toPascalCase(table.name)

            // Insert
            suggestions.push({
              label: `insert${pascalName}`,
              kind: monaco.languages.CompletionItemKind.Method,
              detail: `Insert: ${pascalName}`,
              documentation: `Insert a new ${table.name} record`,
              insertText: `insert${pascalName}(input: { \${1} }) {\n  \${2:id}\n}`,
              insertTextRules:
                monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })

            // Update
            suggestions.push({
              label: `update${pascalName}`,
              kind: monaco.languages.CompletionItemKind.Method,
              detail: `Update: ${pascalName}`,
              documentation: `Update ${table.name} records`,
              insertText: `update${pascalName}(filter: { id: { eq: "\${1:uuid}" } }, input: { \${2} }) {\n  \${3:id}\n}`,
              insertTextRules:
                monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })

            // Delete
            suggestions.push({
              label: `delete${pascalName}`,
              kind: monaco.languages.CompletionItemKind.Method,
              detail: `Delete: ${pascalName}`,
              documentation: `Delete ${table.name} records`,
              insertText: `delete${pascalName}(filter: { id: { eq: "\${1:uuid}" } }) {\n  \${2:id}\n}`,
              insertTextRules:
                monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
              range,
            })
          }
        }
      }

      // Always available suggestions

      // GraphQL keywords
      for (const keyword of GRAPHQL_KEYWORDS) {
        suggestions.push({
          label: keyword,
          kind: monaco.languages.CompletionItemKind.Keyword,
          detail: 'GraphQL keyword',
          insertText: keyword,
          range,
        })
      }

      // Directives (after @)
      if (textBeforeCursor.endsWith('@') || context.wordBefore === '@') {
        for (const directive of GRAPHQL_DIRECTIVES) {
          suggestions.push({
            label: directive.name,
            kind: monaco.languages.CompletionItemKind.Function,
            detail: directive.detail,
            insertText: directive.insertText,
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          })
        }
      }

      // Scalars
      for (const scalar of GRAPHQL_SCALARS) {
        suggestions.push({
          label: scalar,
          kind: monaco.languages.CompletionItemKind.TypeParameter,
          detail: 'GraphQL scalar type',
          insertText: scalar,
          range,
        })
      }

      // Query and mutation templates at root level
      if (!context.isInsideQuery && !context.isInsideMutation) {
        suggestions.push(
          {
            label: 'query',
            kind: monaco.languages.CompletionItemKind.Snippet,
            detail: 'GraphQL query template',
            insertText: 'query ${1:QueryName} {\n  ${2}\n}',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          },
          {
            label: 'mutation',
            kind: monaco.languages.CompletionItemKind.Snippet,
            detail: 'GraphQL mutation template',
            insertText: 'mutation ${1:MutationName} {\n  ${2}\n}',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          },
          {
            label: 'fragment',
            kind: monaco.languages.CompletionItemKind.Snippet,
            detail: 'GraphQL fragment template',
            insertText: 'fragment ${1:FragmentName} on ${2:Type} {\n  ${3}\n}',
            insertTextRules:
              monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          }
        )
      }

      return { suggestions }
    },
  }
}
