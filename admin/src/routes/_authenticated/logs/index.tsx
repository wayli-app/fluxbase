import { createFileRoute } from '@tanstack/react-router'
import { LogViewer } from '@/features/logs/components/log-viewer'

export const Route = createFileRoute('/_authenticated/logs/')({
  component: LogsPage,
})

function LogsPage() {
  return (
    <div className='flex h-full flex-1 flex-col gap-6 p-6'>
      <div>
        <h1 className='text-3xl font-bold'>Log Stream</h1>
        <p className='text-muted-foreground mt-1 text-sm'>
          Real-time application logs
        </p>
      </div>

      <div className='min-h-0 flex-1'>
        <LogViewer />
      </div>
    </div>
  )
}
