import type * as Monaco from 'monaco-editor'
import type { TableInfo, RPCFunction } from '@/lib/api'

// SQL keywords for autocompletion
const SQL_KEYWORDS = [
  'SELECT', 'FROM', 'WHERE', 'AND', 'OR', 'NOT', 'IN', 'LIKE', 'ILIKE',
  'BETWEEN', 'IS', 'NULL', 'TRUE', 'FALSE', 'AS', 'ON', 'JOIN', 'LEFT',
  'RIGHT', 'INNER', 'OUTER', 'FULL', 'CROSS', 'NATURAL', 'USING',
  'GROUP', 'BY', 'HAVING', 'ORDER', 'ASC', 'DESC', 'NULLS', 'FIRST', 'LAST',
  'LIMIT', 'OFFSET', 'FETCH', 'NEXT', 'ROWS', 'ONLY',
  'INSERT', 'INTO', 'VALUES', 'DEFAULT', 'RETURNING',
  'UPDATE', 'SET',
  'DELETE',
  'CREATE', 'TABLE', 'INDEX', 'VIEW', 'SCHEMA', 'TYPE', 'FUNCTION',
  'ALTER', 'DROP', 'TRUNCATE', 'CASCADE', 'RESTRICT',
  'PRIMARY', 'KEY', 'FOREIGN', 'REFERENCES', 'UNIQUE', 'CHECK', 'CONSTRAINT',
  'WITH', 'RECURSIVE', 'UNION', 'INTERSECT', 'EXCEPT', 'ALL', 'DISTINCT',
  'CASE', 'WHEN', 'THEN', 'ELSE', 'END',
  'CAST', 'COALESCE', 'NULLIF', 'GREATEST', 'LEAST',
  'EXISTS', 'ANY', 'SOME',
  'BEGIN', 'COMMIT', 'ROLLBACK', 'TRANSACTION',
  'GRANT', 'REVOKE', 'TO', 'PUBLIC',
]

