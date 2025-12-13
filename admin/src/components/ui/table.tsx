import * as React from 'react'
import { cn } from '@/lib/utils'

type TableDensity = 'compact' | 'normal'

interface TableProps extends React.ComponentProps<'table'> {
  density?: TableDensity
}

function Table({ className, density = 'normal', ...props }: TableProps) {
  return (
    <div
      data-slot='table-container'
      data-density={density}
      className='relative w-full overflow-x-auto'
    >
      <table
        data-slot='table'
        className={cn(
          'w-full caption-bottom',
          density === 'compact' ? 'text-xs' : 'text-sm',
          className
        )}
        {...props}
      />
    </div>
  )
}

function TableHeader({ className, ...props }: React.ComponentProps<'thead'>) {
  return (
    <thead
      data-slot='table-header'
      className={cn('[&_tr]:border-b', className)}
      {...props}
    />
  )
}

function TableBody({ className, ...props }: React.ComponentProps<'tbody'>) {
  return (
    <tbody
      data-slot='table-body'
      className={cn('[&_tr:last-child]:border-0', className)}
      {...props}
    />
  )
}

function TableFooter({ className, ...props }: React.ComponentProps<'tfoot'>) {
  return (
    <tfoot
      data-slot='table-footer'
      className={cn(
        'bg-muted/50 border-t font-medium [&>tr]:last:border-b-0',
        className
      )}
      {...props}
    />
  )
}

function TableRow({ className, ...props }: React.ComponentProps<'tr'>) {
  return (
    <tr
      data-slot='table-row'
      className={cn(
        'hover:bg-muted/50 data-[state=selected]:bg-muted border-b transition-colors',
        className
      )}
      {...props}
    />
  )
}

function TableHead({ className, ...props }: React.ComponentProps<'th'>) {
  return (
    <th
      data-slot='table-head'
      className={cn(
        'text-foreground text-start align-middle font-medium whitespace-nowrap [&>[role=checkbox]]:translate-y-[2px]',
        // Normal density (default)
        'h-8 px-2',
        // Compact density overrides
        '[[data-density=compact]_&]:h-6 [[data-density=compact]_&]:px-1.5',
        className
      )}
      {...props}
    />
  )
}

function TableCell({ className, ...props }: React.ComponentProps<'td'>) {
  return (
    <td
      data-slot='table-cell'
      className={cn(
        'align-middle whitespace-nowrap [&>[role=checkbox]]:translate-y-[2px]',
        // Normal density (default)
        'px-2 py-1.5',
        // Compact density overrides
        '[[data-density=compact]_&]:px-1.5 [[data-density=compact]_&]:py-0.5',
        className
      )}
      {...props}
    />
  )
}

function TableCaption({
  className,
  ...props
}: React.ComponentProps<'caption'>) {
  return (
    <caption
      data-slot='table-caption'
      className={cn('text-muted-foreground mt-4 text-sm', className)}
      {...props}
    />
  )
}

export {
  Table,
  TableHeader,
  TableBody,
  TableFooter,
  TableHead,
  TableRow,
  TableCell,
  TableCaption,
}

export type { TableDensity }
