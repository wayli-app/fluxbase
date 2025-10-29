import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Database } from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { ScrollArea } from '@/components/ui/scroll-area'
import { Skeleton } from '@/components/ui/skeleton'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { databaseApi } from '@/lib/api'

interface TableSelectorProps {
  selectedTable?: string
  onTableSelect: (table: string) => void
}

export function TableSelector({
  selectedTable,
  onTableSelect,
}: TableSelectorProps) {
  const [selectedSchema, setSelectedSchema] = useState<string>('public')

  const { data: schemas, isLoading: schemasLoading } = useQuery({
    queryKey: ['schemas'],
    queryFn: databaseApi.getSchemas,
  })

  const { data: tables, isLoading: tablesLoading } = useQuery({
    queryKey: ['tables'],
    queryFn: databaseApi.getTables,
  })

  const isLoading = schemasLoading || tablesLoading

  if (isLoading) {
    return (
      <div className='space-y-2 p-4'>
        <Skeleton className='h-4 w-32' />
        <Skeleton className='h-10 w-full' />
        {[...Array(5)].map((_, i) => (
          <Skeleton key={i} className='h-9 w-full' />
        ))}
      </div>
    )
  }

  // Group tables by schema
  const groupedTables = (tables || []).reduce(
    (acc, table) => {
      const [schema, name] = table.includes('.')
        ? table.split('.')
        : ['public', table]
      if (!acc[schema]) acc[schema] = []
      acc[schema].push({ full: table, name })
      return acc
    },
    {} as Record<string, Array<{ full: string; name: string }>>
  )

  // Show only selected schema (no "all" option)
  const filteredTables = groupedTables[selectedSchema] || []

  return (
    <div className='flex h-full flex-col border-r'>
      <div className='border-b p-4'>
        <h2 className='flex items-center gap-2 text-lg font-semibold mb-3'>
          <Database className='size-5' />
          Tables
        </h2>
        <Select value={selectedSchema} onValueChange={setSelectedSchema}>
          <SelectTrigger className='w-full'>
            <SelectValue placeholder='Select schema' />
          </SelectTrigger>
          <SelectContent>
            {(schemas || []).map((schema) => (
              <SelectItem key={schema} value={schema}>
                {schema}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>
      </div>
      <ScrollArea className='flex-1'>
        <div className='p-2'>
          <div className='space-y-1'>
            {filteredTables.map(({ full, name }) => (
              <Button
                key={full}
                variant={selectedTable === full ? 'secondary' : 'ghost'}
                className={cn(
                  'w-full justify-start font-normal',
                  selectedTable === full && 'bg-secondary'
                )}
                onClick={() => onTableSelect(full)}
              >
                {name}
              </Button>
            ))}
          </div>
        </div>
      </ScrollArea>
    </div>
  )
}
