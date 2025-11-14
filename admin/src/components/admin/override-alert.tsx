import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

interface OverrideAlertProps {
  envVar: string
  className?: string
}

export function OverrideAlert({ envVar, className }: OverrideAlertProps) {
  return (
    <Alert className={className}>
      <AlertCircle className="h-4 w-4" />
      <AlertDescription>
        This setting is controlled by the environment variable <code className="text-sm bg-muted px-1 py-0.5 rounded">{envVar}</code> and cannot be changed here.
      </AlertDescription>
    </Alert>
  )
}
