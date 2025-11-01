import { useEffect } from 'react'
import { getRouteApi } from '@tanstack/react-router'
import { ConfigDrawer } from '@/components/config-drawer'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { ProfileDropdown } from '@/components/profile-dropdown'
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

  return (
    <>
      <Header fixed>
        <Search />
        <div className='ms-auto flex items-center space-x-4'>
          <ImpersonationSelector />
          <ThemeSwitch />
          <ConfigDrawer />
          <ProfileDropdown />
        </div>
      </Header>

      <ImpersonationBanner />

      <Main className='flex h-[calc(100vh-4rem)] gap-0 p-0'>
        <aside className='w-64 flex-shrink-0'>
          <TableSelector
            selectedTable={selectedTable}
            onTableSelect={handleTableSelect}
          />
        </aside>
        <main className='flex-1 overflow-auto'>
          {selectedTable ? (
            <TableViewer tableName={selectedTable} />
          ) : (
            <div className='flex h-full items-center justify-center'>
              <p className='text-muted-foreground'>
                Select a table from the sidebar to view its data
              </p>
            </div>
          )}
        </main>
      </Main>
    </>
  )
}
