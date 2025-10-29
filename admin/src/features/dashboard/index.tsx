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
import { TopNav } from '@/components/layout/top-nav'
import { ProfileDropdown } from '@/components/profile-dropdown'
import { Search } from '@/components/search'
import { ThemeSwitch } from '@/components/theme-switch'
import { FluxbaseStats } from './components/fluxbase-stats'
import { Overview } from './components/overview'

export function Dashboard() {
  return (
    <>
      {/* ===== Top Heading ===== */}
      <Header>
        <TopNav links={topNav} />
        <div className='ms-auto flex items-center space-x-4'>
          <Search />
          <ThemeSwitch />
          <ConfigDrawer />
          <ProfileDropdown />
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

            {/* Additional Charts */}
            <div className='grid grid-cols-1 gap-4 lg:grid-cols-7'>
              <Card className='col-span-1 lg:col-span-4'>
                <CardHeader>
                  <CardTitle>API Usage Overview</CardTitle>
                  <CardDescription>
                    Request volume over the last 7 days
                  </CardDescription>
                </CardHeader>
                <CardContent className='ps-2'>
                  <Overview />
                </CardContent>
              </Card>
              <Card className='col-span-1 lg:col-span-3'>
                <CardHeader>
                  <CardTitle>Quick Actions</CardTitle>
                  <CardDescription>
                    Common administrative tasks
                  </CardDescription>
                </CardHeader>
                <CardContent className='space-y-2'>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Database</p>
                    <a
                      href='/#/tables'
                      className='text-primary hover:underline'
                    >
                      Browse database tables →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Users</p>
                    <a
                      href='/#/users'
                      className='text-primary hover:underline'
                    >
                      Manage user accounts →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>API</p>
                    <a
                      href='/#/api-explorer'
                      className='text-primary hover:underline'
                    >
                      Test API endpoints →
                    </a>
                  </div>
                  <div className='text-sm'>
                    <p className='text-muted-foreground mb-1'>Settings</p>
                    <a
                      href='/#/settings'
                      className='text-primary hover:underline'
                    >
                      Configure system settings →
                    </a>
                  </div>
                </CardContent>
              </Card>
            </div>
          </TabsContent>
        </Tabs>
      </Main>
    </>
  )
}

const topNav = [
  {
    title: 'Overview',
    href: '/',
    isActive: true,
    disabled: false,
  },
  {
    title: 'Tables',
    href: '/tables',
    isActive: false,
    disabled: false,
  },
  {
    title: 'Users',
    href: '/users',
    isActive: false,
    disabled: false,
  },
  {
    title: 'Settings',
    href: '/settings',
    isActive: false,
    disabled: false,
  },
]
