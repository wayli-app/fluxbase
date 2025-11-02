import {
  LayoutDashboard,
  Database,
  Users,
  Settings,
  Palette,
  Wrench,
  Command,
  Zap,
  FileCode,
  FolderOpen,
  Radio,
  Key,
  Shield,
  Webhook,
  Activity,
  Sliders,
  Server,
} from 'lucide-react'
import { type SidebarData } from '../types'

export const sidebarData: SidebarData = {
  user: {
    name: 'Admin',
    email: 'admin@fluxbase.eu',
    avatar: '',
  },
  teams: [
    {
      name: 'Fluxbase',
      logo: Command,
      plan: 'Backend as a Service',
    },
  ],
  navGroups: [
    {
      title: 'General',
      items: [
        {
          title: 'Overview',
          url: '/',
          icon: LayoutDashboard,
        },
        {
          title: 'Tables',
          url: '/tables',
          icon: Database,
        },
        {
          title: 'Monitoring',
          url: '/monitoring',
          icon: Activity,
        },
        {
          title: 'Users',
          url: '/users',
          icon: Users,
        },
      ],
    },
    {
      title: 'API & Services',
      items: [
        {
          title: 'REST API',
          url: '/api/rest',
          icon: Zap,
        },
        {
          title: 'Realtime',
          url: '/realtime',
          icon: Radio,
        },
        {
          title: 'Storage',
          url: '/storage',
          icon: FolderOpen,
        },
        {
          title: 'Functions',
          url: '/functions',
          icon: FileCode,
        },
      ],
    },
    {
      title: 'Security',
      items: [
        {
          title: 'Authentication',
          url: '/authentication',
          icon: Shield,
        },
        {
          title: 'API Keys',
          url: '/api-keys',
          icon: Key,
        },
        {
          title: 'Webhooks',
          url: '/webhooks',
          icon: Webhook,
        },
      ],
    },
    {
      title: 'Configuration',
      items: [
        {
          title: 'App Settings',
          url: '/app-settings',
          icon: Sliders,
        },
        {
          title: 'System Settings',
          url: '/system-settings',
          icon: Server,
        },
      ],
    },
    {
      title: 'Settings',
      items: [
        {
          title: 'Profile',
          url: '/settings',
          icon: Settings,
        },
        {
          title: 'Account',
          url: '/settings/account',
          icon: Wrench,
        },
        {
          title: 'Appearance',
          url: '/settings/appearance',
          icon: Palette,
        },
      ],
    },
  ],
}
