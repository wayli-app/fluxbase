import { useState, useEffect, useMemo } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import z from 'zod'
import { createFileRoute, getRouteApi } from '@tanstack/react-router'
import { Loader2, User, Shield, AlertCircle, Copy, Check } from 'lucide-react'
import { toast } from 'sonner'
import { dashboardAuthAPI, type DashboardUser } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { Alert, AlertDescription } from '@/components/ui/alert'
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog'
import { clearTokens } from '@/lib/auth'

const settingsSearchSchema = z.object({
  tab: z.string().optional().catch('profile'),
})

export const Route = createFileRoute('/_authenticated/settings/')({
  validateSearch: settingsSearchSchema,
  component: SettingsPage,
})

const route = getRouteApi('/_authenticated/settings/')

function SettingsPage() {
  const queryClient = useQueryClient()
  const search = route.useSearch()
  const navigate = route.useNavigate()

  // Fetch current user data
  const { data: user, isLoading } = useQuery<DashboardUser>({
    queryKey: ['dashboard-user'],
    queryFn: dashboardAuthAPI.me,
  })

  // Profile state
  const initialFullName = useMemo(
    () => user?.full_name || '',
    [user?.full_name]
  )
  const [fullName, setFullName] = useState(initialFullName)

  // Password change state
  const [currentPassword, setCurrentPassword] = useState('')
  const [newPassword, setNewPassword] = useState('')
  const [confirmPassword, setConfirmPassword] = useState('')

  // 2FA state
  const [qrCodeUrl, setQrCodeUrl] = useState('')
  const [totpSecret, setTotpSecret] = useState('')
  const [verificationCode, setVerificationCode] = useState('')
  const [backupCodes, setBackupCodes] = useState<string[]>([])
  const [copiedCode, setCopiedCode] = useState<string | null>(null)
  const [disable2FAPassword, setDisable2FAPassword] = useState('')

  // Delete account state
  const [deletePassword, setDeletePassword] = useState('')

  // Update fullName when user data changes
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
          localStorage.setItem(
            'user',
            JSON.stringify({
              ...user,
              full_name: data.full_name,
              avatar_url: data.avatar_url,
            })
          )
        } catch {
          // Silently fail if localStorage update fails
        }
      }
      toast.success('Profile updated successfully')
    },
    onError: (error: unknown) => {
      const errorMessage =
        error instanceof Error && 'response' in error
          ? (error as { response?: { data?: { error?: string } } }).response
              ?.data?.error || (error as Error).message
          : 'Failed to update profile'
      toast.error(errorMessage)
    },
  })

  // Change password mutation
  const changePasswordMutation = useMutation({
    mutationFn: dashboardAuthAPI.changePassword,
    onSuccess: () => {
      toast.success('Password changed successfully')
      setCurrentPassword('')
      setNewPassword('')
      setConfirmPassword('')
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to change password'
        : 'Failed to change password'
      toast.error(errorMessage)
    },
  })

  // Setup 2FA mutation
  const setup2FAMutation = useMutation({
    mutationFn: dashboardAuthAPI.setup2FA,
    onSuccess: (data) => {
      setQrCodeUrl(data.qr_url)
      setTotpSecret(data.secret)
      toast.success('Scan the QR code with your authenticator app')
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to setup 2FA'
        : 'Failed to setup 2FA'
      toast.error(errorMessage)
    },
  })

  // Enable 2FA mutation
  const enable2FAMutation = useMutation({
    mutationFn: dashboardAuthAPI.enable2FA,
    onSuccess: (data) => {
      setBackupCodes(data.backup_codes)
      setQrCodeUrl('')
      setTotpSecret('')
      setVerificationCode('')
      queryClient.invalidateQueries({ queryKey: ['dashboard-user'] })
      toast.success('2FA enabled successfully. Save your backup codes!')
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Invalid verification code'
        : 'Invalid verification code'
      toast.error(errorMessage)
    },
  })

  // Disable 2FA mutation
  const disable2FAMutation = useMutation({
    mutationFn: dashboardAuthAPI.disable2FA,
    onSuccess: () => {
      setDisable2FAPassword('')
      queryClient.invalidateQueries({ queryKey: ['dashboard-user'] })
      toast.success('2FA disabled successfully')
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to disable 2FA'
        : 'Failed to disable 2FA'
      toast.error(errorMessage)
    },
  })

  // Delete account mutation
  const deleteAccountMutation = useMutation({
    mutationFn: dashboardAuthAPI.deleteAccount,
    onSuccess: () => {
      toast.success('Account deleted successfully')
      clearTokens()
      window.location.href = '/login'
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } } }).response?.data?.error || 'Failed to delete account'
        : 'Failed to delete account'
      toast.error(errorMessage)
    },
  })

  const handleProfileSubmit = (e: React.FormEvent) => {
    e.preventDefault()

    if (!fullName.trim()) {
      toast.error('Display name is required')
      return
    }

    updateProfileMutation.mutate({
      full_name: fullName,
    })
  }

  const handlePasswordChange = (e: React.FormEvent) => {
    e.preventDefault()

    if (newPassword !== confirmPassword) {
      toast.error('New passwords do not match')
      return
    }

    if (newPassword.length < 8) {
      toast.error('Password must be at least 8 characters')
      return
    }

    changePasswordMutation.mutate({
      current_password: currentPassword,
      new_password: newPassword,
    })
  }

  const handleSetup2FA = () => {
    setup2FAMutation.mutate()
  }

  const handleEnable2FA = (e: React.FormEvent) => {
    e.preventDefault()
    enable2FAMutation.mutate({ code: verificationCode })
  }

  const handleDisable2FA = (e: React.FormEvent) => {
    e.preventDefault()
    disable2FAMutation.mutate({ password: disable2FAPassword })
  }

  const handleDeleteAccount = () => {
    if (!deletePassword) {
      toast.error('Please enter your password')
      return
    }

    deleteAccountMutation.mutate({ password: deletePassword })
  }

  const copyToClipboard = (code: string) => {
    navigator.clipboard.writeText(code)
    setCopiedCode(code)
    setTimeout(() => setCopiedCode(null), 2000)
    toast.success('Copied to clipboard')
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
      </div>
    )
  }

  return (
    <div className="flex flex-col gap-6 p-6">
      <div>
        <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
          <User className="h-8 w-8" />
          Account
        </h1>
        <p className="text-muted-foreground mt-2">
          Manage your profile and account security settings.
        </p>
      </div>

      <Tabs value={search.tab || 'profile'} onValueChange={(tab) => navigate({ search: { tab } })} className="space-y-4">
        <TabsList>
          <TabsTrigger value="profile" className="flex items-center gap-2">
            <User className="h-4 w-4" />
            Profile
          </TabsTrigger>
          <TabsTrigger value="security" className="flex items-center gap-2">
            <Shield className="h-4 w-4" />
            Security
          </TabsTrigger>
        </TabsList>

        <TabsContent value="profile" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Personal Information</CardTitle>
              <CardDescription>Update your personal details.</CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handleProfileSubmit} className='space-y-4'>
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
                  <p className='text-muted-foreground text-xs'>
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
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="security" className="space-y-4">
          {/* Change Password */}
          <Card>
            <CardHeader>
              <CardTitle>Change Password</CardTitle>
              <CardDescription>
                Update your password to keep your account secure.
              </CardDescription>
            </CardHeader>
            <CardContent>
              <form onSubmit={handlePasswordChange} className='space-y-4'>
                <div className='space-y-2'>
                  <Label htmlFor='current-password'>Current Password</Label>
                  <Input
                    id='current-password'
                    type='password'
                    value={currentPassword}
                    onChange={(e) => setCurrentPassword(e.target.value)}
                    required
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='new-password'>New Password</Label>
                  <Input
                    id='new-password'
                    type='password'
                    value={newPassword}
                    onChange={(e) => setNewPassword(e.target.value)}
                    required
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='confirm-password'>Confirm New Password</Label>
                  <Input
                    id='confirm-password'
                    type='password'
                    value={confirmPassword}
                    onChange={(e) => setConfirmPassword(e.target.value)}
                    required
                  />
                </div>
                <Button type='submit' disabled={changePasswordMutation.isPending}>
                  {changePasswordMutation.isPending && (
                    <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                  )}
                  Update Password
                </Button>
              </form>
            </CardContent>
          </Card>

          {/* Two-Factor Authentication */}
          <Card>
            <CardHeader>
              <CardTitle>Two-Factor Authentication</CardTitle>
              <CardDescription>
                Add an extra layer of security to your account.
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-4'>
              {user?.totp_enabled ? (
                <>
                  <Alert>
                    <AlertCircle className='h-4 w-4' />
                    <AlertDescription>
                      Two-factor authentication is currently <strong>enabled</strong> for your account.
                    </AlertDescription>
                  </Alert>
                  <form onSubmit={handleDisable2FA} className='space-y-4'>
                    <div className='space-y-2'>
                      <Label htmlFor='disable-2fa-password'>Password</Label>
                      <Input
                        id='disable-2fa-password'
                        type='password'
                        placeholder='Enter your password to disable 2FA'
                        value={disable2FAPassword}
                        onChange={(e) => setDisable2FAPassword(e.target.value)}
                        required
                      />
                    </div>
                    <Button type='submit' variant='destructive' disabled={disable2FAMutation.isPending}>
                      {disable2FAMutation.isPending && (
                        <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                      )}
                      Disable 2FA
                    </Button>
                  </form>
                </>
              ) : qrCodeUrl ? (
                <>
                  <div className='space-y-4'>
                    <div>
                      <p className='text-sm font-medium mb-2'>Scan this QR code with your authenticator app:</p>
                      <img src={qrCodeUrl} alt='QR Code' className='border rounded-lg p-4 bg-white' />
                    </div>
                    <div>
                      <p className='text-sm font-medium mb-2'>Or enter this secret manually:</p>
                      <code className='block p-2 bg-muted rounded text-sm'>{totpSecret}</code>
                    </div>
                    <form onSubmit={handleEnable2FA} className='space-y-4'>
                      <div className='space-y-2'>
                        <Label htmlFor='verification-code'>Verification Code</Label>
                        <Input
                          id='verification-code'
                          placeholder='Enter 6-digit code'
                          value={verificationCode}
                          onChange={(e) => setVerificationCode(e.target.value)}
                          required
                        />
                      </div>
                      <Button type='submit' disabled={enable2FAMutation.isPending}>
                        {enable2FAMutation.isPending && (
                          <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                        )}
                        Verify and Enable
                      </Button>
                    </form>
                  </div>
                </>
              ) : backupCodes.length > 0 ? (
                <div className='space-y-4'>
                  <Alert>
                    <AlertCircle className='h-4 w-4' />
                    <AlertDescription>
                      Save these backup codes in a safe place. You can use them to access your account if you lose your authenticator device.
                    </AlertDescription>
                  </Alert>
                  <div className='grid grid-cols-2 gap-2'>
                    {backupCodes.map((code) => (
                      <div key={code} className='flex items-center justify-between p-2 bg-muted rounded'>
                        <code className='text-sm'>{code}</code>
                        <Button
                          size='sm'
                          variant='ghost'
                          onClick={() => copyToClipboard(code)}
                        >
                          {copiedCode === code ? (
                            <Check className='h-4 w-4' />
                          ) : (
                            <Copy className='h-4 w-4' />
                          )}
                        </Button>
                      </div>
                    ))}
                  </div>
                  <Button onClick={() => setBackupCodes([])}>I've Saved My Codes</Button>
                </div>
              ) : (
                <>
                  <Alert>
                    <AlertCircle className='h-4 w-4' />
                    <AlertDescription>
                      Two-factor authentication is not currently enabled for your account.
                    </AlertDescription>
                  </Alert>
                  <Button variant='outline' onClick={handleSetup2FA} disabled={setup2FAMutation.isPending}>
                    {setup2FAMutation.isPending && (
                      <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                    )}
                    Enable 2FA
                  </Button>
                </>
              )}
            </CardContent>
          </Card>

          {/* Danger Zone */}
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
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant='destructive'>Delete Account</Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Are you absolutely sure?</AlertDialogTitle>
                      <AlertDialogDescription className='space-y-4'>
                        <p>
                          This action cannot be undone. This will permanently delete your
                          account and remove your data from our servers.
                        </p>
                        <div className='space-y-2'>
                          <Label htmlFor='delete-password'>Enter your password to confirm:</Label>
                          <Input
                            id='delete-password'
                            type='password'
                            placeholder='Password'
                            value={deletePassword}
                            onChange={(e) => setDeletePassword(e.target.value)}
                          />
                        </div>
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel onClick={() => setDeletePassword('')}>Cancel</AlertDialogCancel>
                      <AlertDialogAction
                        onClick={handleDeleteAccount}
                        className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
                        disabled={deleteAccountMutation.isPending}
                      >
                        {deleteAccountMutation.isPending && (
                          <Loader2 className='mr-2 h-4 w-4 animate-spin' />
                        )}
                        Delete Account
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}
