import { Label } from '@/components/ui/label'
import { Switch } from '@/components/ui/switch'
import { OverrideBadge } from './override-badge'

export interface SettingOverride {
  is_overridden: boolean
  env_var: string
}

interface OverridableSwitchProps {
  id: string
  label: string
  description?: string
  checked: boolean
  onCheckedChange: (checked: boolean) => void
  override?: SettingOverride
  disabled?: boolean
}

export function OverridableSwitch({
  id,
  label,
  description,
  checked,
  onCheckedChange,
  override,
  disabled,
}: OverridableSwitchProps) {
  const isOverridden = override?.is_overridden || false
  const isDisabled = disabled || isOverridden

  return (
    <div className="flex items-center justify-between">
      <div className="space-y-0.5 flex-1">
        <div className="flex items-center gap-2">
          <Label htmlFor={id}>{label}</Label>
          {isOverridden && <OverrideBadge envVar={override?.env_var} />}
        </div>
        {description && (
          <p className="text-sm text-muted-foreground">
            {description}
          </p>
        )}
        {isOverridden && (
          <p className="text-xs text-muted-foreground mt-1">
            Controlled by <code className="bg-muted px-1 py-0.5 rounded">{override?.env_var}</code>
          </p>
        )}
      </div>
      <Switch
        id={id}
        checked={checked}
        onCheckedChange={onCheckedChange}
        disabled={isDisabled}
      />
    </div>
  )
}
