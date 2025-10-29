import { useState, useEffect, useRef } from 'react'
import { Check, X } from 'lucide-react'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface EditableCellProps {
  value: unknown
  onSave: (value: unknown) => Promise<void>
  isReadOnly?: boolean
}

export function EditableCell({
  value,
  onSave,
  isReadOnly = false,
}: EditableCellProps) {
  const [isEditing, setIsEditing] = useState(false)
  const [editValue, setEditValue] = useState('')
  const [isSaving, setIsSaving] = useState(false)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    if (isEditing && inputRef.current) {
      inputRef.current.focus()
      inputRef.current.select()
    }
  }, [isEditing])

  const handleStartEdit = () => {
    if (isReadOnly) return

    // Convert value to string for editing
    let stringValue = ''
    if (value === null || value === undefined) {
      stringValue = ''
    } else if (typeof value === 'object') {
      stringValue = JSON.stringify(value)
    } else {
      stringValue = String(value)
    }

    setEditValue(stringValue)
    setIsEditing(true)
  }

  const handleCancel = () => {
    setIsEditing(false)
    setEditValue('')
  }

  const handleSave = async () => {
    setIsSaving(true)
    try {
      // Convert string back to appropriate type
      let processedValue: unknown = editValue

      // Handle null/empty
      if (editValue === '' || editValue.toLowerCase() === 'null') {
        processedValue = null
      }
      // Try to parse as JSON for objects/arrays
      else if (editValue.startsWith('{') || editValue.startsWith('[')) {
        try {
          processedValue = JSON.parse(editValue)
        } catch {
          // If parsing fails, keep as string
        }
      }
      // Try to parse as number
      else if (!isNaN(Number(editValue)) && editValue !== '') {
        processedValue = Number(editValue)
      }
      // Try to parse as boolean
      else if (editValue === 'true' || editValue === 'false') {
        processedValue = editValue === 'true'
      }

      await onSave(processedValue)
      setIsEditing(false)
    } catch (error) {
      console.error('Failed to save:', error)
    } finally {
      setIsSaving(false)
    }
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSave()
    } else if (e.key === 'Escape') {
      handleCancel()
    }
  }

  if (isEditing) {
    return (
      <div className='flex items-center gap-1'>
        <Input
          ref={inputRef}
          value={editValue}
          onChange={(e) => setEditValue(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isSaving}
          className='h-8 text-sm'
        />
        <Button
          size='sm'
          variant='ghost'
          className='size-8 p-0'
          onClick={handleSave}
          disabled={isSaving}
        >
          <Check className='size-4 text-green-600' />
        </Button>
        <Button
          size='sm'
          variant='ghost'
          className='size-8 p-0'
          onClick={handleCancel}
          disabled={isSaving}
        >
          <X className='size-4 text-red-600' />
        </Button>
      </div>
    )
  }

  // Format display value
  let displayValue: string
  if (value === null || value === undefined) {
    displayValue = 'NULL'
  } else if (typeof value === 'boolean') {
    displayValue = value ? 'true' : 'false'
  } else if (typeof value === 'object') {
    displayValue = JSON.stringify(value)
  } else {
    displayValue = String(value)
  }

  // Truncate long values
  const shouldTruncate = displayValue.length > 50
  const truncatedValue = shouldTruncate
    ? displayValue.slice(0, 50) + '...'
    : displayValue

  return (
    <button
      onClick={handleStartEdit}
      disabled={isReadOnly}
      className={cn(
        'w-full text-left hover:bg-accent hover:text-accent-foreground rounded px-2 py-1 -mx-2 -my-1 transition-colors',
        value === null && 'text-muted-foreground italic',
        isReadOnly && 'cursor-default hover:bg-transparent'
      )}
      title={shouldTruncate ? displayValue : undefined}
    >
      {truncatedValue}
    </button>
  )
}
