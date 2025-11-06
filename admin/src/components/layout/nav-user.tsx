import { SidebarMenu, SidebarMenuItem } from '@/components/ui/sidebar'

// Component to display user name in sidebar footer
type NavUserProps = {
  user: {
    name: string
    email: string
    avatar: string
  }
}

export function NavUser({ user }: NavUserProps) {
  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <div className='flex items-center gap-2 px-2 py-1.5'>
          <div className='grid flex-1 text-start text-sm leading-tight'>
            <span className='truncate font-semibold'>{user.name}</span>
          </div>
        </div>
      </SidebarMenuItem>
    </SidebarMenu>
  )
}
