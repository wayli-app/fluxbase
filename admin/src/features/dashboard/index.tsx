import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { ConfigDrawer } from '@/components/config-drawer'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { FluxbaseStats } from './components/fluxbase-stats'

export function Dashboard() {
  return (
    <>
      {/* ===== Top Heading ===== */}
      <Header>
        <div className='ms-auto flex items-center space-x-4'>
          <Search />
          <ThemeSwitch />
          <ConfigDrawer />
        </div>
      </Header>

      {/* ===== Main ===== */}
      <Main>
        <div className='mb-2 flex items-center justify-between space-y-2'>
          <div>
            <h1 className='text-2xl font-bold tracking-tight'>Fluxbase Dashboard</h1>
            <p className='text-muted-foreground text-sm'>
              Monitor your Backend as a Service
            </p>
          </div>
        </div>
        <Tabs
          orientation='vertical'
          defaultValue='overview'
          className='space-y-4'
        >
          <div className='w-full overflow-x-auto pb-2'>
            <TabsList>
              <TabsTrigger value='overview'>Overview</TabsTrigger>
            </TabsList>
          </div>
          <TabsContent value='overview' className='space-y-4'>
            {/* Fluxbase System Stats */}
            <FluxbaseStats />

            {/* Quick Actions */}
            <Card>
              <CardHeader>
                <CardTitle>Quick Actions</CardTitle>
                <CardDescription>
                  Common administrative tasks
                </CardDescription>
              </CardHeader>
              <CardContent>
                <div className='grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4'>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Database</p>
                    <a
                      href='/admin/tables'
                      className='text-primary hover:underline'
                    >
                      Browse database tables →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Users</p>
                    <a
                      href='/admin/users'
                      className='text-primary hover:underline'
                    >
                      Manage user accounts →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Functions</p>
                    <a
                      href='/admin/functions'
                      className='text-primary hover:underline'
                    >
                      Test RPC functions →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Settings</p>
                    <a
                      href='/admin/settings'
                      className='text-primary hover:underline'
                    >
                      Configure system settings →
                    </a>
                  </div>
                </div>
              </CardContent>
            </Card>
          </TabsContent>
        </Tabs>
      </Main>
    </>
  )
}
