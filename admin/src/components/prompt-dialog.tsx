import { useState, useEffect, useRef } from 'react'
import { cn } from '@/lib/utils'
import {
  AlertDialog,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'

type PromptDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: React.ReactNode
  description?: React.ReactNode
  placeholder?: string
  defaultValue?: string
  inputType?: 'text' | 'email' | 'number'
  confirmText?: string
  cancelText?: string
  onConfirm: (value: string) => void
  isLoading?: boolean
  disabled?: boolean
  validation?: (value: string) => string | null
  className?: string
}

export function PromptDialog(props: PromptDialogProps) {
  const {
    open,
    onOpenChange,
    title,
    description,
    placeholder,
    defaultValue = '',
    inputType = 'text',
    confirmText = 'Continue',
    cancelText = 'Cancel',
    onConfirm,
    isLoading = false,
    disabled = false,
    validation,
    className,
  } = props

  const [value, setValue] = useState(defaultValue)
  const [error, setError] = useState<string | null>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  const prevOpenRef = useRef(open)

  // Reset form state when dialog opens (transition from closed to open)
  useEffect(() => {
    const justOpened = open && !prevOpenRef.current
    prevOpenRef.current = open

    if (justOpened) {
      // Use microtask to avoid synchronous setState cascading render warning
      queueMicrotask(() => {
        setValue(defaultValue)
        setError(null)
      })
      // Focus input after dialog animation
      setTimeout(() => inputRef.current?.focus(), 0)
    }
  }, [open, defaultValue])

  const handleConfirm = () => {
    if (validation) {
      const validationError = validation(value)
      if (validationError) {
        setError(validationError)
        return
      }
    }
    onConfirm(value)
  }

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === 'Enter' && !disabled && !isLoading) {
      e.preventDefault()
      handleConfirm()
    }
  }

  const handleValueChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue(e.target.value)
    if (error) setError(null)
  }

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent className={cn(className)}>
        <AlertDialogHeader className='text-start'>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          {description && (
            <AlertDialogDescription asChild>
              <div>{description}</div>
            </AlertDialogDescription>
          )}
        </AlertDialogHeader>
        <div className='py-2'>
          <Input
            ref={inputRef}
            type={inputType}
            value={value}
            onChange={handleValueChange}
            onKeyDown={handleKeyDown}
            placeholder={placeholder}
            disabled={isLoading}
            className={cn(error && 'border-destructive')}
          />
          {error && (
            <p className='text-destructive mt-1.5 text-sm'>{error}</p>
          )}
        </div>
        <AlertDialogFooter>
          <AlertDialogCancel disabled={isLoading}>
            {cancelText}
          </AlertDialogCancel>
          <Button
            onClick={handleConfirm}
            disabled={disabled || isLoading}
          >
            {confirmText}
          </Button>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
