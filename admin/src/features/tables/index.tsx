import { useEffect } from 'react'
import { getRouteApi } from '@tanstack/react-router'
import { Panel, PanelGroup, PanelResizeHandle } from 'react-resizable-panels'
import { ConfigDrawer } from '@/components/config-drawer'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { TableSelector } from './components/table-selector'
import { TableViewer } from './components/table-viewer'
import { ImpersonationBanner } from '@/components/impersonation-banner'
import { ImpersonationSelector } from '@/features/impersonation/components/impersonation-selector'

const route = getRouteApi('/_authenticated/tables/')

export function Tables() {
  const navigate = route.useNavigate()
  const search = route.useSearch()
  const selectedTable = search.table
  const selectedSchema = search.schema || 'public'

  // Auto-select first table if none selected
  useEffect(() => {
    if (!selectedTable) {
      // This will be handled by the TableSelector component
    }
  }, [selectedTable])

  const handleTableSelect = (table: string) => {
    navigate({
      search: (prev) => ({ ...prev, table, page: 1 }),
    })
  }

  const handleSchemaChange = (schema: string) => {
    navigate({
      search: (prev) => ({ ...prev, schema, table: undefined, page: 1 }),
    })
  }

  return (
    <>
      <Header fixed>
        <Search />
        <div className='ms-auto flex items-center space-x-4'>
          <ImpersonationSelector />
          <ThemeSwitch />
          <ConfigDrawer />
        </div>
      </Header>

      <ImpersonationBanner />

      <Main className='h-[calc(100vh-4rem)] p-0'>
        <PanelGroup direction='horizontal'>
          <Panel defaultSize={20} minSize={15} maxSize={40}>
            <TableSelector
              selectedTable={selectedTable}
              selectedSchema={selectedSchema}
              onTableSelect={handleTableSelect}
              onSchemaChange={handleSchemaChange}
            />
          </Panel>
          <PanelResizeHandle className='w-1 bg-border transition-colors hover:bg-primary' />
          <Panel>
            <main className='h-full overflow-auto'>
              {selectedTable ? (
                <TableViewer tableName={selectedTable} schema={selectedSchema} />
              ) : (
                <div className='flex h-full items-center justify-center'>
                  <p className='text-muted-foreground'>
                    Select a table from the sidebar to view its data
                  </p>
                </div>
              )}
            </main>
          </Panel>
        </PanelGroup>
      </Main>
    </>
  )
}
