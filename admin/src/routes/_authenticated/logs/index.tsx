import { createFileRoute } from '@tanstack/react-router'
import { ScrollText } from 'lucide-react'
import { LogViewer } from '@/features/logs/components/log-viewer'
import { PageHeader } from '@/components/layout/page-header'

export const Route = createFileRoute('/_authenticated/logs/')({
  component: LogsPage,
})

function LogsPage() {
  return (
    <div className='flex h-full flex-col'>
      <PageHeader
        icon={<ScrollText />}
        title="Log Stream"
        description="Real-time application logs"
      />

      <div className='min-h-0 flex-1 p-6'>
        <LogViewer />
      </div>
    </div>
  )
}
