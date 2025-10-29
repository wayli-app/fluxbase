import { useEffect, useState } from 'react'
import { useQueryClient } from '@tanstack/react-query'
import { useInsert, useUpdate } from '@fluxbase/sdk-react'
import { X } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { ScrollArea } from '@/components/ui/scroll-area'
import { toast } from 'sonner'

interface RecordEditDialogProps {
  tableName: string
  record: Record<string, unknown> | null
  isOpen: boolean
  onClose: () => void
  isCreate?: boolean
}

export function RecordEditDialog({
  tableName,
  record,
  isOpen,
  onClose,
  isCreate = false,
}: RecordEditDialogProps) {
  const queryClient = useQueryClient()
  const [formData, setFormData] = useState<Record<string, string>>({})

  useEffect(() => {
    if (record && !isCreate) {
      // Convert record values to strings for form inputs
      const stringData: Record<string, string> = {}
      Object.entries(record).forEach(([key, value]) => {
        if (value !== null && value !== undefined) {
          stringData[key] =
            typeof value === 'object' ? JSON.stringify(value) : String(value)
        }
      })
      setFormData(stringData)
    } else if (isCreate) {
      setFormData({})
    }
  }, [record, isCreate])

  const insertMutation = useInsert(tableName)
  const updateFluxbase = useUpdate(tableName)

  const createMutation = {
    mutateAsync: async (data: Record<string, unknown>) => {
      try {
        await insertMutation.mutateAsync(data)
        queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
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
          queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
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
        queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
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
          queryClient.invalidateQueries({ queryKey: ['table-data', tableName] })
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
      if (!isNaN(Number(value)) && value !== '') {
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

  const handleAddField = () => {
    const fieldName = prompt('Enter field name:')
    if (fieldName && !formData[fieldName]) {
      setFormData((prev) => ({ ...prev, [fieldName]: '' }))
    }
  }

  const handleRemoveField = (key: string) => {
    setFormData((prev) => {
      const newData = { ...prev }
      delete newData[key]
      return newData
    })
  }

  const isLoading = createMutation.isPending || updateMutation.isPending

  return (
    <Dialog open={isOpen} onOpenChange={onClose}>
      <DialogContent className='max-h-[85vh] max-w-2xl flex flex-col'>
        <DialogHeader className='flex-shrink-0'>
          <DialogTitle>
            {isCreate ? 'Create New Record' : 'Edit Record'}
          </DialogTitle>
          <DialogDescription>
            {isCreate
              ? `Add a new record to ${tableName}`
              : `Update record in ${tableName}`}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit} className='flex flex-1 flex-col overflow-hidden'>
          <ScrollArea className='flex-1 pr-4'>
            <div className='space-y-4 py-4'>
              {Object.entries(formData).map(([key, value]) => (
                <div key={key} className='space-y-2'>
                  <div className='flex items-center justify-between'>
                    <Label htmlFor={key}>{key}</Label>
                    {isCreate && key !== 'id' && (
                      <Button
                        type='button'
                        variant='ghost'
                        size='sm'
                        onClick={() => handleRemoveField(key)}
                      >
                        <X className='size-4' />
                      </Button>
                    )}
                  </div>
                  <Input
                    id={key}
                    value={value}
                    onChange={(e) => handleChange(key, e.target.value)}
                    disabled={key === 'id' && !isCreate}
                    placeholder={`Enter ${key}`}
                  />
                </div>
              ))}

              {isCreate && (
                <Button
                  type='button'
                  variant='outline'
                  onClick={handleAddField}
                  className='w-full'
                >
                  Add Field
                </Button>
              )}
            </div>
          </ScrollArea>

          <DialogFooter className='mt-4 flex-shrink-0'>
            <Button type='button' variant='outline' onClick={onClose}>
              Cancel
            </Button>
            <Button type='submit' disabled={isLoading}>
              {isLoading
                ? 'Saving...'
                : isCreate
                  ? 'Create'
                  : 'Update'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