// Common PostgreSQL functions
const POSTGRESQL_FUNCTIONS = [
  // Aggregate functions
  { name: 'COUNT', detail: 'count(*) or count(expression)', insertText: 'COUNT(${1:*})' },
  { name: 'SUM', detail: 'sum(expression)', insertText: 'SUM(${1:column})' },
  { name: 'AVG', detail: 'avg(expression)', insertText: 'AVG(${1:column})' },
  { name: 'MIN', detail: 'min(expression)', insertText: 'MIN(${1:column})' },
  { name: 'MAX', detail: 'max(expression)', insertText: 'MAX(${1:column})' },
  { name: 'ARRAY_AGG', detail: 'array_agg(expression)', insertText: 'ARRAY_AGG(${1:column})' },
  { name: 'STRING_AGG', detail: 'string_agg(expression, delimiter)', insertText: 'STRING_AGG(${1:column}, ${2:\', \'})' },
  { name: 'JSON_AGG', detail: 'json_agg(expression)', insertText: 'JSON_AGG(${1:column})' },
  { name: 'JSONB_AGG', detail: 'jsonb_agg(expression)', insertText: 'JSONB_AGG(${1:column})' },

  // String functions
  { name: 'CONCAT', detail: 'concat(str1, str2, ...)', insertText: 'CONCAT(${1:str1}, ${2:str2})' },
  { name: 'CONCAT_WS', detail: 'concat_ws(separator, str1, str2, ...)', insertText: 'CONCAT_WS(${1:\', \'}, ${2:str1}, ${3:str2})' },
  { name: 'LENGTH', detail: 'length(string)', insertText: 'LENGTH(${1:string})' },
  { name: 'LOWER', detail: 'lower(string)', insertText: 'LOWER(${1:string})' },
  { name: 'UPPER', detail: 'upper(string)', insertText: 'UPPER(${1:string})' },
  { name: 'TRIM', detail: 'trim(string)', insertText: 'TRIM(${1:string})' },
  { name: 'LTRIM', detail: 'ltrim(string)', insertText: 'LTRIM(${1:string})' },
  { name: 'RTRIM', detail: 'rtrim(string)', insertText: 'RTRIM(${1:string})' },
  { name: 'SUBSTRING', detail: 'substring(string, start, length)', insertText: 'SUBSTRING(${1:string}, ${2:1}, ${3:10})' },
  { name: 'REPLACE', detail: 'replace(string, from, to)', insertText: 'REPLACE(${1:string}, ${2:from}, ${3:to})' },
  { name: 'SPLIT_PART', detail: 'split_part(string, delimiter, position)', insertText: 'SPLIT_PART(${1:string}, ${2:\',\'}, ${3:1})' },
  { name: 'REGEXP_REPLACE', detail: 'regexp_replace(string, pattern, replacement)', insertText: 'REGEXP_REPLACE(${1:string}, ${2:pattern}, ${3:replacement})' },

  // Date/Time functions
  { name: 'NOW', detail: 'now() - current timestamp', insertText: 'NOW()' },
  { name: 'CURRENT_DATE', detail: 'current date', insertText: 'CURRENT_DATE' },
  { name: 'CURRENT_TIME', detail: 'current time', insertText: 'CURRENT_TIME' },
  { name: 'CURRENT_TIMESTAMP', detail: 'current timestamp', insertText: 'CURRENT_TIMESTAMP' },
  { name: 'DATE_TRUNC', detail: 'date_trunc(field, source)', insertText: 'DATE_TRUNC(${1:\'day\'}, ${2:timestamp})' },
  { name: 'DATE_PART', detail: 'date_part(field, source)', insertText: 'DATE_PART(${1:\'year\'}, ${2:timestamp})' },
  { name: 'EXTRACT', detail: 'extract(field from source)', insertText: 'EXTRACT(${1:YEAR} FROM ${2:timestamp})' },
  { name: 'AGE', detail: 'age(timestamp1, timestamp2)', insertText: 'AGE(${1:timestamp1}, ${2:timestamp2})' },
  { name: 'TO_CHAR', detail: 'to_char(timestamp, format)', insertText: 'TO_CHAR(${1:timestamp}, ${2:\'YYYY-MM-DD\'})' },
  { name: 'TO_DATE', detail: 'to_date(string, format)', insertText: 'TO_DATE(${1:string}, ${2:\'YYYY-MM-DD\'})' },
  { name: 'TO_TIMESTAMP', detail: 'to_timestamp(string, format)', insertText: 'TO_TIMESTAMP(${1:string}, ${2:\'YYYY-MM-DD HH24:MI:SS\'})' },

  // JSON functions
  { name: 'JSON_BUILD_OBJECT', detail: 'json_build_object(key1, val1, ...)', insertText: 'JSON_BUILD_OBJECT(${1:\'key\'}, ${2:value})' },
  { name: 'JSONB_BUILD_OBJECT', detail: 'jsonb_build_object(key1, val1, ...)', insertText: 'JSONB_BUILD_OBJECT(${1:\'key\'}, ${2:value})' },
  { name: 'JSON_ARRAY_LENGTH', detail: 'json_array_length(json)', insertText: 'JSON_ARRAY_LENGTH(${1:json_column})' },
  { name: 'JSONB_ARRAY_LENGTH', detail: 'jsonb_array_length(jsonb)', insertText: 'JSONB_ARRAY_LENGTH(${1:jsonb_column})' },
  { name: 'TO_JSON', detail: 'to_json(value)', insertText: 'TO_JSON(${1:value})' },
  { name: 'TO_JSONB', detail: 'to_jsonb(value)', insertText: 'TO_JSONB(${1:value})' },

  // Utility functions
  { name: 'COALESCE', detail: 'coalesce(val1, val2, ...) - first non-null', insertText: 'COALESCE(${1:column}, ${2:default})' },
  { name: 'NULLIF', detail: 'nullif(val1, val2) - null if equal', insertText: 'NULLIF(${1:val1}, ${2:val2})' },
  { name: 'GREATEST', detail: 'greatest(val1, val2, ...)', insertText: 'GREATEST(${1:val1}, ${2:val2})' },
  { name: 'LEAST', detail: 'least(val1, val2, ...)', insertText: 'LEAST(${1:val1}, ${2:val2})' },
  { name: 'GENERATE_SERIES', detail: 'generate_series(start, stop, step)', insertText: 'GENERATE_SERIES(${1:1}, ${2:10}, ${3:1})' },
  { name: 'ROW_NUMBER', detail: 'row_number() over(...)', insertText: 'ROW_NUMBER() OVER (${1:ORDER BY column})' },
  { name: 'RANK', detail: 'rank() over(...)', insertText: 'RANK() OVER (${1:ORDER BY column})' },
  { name: 'DENSE_RANK', detail: 'dense_rank() over(...)', insertText: 'DENSE_RANK() OVER (${1:ORDER BY column})' },

  // Type casting
  { name: 'CAST', detail: 'cast(value as type)', insertText: 'CAST(${1:value} AS ${2:type})' },

  // UUID
  { name: 'GEN_RANDOM_UUID', detail: 'gen_random_uuid() - generate UUID', insertText: 'GEN_RANDOM_UUID()' },
]

export interface SchemaMetadataForCompletion {
  schemas: string[]
  tables: TableInfo[]
  functions: RPCFunction[]
}

