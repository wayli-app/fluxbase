import { useState, useEffect } from 'react'
import { z } from 'zod'
import { useForm } from 'react-hook-form'
import { zodResolver } from '@hookform/resolvers/zod'
import { useNavigate, getRouteApi } from '@tanstack/react-router'
import { CheckCircle, Loader2, AlertCircle } from 'lucide-react'
import { toast } from 'sonner'
import { cn } from '@/lib/utils'
import { authApi } from '@/lib/api'
import { Button } from '@/components/ui/button'
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Alert, AlertDescription } from '@/components/ui/alert'

const route = getRouteApi('/(auth)/reset-password')

const formSchema = z.object({
  password: z.string().min(8, 'Password must be at least 8 characters'),
  confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
  message: "Passwords don't match",
  path: ['confirmPassword'],
})

export function ResetPasswordForm({
  className,
  ...props
}: React.HTMLAttributes<HTMLFormElement>) {
  const navigate = useNavigate()
  const search = route.useSearch()
  const [isLoading, setIsLoading] = useState(false)
  const [isValidating, setIsValidating] = useState(true)
  const [tokenValid, setTokenValid] = useState(false)
  const [tokenError, setTokenError] = useState<string | null>(null)
  const [resetSuccess, setResetSuccess] = useState(false)

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: { password: '', confirmPassword: '' },
  })

  // Validate token on mount
  useEffect(() => {
    const validateToken = async () => {
      if (!search.token) {
        setTokenError('No reset token provided')
        setTokenValid(false)
        setIsValidating(false)
        return
      }

      try {
        const result = await authApi.verifyResetToken(search.token)
        if (result.valid) {
          setTokenValid(true)
          setTokenError(null)
        } else {
          setTokenValid(false)
          setTokenError(result.message || 'Invalid or expired reset token')
        }
      } catch {
        setTokenValid(false)
        setTokenError('Failed to validate reset token')
      } finally {
        setIsValidating(false)
      }
    }

    validateToken()
  }, [search.token])

  async function onSubmit(data: z.infer<typeof formSchema>) {
    if (!search.token) {
      toast.error('No reset token provided')
      return
    }

    setIsLoading(true)

    try {
      await authApi.resetPassword(search.token, data.password)
      setResetSuccess(true)
      toast.success('Password reset successfully!')
      form.reset()

      // Redirect to sign-in after 2 seconds
      setTimeout(() => {
        navigate({ to: '/sign-in' })
      }, 2000)
    } catch (error) {
      const errorMessage = error instanceof Error ? error.message : 'Failed to reset password'
      toast.error(errorMessage)
    } finally {
      setIsLoading(false)
    }
  }

  if (isValidating) {
    return (
      <div className='flex items-center justify-center py-8'>
        <Loader2 className='h-8 w-8 animate-spin text-muted-foreground' />
      </div>
    )
  }

  if (!tokenValid) {
    return (
      <Alert variant='destructive'>
        <AlertCircle className='h-4 w-4' />
        <AlertDescription>
          {tokenError || 'Invalid or expired reset token. Please request a new password reset.'}
        </AlertDescription>
      </Alert>
    )
  }

  if (resetSuccess) {
    return (
      <Alert className='border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-900/20'>
        <CheckCircle className='h-4 w-4 text-green-600 dark:text-green-400' />
        <AlertDescription className='text-green-800 dark:text-green-200'>
          Password reset successfully! Redirecting to sign in...
        </AlertDescription>
      </Alert>
    )
  }

  return (
    <Form {...form}>
      <form
        onSubmit={form.handleSubmit(onSubmit)}
        className={cn('grid gap-4', className)}
        {...props}
      >
        <FormField
          control={form.control}
          name='password'
          render={({ field }) => (
            <FormItem>
              <FormLabel>New Password</FormLabel>
              <FormControl>
                <Input
                  type='password'
                  placeholder='Enter new password'
                  {...field}
                />
              </FormControl>
              <FormDescription>
                Must be at least 8 characters long
              </FormDescription>
              <FormMessage />
            </FormItem>
          )}
        />
        <FormField
          control={form.control}
          name='confirmPassword'
          render={({ field }) => (
            <FormItem>
              <FormLabel>Confirm Password</FormLabel>
              <FormControl>
                <Input
                  type='password'
                  placeholder='Confirm new password'
                  {...field}
                />
              </FormControl>
              <FormMessage />
            </FormItem>
          )}
        />
        <Button className='mt-2' disabled={isLoading}>
          {isLoading ? (
            <>
              Resetting password...
              <Loader2 className='animate-spin ml-2 h-4 w-4' />
            </>
          ) : (
            'Reset Password'
          )}
        </Button>
      </form>
    </Form>
  )
}
