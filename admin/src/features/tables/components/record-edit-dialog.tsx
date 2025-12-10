import { useEffect, useState, useMemo } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useInsert, useUpdate } from '@fluxbase/sdk-react'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetFooter,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { Label } from '@/components/ui/label'
import { toast } from 'sonner'

interface TableColumn {
  name: string
  data_type: string
  is_nullable: boolean
  default_value: string | null
  is_primary_key: boolean
}

interface RecordEditDialogProps {
  tableName: string
  tableDisplayName: string
  tableSchema?: TableColumn[]
  record: Record<string, unknown> | null
  isOpen: boolean
  onClose: () => void
  isCreate?: boolean
}

export function RecordEditDialog({
  tableName,
  tableDisplayName,
  tableSchema = [],
  record,
  isOpen,
  onClose,
  isCreate = false,
}: RecordEditDialogProps) {
  const queryClient = useQueryClient()

  const initialFormData = useMemo(() => {
    if (record && !isCreate) {
      // Convert record values to strings for form inputs
      const stringData: Record<string, string> = {}
      Object.entries(record).forEach(([key, value]) => {
        if (value !== null && value !== undefined) {
          stringData[key] =
            typeof value === 'object' ? JSON.stringify(value) : String(value)
        }
      })
      return stringData
    } else if (isCreate && tableSchema.length > 0) {
      // Initialize form with table schema columns and defaults
      const stringData: Record<string, string> = {}
      tableSchema.forEach((col) => {
        if (col.default_value) {
          // Show the default value as a placeholder hint
          stringData[col.name] = ''
        } else if (!col.is_nullable) {
          // Required fields should be in form
          stringData[col.name] = ''
        }
      })
      return stringData
    }
    return {}
  }, [record, isCreate, tableSchema])

  const [formData, setFormData] = useState<Record<string, string>>(initialFormData)

  useEffect(() => {
    setFormData(initialFormData)
  }, [initialFormData])

  const insertMutation = useInsert(tableName)
  const updateFluxbase = useUpdate(tableName)

  const createMutation = {
    mutateAsync: async (data: Record<string, unknown>) => {
      try {
        await insertMutation.mutateAsync(data)
        queryClient.invalidateQueries({ queryKey: ['table-data', tableDisplayName] })
        queryClient.invalidateQueries({ queryKey: ['table-count', tableDisplayName] })
        toast.success('Record created successfully')
        onClose()
      } catch (error) {
        toast.error(`Failed to create record: ${(error as Error).message}`)
        throw error
      }
    },
    mutate: (data: Record<string, unknown>) => {
      insertMutation.mutateAsync(data)
        .then(() => {
          queryClient.invalidateQueries({ queryKey: ['table-data', tableDisplayName] })
          queryClient.invalidateQueries({ queryKey: ['table-count', tableDisplayName] })
          toast.success('Record created successfully')
          onClose()
        })
        .catch((error) => {
          toast.error(`Failed to create record: ${(error as Error).message}`)
        })
    },
    isPending: insertMutation.isPending,
  }

  const updateMutation = {
    mutateAsync: async (data: Record<string, unknown>) => {
      try {
        await updateFluxbase.mutateAsync({
          data,
          buildQuery: (q) => q.eq('id', record!.id),
        })
        queryClient.invalidateQueries({ queryKey: ['table-data', tableDisplayName] })
        toast.success('Record updated successfully')
        onClose()
      } catch (error) {
        toast.error(`Failed to update record: ${(error as Error).message}`)
        throw error
      }
    },
    mutate: (data: Record<string, unknown>) => {
      updateFluxbase.mutateAsync({
        data,
        buildQuery: (q) => q.eq('id', record!.id),
      })
        .then(() => {
          queryClient.invalidateQueries({ queryKey: ['table-data', tableDisplayName] })
          toast.success('Record updated successfully')
          onClose()
        })
        .catch((error) => {
          toast.error(`Failed to update record: ${(error as Error).message}`)
        })
    },
    isPending: updateFluxbase.isPending,
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    // Convert string values back to appropriate types
    const processedData: Record<string, unknown> = {}
    Object.entries(formData).forEach(([key, value]) => {
      const colSchema = tableSchema.find(col => col.name === key)

      // Skip empty values for columns with defaults (let DB handle it)
      if (value === '' && colSchema?.default_value) {
        return
      }

      // Skip primary key fields with defaults (auto-generated)
      if (colSchema?.is_primary_key && colSchema?.default_value) {
        return
      }

      // Skip empty values entirely - don't send them
      if (value === '') {
        return
      }

      // Try to parse as JSON for objects/arrays
      if (value.startsWith('{') || value.startsWith('[')) {
        try {
          processedData[key] = JSON.parse(value)
          return
        } catch {
          // If parsing fails, treat as string
        }
      }
      // Try to parse as number
      if (!isNaN(Number(value))) {
        processedData[key] = Number(value)
        return
      }
      // Try to parse as boolean
      if (value === 'true' || value === 'false') {
        processedData[key] = value === 'true'
        return
      }
      // Default to string
      processedData[key] = value
    })

    if (isCreate) {
      createMutation.mutate(processedData)
    } else {
      updateMutation.mutate(processedData)
    }
  }

  const handleChange = (key: string, value: string) => {
    setFormData((prev) => ({ ...prev, [key]: value }))
  }

  const isLoading = createMutation.isPending || updateMutation.isPending

  return (
    <Sheet open={isOpen} onOpenChange={onClose}>
      <SheetContent className='w-full sm:max-w-xl overflow-y-auto p-6'>
        <SheetHeader className='mb-6'>
          <SheetTitle>
            {isCreate ? 'Create New Record' : 'Edit Record'}
          </SheetTitle>
          <SheetDescription>
            {isCreate
              ? `Add a new record to ${tableName}`
              : `Update record in ${tableName}`}
          </SheetDescription>
        </SheetHeader>

        <form onSubmit={handleSubmit} className='flex flex-col gap-6'>
          <div className='space-y-4'>
              {Object.entries(formData).map(([key, value]) => {
                const colSchema = tableSchema.find(col => col.name === key)
                const defaultHint = colSchema?.default_value ? `Default: ${colSchema.default_value}` : ''
                const typeHint = colSchema?.data_type || ''
                const isRequired = colSchema ? !colSchema.is_nullable && !colSchema.default_value : false

                return (
                  <div key={key} className='space-y-2'>
                    <div className='flex flex-col gap-0.5'>
                      <Label htmlFor={key} className='flex items-center gap-1.5'>
                        {key}
                        {isRequired && <span className='text-destructive'>*</span>}
                      </Label>
                      {typeHint && (
                        <span className='text-xs text-muted-foreground'>{typeHint}</span>
                      )}
                    </div>
                    {colSchema?.data_type === 'text' ||
                    colSchema?.data_type === 'json' ||
                    colSchema?.data_type === 'jsonb' ? (
                      <Textarea
                        id={key}
                        value={value}
                        onChange={(e) => handleChange(key, e.target.value)}
                        disabled={colSchema?.is_primary_key && colSchema.default_value !== null}
                        placeholder={defaultHint || `Enter ${key}`}
                        className='min-h-[100px] font-mono text-sm'
                      />
                    ) : (
                      <Input
                        id={key}
                        value={value}
                        onChange={(e) => handleChange(key, e.target.value)}
                        disabled={colSchema?.is_primary_key && colSchema.default_value !== null}
                        placeholder={defaultHint || `Enter ${key}`}
                      />
                    )}
                    {defaultHint && value === '' && (
                      <p className='text-xs text-muted-foreground'>{defaultHint}</p>
                    )}
                  </div>
                )
              })}
          </div>

          <SheetFooter className='flex-row gap-2 pt-4'>
            <Button type='button' variant='outline' onClick={onClose} className='flex-1'>
              Cancel
            </Button>
            <Button type='submit' disabled={isLoading} className='flex-1'>
              {isLoading
                ? 'Saving...'
                : isCreate
                  ? 'Create'
                  : 'Update'}
            </Button>
          </SheetFooter>
        </form>
      </SheetContent>
    </Sheet>
  )
}