// Extract table aliases from SQL using regex
function extractTableAliases(sql: string, tables: TableInfo[]): Map<string, TableInfo> {
  const aliases = new Map<string, TableInfo>()

  // Match FROM/JOIN clauses with optional schema, table name, and optional alias
  // Examples: FROM users, FROM public.users, FROM users u, FROM users AS u
  const fromPattern = /(?:FROM|JOIN)\s+(?:(\w+)\.)?(\w+)(?:\s+(?:AS\s+)?(\w+))?/gi
  let match
  while ((match = fromPattern.exec(sql)) !== null) {
    const schema = match[1] || 'public'
    const tableName = match[2]
    const alias = match[3]

    // Find matching table in metadata
    const matchingTable = tables.find(t =>
      t.name.toLowerCase() === tableName.toLowerCase() &&
      t.schema.toLowerCase() === schema.toLowerCase()
    )

    if (matchingTable) {
      // Add table name as key
      if (!aliases.has(tableName.toLowerCase())) {
        aliases.set(tableName.toLowerCase(), matchingTable)
      }
      // Add alias as key if present
      if (alias && !aliases.has(alias.toLowerCase())) {
        aliases.set(alias.toLowerCase(), matchingTable)
      }
    }
  }

  return aliases
}

// Get context from cursor position
function getCompletionContext(
  model: Monaco.editor.ITextModel,
  position: Monaco.Position
): { prefix: string; wordBefore: string } {
  const lineText = model.getLineContent(position.lineNumber)
  const textBeforeCursor = lineText.substring(0, position.column - 1)

  // Get the word being typed
  const wordMatch = textBeforeCursor.match(/[\w.]*$/)
  const prefix = wordMatch ? wordMatch[0] : ''

  // Get the word before the current word (to understand context)
  const beforePrefix = textBeforeCursor.substring(0, textBeforeCursor.length - prefix.length).trim()
  const wordBeforeMatch = beforePrefix.match(/(\w+)\s*$/)
  const wordBefore = wordBeforeMatch ? wordBeforeMatch[1].toUpperCase() : ''

  return { prefix, wordBefore }
}

