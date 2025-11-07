import { Database } from 'lucide-react'
import { getStoredUser, type AdminUser, type DashboardUser } from '@/lib/auth'
import { useLayout } from '@/context/layout-provider'
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
import { sidebarData } from './data/sidebar-data'
import { NavGroup } from './nav-group'
import { NavUser } from './nav-user'

// Type guard to check if user is a DashboardUser
function isDashboardUser(
  user: AdminUser | DashboardUser
): user is DashboardUser {
  return 'full_name' in user
}

export function AppSidebar() {
  const { collapsible, variant } = useLayout()

  // Get the logged-in user from localStorage
  const storedUser = getStoredUser()

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

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <SidebarMenu>
          <SidebarMenuItem>
            <SidebarMenuButton
              size='lg'
              className='data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground'
            >
              <div className='bg-sidebar-primary text-sidebar-primary-foreground flex aspect-square size-8 items-center justify-center rounded-lg'>
                <Database className='size-4' />
              </div>
              <div className='grid flex-1 text-start text-sm leading-tight'>
                <span className='truncate font-semibold'>Fluxbase</span>
                <span className='truncate text-xs'>Backend-as-a-Service</span>
              </div>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarHeader>
      <SidebarContent>
        {sidebarData.navGroups.map((props) => (
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
