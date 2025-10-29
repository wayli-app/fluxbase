import { createFileRoute } from '@tanstack/react-router'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Label } from '@/components/ui/label'
import { Input } from '@/components/ui/input'
import { Button } from '@/components/ui/button'
import { AlertCircle } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'

export const Route = createFileRoute('/_authenticated/settings/account')({
  component: SettingsAccountPage,
})

function SettingsAccountPage() {
  return (
    <div className='space-y-4'>
      <div>
        <h1 className='text-3xl font-bold tracking-tight'>Account</h1>
        <p className='text-muted-foreground'>
          Manage your account security and preferences.
        </p>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Change Password</CardTitle>
          <CardDescription>
            Update your password to keep your account secure.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='space-y-2'>
            <Label htmlFor='current-password'>Current Password</Label>
            <Input id='current-password' type='password' />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='new-password'>New Password</Label>
            <Input id='new-password' type='password' />
          </div>
          <div className='space-y-2'>
            <Label htmlFor='confirm-password'>Confirm New Password</Label>
            <Input id='confirm-password' type='password' />
          </div>
          <Button>Update Password</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Two-Factor Authentication</CardTitle>
          <CardDescription>
            Add an extra layer of security to your account.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <Alert>
            <AlertCircle className='h-4 w-4' />
            <AlertDescription>
              Two-factor authentication is not currently enabled for your account.
            </AlertDescription>
          </Alert>
          <Button variant='outline'>Enable 2FA</Button>
        </CardContent>
      </Card>

      <Card className='border-destructive'>
        <CardHeader>
          <CardTitle className='text-destructive'>Danger Zone</CardTitle>
          <CardDescription>
            Irreversible and destructive actions.
          </CardDescription>
        </CardHeader>
        <CardContent className='space-y-4'>
          <div className='space-y-2'>
            <p className='text-sm'>
              Once you delete your account, there is no going back. Please be certain.
            </p>
            <Button variant='destructive'>Delete Account</Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
