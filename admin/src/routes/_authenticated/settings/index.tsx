import { createFileRoute } from '@tanstack/react-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { toast } from 'sonner'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { dashboardAuthAPI, type DashboardUser } from '@/lib/api'
import { Loader2 } from 'lucide-react'
import { useState, useEffect, useMemo } from 'react'

export const Route = createFileRoute('/_authenticated/settings/')({
  component: SettingsProfilePage,
})

function SettingsProfilePage() {
  const queryClient = useQueryClient()

  // Fetch current user data
  const { data: user, isLoading } = useQuery<DashboardUser>({
    queryKey: ['dashboard-user'],
    queryFn: dashboardAuthAPI.me,
  })

  // Initialize fullName from user data
  const initialFullName = useMemo(() => user?.full_name || '', [user?.full_name])
  const [fullName, setFullName] = useState(initialFullName)

  // Update fullName when initialFullName changes
  useEffect(() => {
    setFullName(initialFullName)
  }, [initialFullName])

  // Update profile mutation
  const updateProfileMutation = useMutation({
    mutationFn: dashboardAuthAPI.updateProfile,
    onSuccess: (data) => {
      queryClient.setQueryData(['dashboard-user'], data)
      // Also update the user in localStorage so the sidebar reflects the change
      const currentUser = localStorage.getItem('user')
      if (currentUser) {
        try {
          const user = JSON.parse(currentUser)
          localStorage.setItem('user', JSON.stringify({ ...user, full_name: data.full_name, avatar_url: data.avatar_url }))
        } catch (e) {
          console.error('Failed to update user in localStorage', e)
        }
      }
      toast.success('Profile updated successfully')
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || (error as Error).message
        : 'Failed to update profile'
      toast.error(errorMessage)
    },
  })

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!fullName.trim()) {
      toast.error('Display name is required')
      return
    }

    updateProfileMutation.mutate({
      full_name: fullName,
    })
  }

  return (
    <div className='space-y-4'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight'>Profile</h1>
        <p className='text-muted-foreground'>
          Manage your dashboard profile settings.
        </p>
      </div>
      <Card>
        <CardHeader>
          <CardTitle>Personal Information</CardTitle>
          <CardDescription>
            Update your personal details.
          </CardDescription>
        </CardHeader>
        <CardContent>
          {isLoading ? (
            <div className='flex items-center justify-center py-8'>
              <Loader2 className='h-6 w-6 animate-spin text-muted-foreground' />
            </div>
          ) : (
            <form onSubmit={handleSubmit} className='space-y-4'>
              <div className='space-y-2'>
                <Label htmlFor='name'>Display Name</Label>
                <Input
                  id='name'
                  placeholder='Enter your name'
                  value={fullName}
                  onChange={(e) => setFullName(e.target.value)}
                  required
                />
              </div>
              <div className='space-y-2'>
                <Label htmlFor='email'>Email</Label>
                <Input
                  id='email'
                  type='email'
                  value={user?.email || ''}
                  disabled
                />
                <p className='text-xs text-muted-foreground'>
                  Your email address cannot be changed.
                </p>
              </div>
              <Button type='submit' disabled={updateProfileMutation.isPending}>
                {updateProfileMutation.isPending && (
                  <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                )}
                Save Changes
              </Button>
            </form>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
