import { Lock } from 'lucide-react'
import { Badge } from '@/components/ui/badge'

interface OverrideBadgeProps {
  envVar?: string
  className?: string
}

export function OverrideBadge({ envVar, className }: OverrideBadgeProps) {
  return (
    <Badge variant="secondary" className={className}>
      <Lock className="h-3 w-3 mr-1" />
      Environment Variable
      {envVar && (
        <span className="ml-1 text-xs opacity-70">({envVar})</span>
      )}
    </Badge>
  )
}
