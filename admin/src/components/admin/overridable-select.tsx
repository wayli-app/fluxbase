import { Label } from '@/components/ui/label'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { OverrideBadge } from './override-badge'
import type { SettingOverride } from './overridable-switch'

interface OverridableSelectProps {
  id: string
  label: string
  description?: string
  value: string
  onValueChange: (value: string) => void
  override?: SettingOverride
  disabled?: boolean
  placeholder?: string
  children: React.ReactNode
}

export function OverridableSelect({
  id,
  label,
  description,
  value,
  onValueChange,
  override,
  disabled,
  placeholder,
  children,
}: OverridableSelectProps) {
  const isOverridden = override?.is_overridden || false
  const isDisabled = disabled || isOverridden

  return (
    <div className="space-y-2">
      <div className="flex items-center gap-2">
        <Label htmlFor={id}>{label}</Label>
        {isOverridden && <OverrideBadge envVar={override?.env_var} />}
      </div>
      {description && (
        <p className="text-sm text-muted-foreground">
          {description}
        </p>
      )}
      <Select
        value={value}
        onValueChange={onValueChange}
        disabled={isDisabled}
      >
        <SelectTrigger id={id}>
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {children}
        </SelectContent>
      </Select>
      {isOverridden && (
        <p className="text-xs text-muted-foreground">
          Controlled by <code className="bg-muted px-1 py-0.5 rounded">{override?.env_var}</code>
        </p>
      )}
    </div>
  )
}

export { SelectItem }
