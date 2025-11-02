import { useState } from 'react'
import { createFileRoute } from '@tanstack/react-router'
import { useQuery } from '@tanstack/react-query'
import { z } from 'zod'
import { Users, UserPlus, UserCheck, Clock } from 'lucide-react'
import { userManagementApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { UsersTable } from '@/features/users/components/users-table'
import { UsersInviteDialog } from '@/features/users/components/users-invite-dialog'
import { UsersProvider } from '@/features/users/components/users-provider'
import { UsersDialogs } from '@/features/users/components/users-dialogs'

const usersSearchSchema = z.object({
  page: z.number().optional(),
  pageSize: z.number().optional(),
  email: z.string().optional(),
  provider: z.array(z.string()).optional(),
  role: z.array(z.string()).optional(),
  tab: z.enum(['app', 'dashboard']).optional(),
})

export const Route = createFileRoute('/_authenticated/users/')({
  component: UsersPage,
  validateSearch: usersSearchSchema,
})

function UsersPage() {
  const navigate = Route.useNavigate()
  const search = Route.useSearch()
  const [inviteDialogOpen, setInviteDialogOpen] = useState(false)
  const activeTab = search.tab || 'app'

  // Fetch users from API based on active tab
  const { data: usersResponse, isLoading } = useQuery({
    queryKey: ['users', activeTab],
    queryFn: () => userManagementApi.listUsers(activeTab),
  })

  // Extract users array from response (backend returns {users: [], total: number})
  const rawUsers = usersResponse?.users || []

  // Convert API response to match frontend schema (date strings to Date objects)
  const users = rawUsers.map((user) => ({
    ...user,
    last_sign_in: user.last_sign_in ? new Date(user.last_sign_in) : null,
    created_at: new Date(user.created_at),
    updated_at: new Date(user.updated_at),
  }))

  // Calculate stats
  const totalUsers = users.length
  const verifiedUsers = users.filter((u) => u.email_verified).length
  const activeToday = users.filter((u) => {
    if (!u.last_sign_in) return false
    const lastSignIn = new Date(u.last_sign_in)
    const today = new Date()
    return (
      lastSignIn.getDate() === today.getDate() &&
      lastSignIn.getMonth() === today.getMonth() &&
      lastSignIn.getFullYear() === today.getFullYear()
    )
  }).length
  const pendingInvites = users.filter((u) => u.provider === 'invite_pending').length

  if (isLoading) {
    return (
      <div className='flex h-full items-center justify-center'>
        <div className='text-muted-foreground'>Loading users...</div>
      </div>
    )
  }

  return (
    <UsersProvider userType={activeTab}>
      <div className='flex h-full flex-col gap-6 p-6'>
        {/* Header */}
        <div className='flex items-center justify-between'>
          <div>
            <h1 className='text-3xl font-bold'>Users</h1>
            <p className='text-muted-foreground'>
              {activeTab === 'app'
                ? 'Manage application users who access your app through the REST API'
                : 'Manage Fluxbase dashboard administrators and operators'}
            </p>
          </div>
          <Button onClick={() => setInviteDialogOpen(true)}>
            <UserPlus className='mr-2 h-4 w-4' />
            Invite User
          </Button>
        </div>

        {/* Tabs for User Types */}
        <Tabs
          value={activeTab}
          onValueChange={(value) => {
            navigate({
              search: { ...search, tab: value as 'app' | 'dashboard' },
            })
          }}
        >
          <TabsList>
            <TabsTrigger value='app'>Application Users</TabsTrigger>
            <TabsTrigger value='dashboard'>Fluxbase Users</TabsTrigger>
          </TabsList>

          <TabsContent value={activeTab} className='space-y-4 mt-6'>
            {/* Stats Cards */}
            <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-4'>
              <Card>
                <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                  <CardTitle className='text-sm font-medium'>Total Users</CardTitle>
                  <Users className='h-4 w-4 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-2xl font-bold'>{totalUsers}</div>
                  <p className='text-xs text-muted-foreground'>
                    {verifiedUsers} verified
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                  <CardTitle className='text-sm font-medium'>Active Today</CardTitle>
                  <Clock className='h-4 w-4 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-2xl font-bold'>{activeToday}</div>
                  <p className='text-xs text-muted-foreground'>
                    Users signed in today
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                  <CardTitle className='text-sm font-medium'>
                    Pending Invites
                  </CardTitle>
                  <UserPlus className='h-4 w-4 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-2xl font-bold'>{pendingInvites}</div>
                  <p className='text-xs text-muted-foreground'>
                    Awaiting first sign in
                  </p>
                </CardContent>
              </Card>

              <Card>
                <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
                  <CardTitle className='text-sm font-medium'>
                    Verified Users
                  </CardTitle>
                  <UserCheck className='h-4 w-4 text-muted-foreground' />
                </CardHeader>
                <CardContent>
                  <div className='text-2xl font-bold'>{verifiedUsers}</div>
                  <p className='text-xs text-muted-foreground'>
                    {Math.round((verifiedUsers / totalUsers) * 100) || 0}% of total
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Users Table */}
            <UsersTable data={users} search={search} navigate={navigate} />
          </TabsContent>
        </Tabs>

        {/* Invite Dialog */}
        <UsersInviteDialog
          open={inviteDialogOpen}
          onOpenChange={setInviteDialogOpen}
        />
      </div>

      {/* Dialogs for edit/delete actions */}
      <UsersDialogs />
    </UsersProvider>
  )
}