export function createSqlCompletionProvider(
  monaco: typeof Monaco,
  metadata: SchemaMetadataForCompletion
): Monaco.languages.CompletionItemProvider {
  return {
    triggerCharacters: ['.', ' ', '('],

    provideCompletionItems(
      model: Monaco.editor.ITextModel,
      position: Monaco.Position
    ): Monaco.languages.ProviderResult<Monaco.languages.CompletionList> {
      const { prefix, wordBefore } = getCompletionContext(model, position)
      const suggestions: Monaco.languages.CompletionItem[] = []

      // Get the full SQL text up to cursor for context
      const fullText = model.getValue()
      const offset = model.getOffsetAt(position)
      const sqlBeforeCursor = fullText.substring(0, offset)

      // Extract table aliases from the SQL
      const tableAliases = extractTableAliases(sqlBeforeCursor, metadata.tables)

      // Determine what kind of completions to show
      const hasDot = prefix.includes('.')
      const dotParts = prefix.split('.')

      // Get the range to replace
      // When there's a dot, only replace the part after the last dot (e.g., for "schema.tab", only replace "tab")
      const word = model.getWordUntilPosition(position)
      const afterDot = hasDot ? dotParts[dotParts.length - 1] : ''
      const range: Monaco.IRange = {
        startLineNumber: position.lineNumber,
        startColumn: hasDot ? position.column - afterDot.length : word.startColumn,
        endLineNumber: position.lineNumber,
        endColumn: position.column,
      }

      // After schema. or table. or alias.
      if (hasDot && dotParts.length === 2) {
        const qualifier = dotParts[0].toLowerCase()

        // Check if it's a schema
        const isSchema = metadata.schemas.some(s => s.toLowerCase() === qualifier)

        if (isSchema) {
          // Suggest tables in this schema
          const tablesInSchema = metadata.tables.filter(t => t.schema.toLowerCase() === qualifier)
          for (const table of tablesInSchema) {
            suggestions.push({
              label: table.name,
              kind: monaco.languages.CompletionItemKind.Class,
              detail: `Table in ${table.schema}`,
              insertText: table.name,
              range,
            })
          }
        }

        // Check if it's a table name or alias
        const aliasedTable = tableAliases.get(qualifier)
        if (aliasedTable) {
          // Suggest columns from this table
          for (const column of aliasedTable.columns) {
            suggestions.push({
              label: column.name,
              kind: monaco.languages.CompletionItemKind.Field,
              detail: `${column.data_type}${column.is_nullable ? ' (nullable)' : ''}`,
              documentation: `Column from ${aliasedTable.schema}.${aliasedTable.name}`,
              insertText: column.name,
              range,
            })
          }
        }

        return { suggestions }
      }

      // Context-specific completions
      const tableContextKeywords = ['FROM', 'JOIN', 'INTO', 'UPDATE', 'TABLE']
      const columnContextKeywords = ['SELECT', 'WHERE', 'AND', 'OR', 'ON', 'SET', 'BY', 'HAVING']

      if (tableContextKeywords.includes(wordBefore)) {
        // Suggest schemas first (user can type schema. to see tables)
        for (const schema of metadata.schemas) {
          suggestions.push({
            label: schema,
            kind: monaco.languages.CompletionItemKind.Module,
            detail: 'Schema',
            documentation: `Type ${schema}. to see tables in this schema`,
            insertText: schema,
            range,
          })
        }

        // Suggest tables (with schema prefix for non-public)
        for (const table of metadata.tables) {
          const displayName = table.schema === 'public' ? table.name : `${table.schema}.${table.name}`
          suggestions.push({
            label: displayName,
            kind: monaco.languages.CompletionItemKind.Class,
            detail: `Table in ${table.schema}`,
            documentation: `Columns: ${table.columns.map(c => c.name).join(', ')}`,
            insertText: displayName,
            range,
          })
        }
      } else if (columnContextKeywords.includes(wordBefore)) {
        // Suggest columns from all tables in the query
        const suggestedColumns = new Set<string>()

        for (const [alias, table] of tableAliases) {
          for (const column of table.columns) {
            // Suggest with alias prefix if there are multiple tables
            const columnLabel = tableAliases.size > 1 ? `${alias}.${column.name}` : column.name
            if (!suggestedColumns.has(columnLabel)) {
              suggestedColumns.add(columnLabel)
              suggestions.push({
                label: columnLabel,
                kind: monaco.languages.CompletionItemKind.Field,
                detail: `${column.data_type} from ${table.name}`,
                insertText: columnLabel,
                range,
              })
            }
          }
        }

        // If no tables in query yet, suggest columns from all tables
        if (tableAliases.size === 0) {
          for (const table of metadata.tables) {
            for (const column of table.columns) {
              const columnLabel = column.name
              if (!suggestedColumns.has(columnLabel)) {
                suggestedColumns.add(columnLabel)
                suggestions.push({
                  label: columnLabel,
                  kind: monaco.languages.CompletionItemKind.Field,
                  detail: `${column.data_type} from ${table.schema}.${table.name}`,
                  insertText: columnLabel,
                  range,
                })
              }
            }
          }
        }

        // Also suggest functions in column context
        for (const func of POSTGRESQL_FUNCTIONS) {
          suggestions.push({
            label: func.name,
            kind: monaco.languages.CompletionItemKind.Function,
            detail: func.detail,
            insertText: func.insertText,
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          })
        }
      } else {
        // General context - suggest everything

        // SQL Keywords
        for (const keyword of SQL_KEYWORDS) {
          if (keyword.toLowerCase().startsWith(prefix.toLowerCase())) {
            suggestions.push({
              label: keyword,
              kind: monaco.languages.CompletionItemKind.Keyword,
              detail: 'SQL Keyword',
              insertText: keyword,
              range,
            })
          }
        }

        // Schemas
        for (const schema of metadata.schemas) {
          suggestions.push({
            label: schema,
            kind: monaco.languages.CompletionItemKind.Module,
            detail: 'Schema',
            insertText: schema,
            range,
          })
        }

        // Tables
        for (const table of metadata.tables) {
          const displayName = table.schema === 'public' ? table.name : `${table.schema}.${table.name}`
          suggestions.push({
            label: displayName,
            kind: monaco.languages.CompletionItemKind.Class,
            detail: `Table in ${table.schema}`,
            insertText: displayName,
            range,
          })
        }

        // PostgreSQL functions
        for (const func of POSTGRESQL_FUNCTIONS) {
          suggestions.push({
            label: func.name,
            kind: monaco.languages.CompletionItemKind.Function,
            detail: func.detail,
            insertText: func.insertText,
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          })
        }

        // RPC Functions from database
        for (const func of metadata.functions) {
          const params = (func.parameters || [])
            .filter(p => p.mode === 'IN' || p.mode === 'INOUT')
            .map((p, i) => `\${${i + 1}:${p.name}}`)
            .join(', ')

          const paramDoc = (func.parameters || []).length > 0
            ? `Parameters: ${func.parameters.map(p => `${p.name}: ${p.type}`).join(', ')}`
            : 'No parameters'

          suggestions.push({
            label: func.name,
            kind: monaco.languages.CompletionItemKind.Function,
            detail: `${func.return_type} - ${func.schema}`,
            documentation: func.description || paramDoc,
            insertText: `${func.name}(${params})`,
            insertTextRules: monaco.languages.CompletionItemInsertTextRule.InsertAsSnippet,
            range,
          })
        }
      }

      return { suggestions }
    },
  }
}
