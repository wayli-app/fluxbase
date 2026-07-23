import { createFileRoute } from '@tanstack/react-router'
import { Globe } from 'lucide-react'
import { ToolIntegrationsTab } from '@/components/ai-integrations/tool-integrations-tab'

const ToolIntegrationsPage = () => {
  return (
    <div className='flex h-full flex-col'>
      <div className='flex-1 overflow-auto p-6'>
        <ToolIntegrationsTab />
      </div>
    </div>
  )
}

export const Route = createFileRoute('/_authenticated/ai-integrations/')({
  component: ToolIntegrationsPage,
})
