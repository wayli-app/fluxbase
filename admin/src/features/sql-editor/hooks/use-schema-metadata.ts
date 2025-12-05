import { useQuery } from '@tanstack/react-query'
import {
  databaseApi,
  type TableInfo,
} from '@/lib/api'

export interface SchemaMetadata {
  schemas: string[]
  tables: TableInfo[]
  isLoading: boolean
  error: Error | null
}

export function useSchemaMetadata(): SchemaMetadata {
  const {
    data: schemas = [],
    isLoading: schemasLoading,
    error: schemasError,
  } = useQuery({
    queryKey: ['sql-editor', 'schemas'],
    queryFn: () => databaseApi.getSchemas(),
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  const {
    data: tables = [],
    isLoading: tablesLoading,
    error: tablesError,
  } = useQuery({
    queryKey: ['sql-editor', 'tables'],
    queryFn: () => databaseApi.getTables(),
    staleTime: 5 * 60 * 1000, // 5 minutes
  })

  return {
    schemas,
    tables,
    isLoading: schemasLoading || tablesLoading,
    error: schemasError || tablesError,
  }
}
