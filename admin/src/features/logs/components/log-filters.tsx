import { Search, Filter, X, History, FileCode } from 'lucide-react'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Checkbox } from '@/components/ui/checkbox'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import {
  LOG_CATEGORIES,
  LOG_LEVELS,
  LOG_CATEGORY_CONFIG,
  LOG_LEVEL_CONFIG,
  TIME_RANGE_PRESETS,
} from '../constants'
import type { LogFilters, LogCategory, LogLevel } from '../types'

interface LogFiltersToolbarProps {
  filters: LogFilters
  onFiltersChange: (filters: LogFilters) => void
  /** Currently selected time preset in minutes (if any) */
  activeTimePreset?: number
  /** Callback when a time preset is selected */
  onTimePresetChange?: (minutes: number | null) => void
}

export function LogFiltersToolbar({
  filters,
  onFiltersChange,
  activeTimePreset,
  onTimePresetChange,
}: LogFiltersToolbarProps) {
  const hasActiveFilters =
    filters.category !== 'all' ||
    filters.levels.length > 0 ||
    filters.search ||
    filters.component ||
    filters.hideStaticAssets

  const clearFilters = () => {
    onFiltersChange({
      category: 'all',
      levels: [],
      component: '',
      search: '',
      timeRange: { start: null, end: null },
      hideStaticAssets: false,
    })
    onTimePresetChange?.(null)
  }

  const toggleLevel = (level: LogLevel) => {
    const newLevels = filters.levels.includes(level)
      ? filters.levels.filter((l) => l !== level)
      : [...filters.levels, level]
    onFiltersChange({ ...filters, levels: newLevels })
  }

  const handleTimePreset = (minutes: number) => {
    const end = new Date()
    const start = new Date(end.getTime() - minutes * 60 * 1000)
    onFiltersChange({
      ...filters,
      timeRange: { start, end },
    })
    onTimePresetChange?.(minutes)
  }

  return (
    <div className='flex flex-wrap items-center gap-3'>
      {/* Search Input */}
      <div className='relative max-w-sm min-w-[200px] flex-1'>
        <Search className='text-muted-foreground absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2' />
        <Input
          placeholder='Search logs...'
          value={filters.search}
          onChange={(e) =>
            onFiltersChange({ ...filters, search: e.target.value })
          }
          className='h-8 pl-9'
        />
      </div>

      {/* Category Filter */}
      <Select
        value={filters.category}
        onValueChange={(value) =>
          onFiltersChange({
            ...filters,
            category: value as LogCategory | 'all',
          })
        }
      >
        <SelectTrigger className='h-8 w-[140px]'>
          <SelectValue placeholder='Category' />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value='all'>All Categories</SelectItem>
          {LOG_CATEGORIES.map((cat) => {
            const config = LOG_CATEGORY_CONFIG[cat]
            return (
              <SelectItem key={cat} value={cat}>
                <div className='flex items-center gap-2'>
                  <config.icon className='h-3.5 w-3.5' />
                  {config.label}
                </div>
              </SelectItem>
            )
          })}
        </SelectContent>
      </Select>

      {/* Level Filter */}
      <Popover>
        <PopoverTrigger asChild>
          <Button variant='outline' size='sm' className='h-8'>
            <Filter className='mr-2 h-3 w-3' />
            Levels
            {filters.levels.length > 0 && (
              <Badge variant='secondary' className='ml-2 h-4 px-1 text-xs'>
                {filters.levels.length}
              </Badge>
            )}
          </Button>
        </PopoverTrigger>
        <PopoverContent className='w-48 p-3' align='start'>
          <div className='space-y-2'>
            {LOG_LEVELS.map((level) => {
              const config = LOG_LEVEL_CONFIG[level]
              return (
                <div key={level} className='flex items-center gap-2'>
                  <Checkbox
                    id={`level-${level}`}
                    checked={filters.levels.includes(level)}
                    onCheckedChange={() => toggleLevel(level)}
                  />
                  <Label
                    htmlFor={`level-${level}`}
                    className='flex cursor-pointer items-center gap-2 text-sm'
                  >
                    <span className={`h-2 w-2 rounded-full ${config.color}`} />
                    {config.label}
                  </Label>
                </div>
              )
            })}
          </div>
        </PopoverContent>
      </Popover>

      {/* Time Jump Presets */}
      <div className='flex items-center gap-1'>
        <History className='text-muted-foreground mr-1 h-3.5 w-3.5' />
        {TIME_RANGE_PRESETS.slice(0, 4).map((preset) => (
          <Button
            key={preset.minutes}
            variant={
              activeTimePreset === preset.minutes ? 'secondary' : 'ghost'
            }
            size='sm'
            className='h-7 px-2 text-xs'
            onClick={() => handleTimePreset(preset.minutes)}
          >
            {preset.label.replace('Last ', '')}
          </Button>
        ))}
      </div>

      {/* Hide Static Assets Toggle (only visible for HTTP category) */}
      {filters.category === 'http' && (
        <Button
          variant={filters.hideStaticAssets ? 'secondary' : 'outline'}
          size='sm'
          className='h-8 gap-1.5'
          onClick={() =>
            onFiltersChange({
              ...filters,
              hideStaticAssets: !filters.hideStaticAssets,
            })
          }
        >
          <FileCode className='h-3 w-3' />
          Hide Assets
        </Button>
      )}

      {/* Clear Filters */}
      {hasActiveFilters && (
        <Button
          variant='ghost'
          size='sm'
          onClick={clearFilters}
          className='text-muted-foreground h-8'
        >
          <X className='mr-1 h-3 w-3' />
          Clear
        </Button>
      )}
    </div>
  )
}
