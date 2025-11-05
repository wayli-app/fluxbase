import { useState } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { MailPlus, Send, Copy, Check, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { userManagementApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { PasswordInput } from '@/components/password-input'
import { SelectDropdown } from '@/components/select-dropdown'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { roles } from '../data/data'
import { useUsers } from './users-provider'

const formSchema = z.object({
  email: z.string().email('Please enter a valid email address.'),
  role: z.string().min(1, 'Role is required.'),
  password: z.string().min(8, 'Password must be at least 8 characters.').optional().or(z.literal('')),
})

type UserInviteForm = z.infer<typeof formSchema>

type UserInviteDialogProps = {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function UsersInviteDialog({
  open,
  onOpenChange,
}: UserInviteDialogProps) {
  const queryClient = useQueryClient()
  const { userType } = useUsers()
  const [inviteResult, setInviteResult] = useState<{
    temporaryPassword?: string
    message: string
    emailSent: boolean
  } | null>(null)
  const [copied, setCopied] = useState(false)

  const form = useForm<UserInviteForm>({
    resolver: zodResolver(formSchema),
    defaultValues: { email: '', role: 'user', password: '' },
  })

  const inviteMutation = useMutation({
    mutationFn: (data: { email: string; role: string; password?: string }) => userManagementApi.inviteUser(data, userType),
    onSuccess: (data) => {
      // Invalidate users query to refresh the list
      queryClient.invalidateQueries({ queryKey: ['users'] })

      // Show result
      setInviteResult({
        temporaryPassword: data.temporary_password,
        message: data.message,
        emailSent: data.email_sent,
      })

      // If email was sent, show toast and close dialog
      if (data.email_sent) {
        toast.success('User invited', { description: data.message })
        handleClose()
      }
    },
    onError: (error: unknown) => {
      const errorMessage = error instanceof Error && 'response' in error
        ? (error as { response?: { data?: { error?: string } }; message?: string }).response?.data?.error || (error as Error).message
        : 'Unknown error'
      toast.error('Failed to invite user', {
        description: errorMessage,
      })
    },
  })

  const onSubmit = (values: UserInviteForm) => {
    // Only send password if it's not empty
    const payload = {
      email: values.email,
      role: values.role,
      ...(values.password && { password: values.password }),
    }
    inviteMutation.mutate(payload)
  }

  const handleClose = () => {
    form.reset()
    setInviteResult(null)
    setCopied(false)
    onOpenChange(false)
  }

  const copyToClipboard = async () => {
    if (inviteResult?.temporaryPassword) {
      await navigator.clipboard.writeText(inviteResult.temporaryPassword)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    }
  }

  // Show temporary password result if SMTP is disabled
  if (inviteResult?.temporaryPassword) {
    return (
      <Dialog open={open} onOpenChange={handleClose}>
        <DialogContent className='sm:max-w-md'>
          <DialogHeader className='text-start'>
            <DialogTitle className='flex items-center gap-2'>
              <MailPlus /> User Invited
            </DialogTitle>
            <DialogDescription>{inviteResult.message}</DialogDescription>
          </DialogHeader>
          <Alert>
            <AlertCircle className='h-4 w-4' />
            <AlertDescription>
              SMTP is not configured. Share this temporary password with the
              user. They can use it to sign in and should change it immediately.
            </AlertDescription>
          </Alert>
          <div className='space-y-2'>
            <FormLabel>Temporary Password</FormLabel>
            <div className='flex gap-2'>
              <Input
                readOnly
                value={inviteResult.temporaryPassword}
                className='font-mono'
              />
              <Button
                type='button'
                variant='outline'
                size='icon'
                onClick={copyToClipboard}
              >
                {copied ? (
                  <Check className='h-4 w-4' />
                ) : (
                  <Copy className='h-4 w-4' />
                )}
              </Button>
            </div>
          </div>
          <DialogFooter>
            <Button onClick={handleClose}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleClose}>
      <DialogContent className='sm:max-w-md'>
        <DialogHeader className='text-start'>
          <DialogTitle className='flex items-center gap-2'>
            <MailPlus /> Invite User
          </DialogTitle>
          <DialogDescription>
            Invite new user to join your team. Assign a role to define their
            access level.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form
            id='user-invite-form'
            onSubmit={form.handleSubmit(onSubmit)}
            className='space-y-4'
          >
            <FormField
              control={form.control}
              name='email'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Email</FormLabel>
                  <FormControl>
                    <Input
                      type='email'
                      placeholder='eg: john.doe@gmail.com'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='role'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Role</FormLabel>
                  <SelectDropdown
                    defaultValue={field.value}
                    onValueChange={field.onChange}
                    placeholder='Select a role'
                    items={roles.map(({ label, value }) => ({
                      label,
                      value,
                    }))}
                  />
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name='password'
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Password (optional)</FormLabel>
                  <FormControl>
                    <PasswordInput
                      placeholder='Leave empty to auto-generate'
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                  <p className='text-xs text-muted-foreground'>
                    If left empty, a random password will be generated
                  </p>
                </FormItem>
              )}
            />
          </form>
        </Form>
        <DialogFooter className='gap-y-2'>
          <DialogClose asChild>
            <Button variant='outline' disabled={inviteMutation.isPending}>
              Cancel
            </Button>
          </DialogClose>
          <Button
            type='submit'
            form='user-invite-form'
            disabled={inviteMutation.isPending}
          >
            {inviteMutation.isPending ? 'Inviting...' : 'Invite'} <Send />
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
