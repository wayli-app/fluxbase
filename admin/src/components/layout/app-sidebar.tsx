import { useTenantStore } from '@/stores/tenant-store'
import { getStoredUser, type AdminUser, type DashboardUser } from '@/lib/auth'
import { useLayout } from '@/context/layout-provider'
import { Badge } from '@/components/ui/badge'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
  SidebarMenu,
  SidebarMenuItem,
  SidebarMenuButton,
} from '@/components/ui/sidebar'
import { sidebarData, filterSidebarForContext } from './data/sidebar-data'
import { NavGroup } from './nav-group'
import { NavUser } from './nav-user'

// Type guard to check if user is a DashboardUser
function isDashboardUser(
  user: AdminUser | DashboardUser
): user is DashboardUser {
  return 'full_name' in user
}

// Check if user is an instance admin
function isInstanceAdmin(user: AdminUser | DashboardUser | null): boolean {
  if (!user) return false
  // Check for role property (may not exist on all DashboardUser objects)
  if ('role' in user && user.role) {
    return user.role === 'instance_admin'
  }
  return false
}

export function AppSidebar() {
  const { collapsible, variant } = useLayout()

  // Get the logged-in user from localStorage
  const storedUser = getStoredUser()

  // Get tenant context from store
  const { actingAsTenantAdmin } = useTenantStore()

  // Construct user data for NavUser component
  // Handle both AdminUser (metadata.name) and DashboardUser (full_name) types
  const user = storedUser
    ? {
        name: isDashboardUser(storedUser)
          ? storedUser.full_name || storedUser.email.split('@')[0]
          : (storedUser.metadata?.name as string) ||
            storedUser.email.split('@')[0],
        email: storedUser.email,
        avatar: isDashboardUser(storedUser)
          ? storedUser.avatar_url || ''
          : (storedUser.metadata?.avatar as string) || '',
      }
    : sidebarData.user // Fallback to default user if not logged in

  // Filter sidebar based on user context
  const filteredNavGroups = filterSidebarForContext(sidebarData.navGroups, {
    isInstanceAdmin: isInstanceAdmin(storedUser),
    actingAsTenantAdmin,
  })

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size='lg'
              className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
            >
              <img
                src='/admin/images/logo-icon.svg'
                alt='Fluxbase'
                className='size-8 rounded-xl bg-white/80 backdrop-blur-sm dark:bg-white/80 dark:backdrop-blur-md'
              />
              <div className='grid flex-1 text-start text-sm leading-tight'>
                <span className='flex items-center gap-2 truncate font-semibold'>
                  Fluxbase
                  <Badge variant='outline' className='px-1.5 py-0 text-[10px]'>
                    Beta
                  </Badge>
                </span>
                <span className='truncate text-xs'>Backend-as-a-Service</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        {filteredNavGroups.map((props) => (
          <NavGroup key={props.title} {...props} />
        ))}
      </SidebarContent>
      <SidebarFooter>
        <NavUser user={user} />
      </SidebarFooter>
      <SidebarRail />
    </Sidebar>
  )
}
