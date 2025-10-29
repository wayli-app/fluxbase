import { useLayout } from '@/context/layout-provider'
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarRail,
} from '@/components/ui/sidebar'
// import { AppTitle } from './app-title'
import { sidebarData } from './data/sidebar-data'
import { NavGroup } from './nav-group'
import { NavUser } from './nav-user'
import { TeamSwitcher } from './team-switcher'
import { getStoredUser } from '@/lib/auth'

export function AppSidebar() {
  const { collapsible, variant } = useLayout()

  // Get the logged-in user from localStorage
  const storedUser = getStoredUser()

  // Construct user data for NavUser component
  const user = storedUser
    ? {
        name: (storedUser.metadata?.name as string) || storedUser.email.split('@')[0],
        email: storedUser.email,
        avatar: (storedUser.metadata?.avatar as string) || '/avatars/shadcn.jpg',
      }
    : sidebarData.user // Fallback to default user if not logged in

  return (
    <Sidebar collapsible={collapsible} variant={variant}>
      <SidebarHeader>
        <TeamSwitcher teams={sidebarData.teams} />

        {/* Replace <TeamSwitch /> with the following <AppTitle />
         /* if you want to use the normal app title instead of TeamSwitch dropdown */}
        {/* <AppTitle /> */}
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
