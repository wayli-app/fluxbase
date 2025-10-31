import { type QueryClient } from '@tanstack/react-query'
import { createRootRouteWithContext, Outlet } from '@tanstack/react-router'
import { ReactQueryDevtools } from '@tanstack/react-query-devtools'
import { TanStackRouterDevtools } from '@tanstack/react-router-devtools'
import { Toaster } from '@/components/ui/sonner'
import { NavigationProgress } from '@/components/navigation-progress'
import { GeneralError } from '@/features/errors/general-error'
import { NotFoundError } from '@/features/errors/not-found-error'

export const Route = createRootRouteWithContext<{
  queryClient: QueryClient
}>()({
  beforeLoad: async ({ location }) => {
    // Skip setup check for auth pages
    if (location.pathname === '/setup' || location.pathname === '/login' || location.pathname === '/sign-up' || location.pathname === '/sign-in') {
      return
    }

    // TEMPORARILY DISABLED: Backend server hanging issue
    // Check if initial setup is needed (no users exist)
    // try {
    //   const status = await adminAuthAPI.getSetupStatus()
    //   if (status.needs_setup) {
    //     // Redirect to sign-up if no admin users exist
    //     throw redirect({
    //       to: '/sign-up',
    //     })
    //   }
    // } catch (error) {
    //   // Silently fail - don't block if API is down or during redirects
    //   // Only log in development mode
    //   if (import.meta.env.MODE === 'development') {
    //     console.debug('Setup status check failed:', error)
    //   }
    // }
  },
  component: () => {
    return (
      <>
        <NavigationProgress />
        <Outlet />
        <Toaster duration={5000} />
        {import.meta.env.MODE === 'development' && (
          <>
            <ReactQueryDevtools buttonPosition='bottom-left' />
            <TanStackRouterDevtools position='bottom-right' />
          </>
        )}
      </>
    )
  },
  notFoundComponent: NotFoundError,
  errorComponent: GeneralError,
})
