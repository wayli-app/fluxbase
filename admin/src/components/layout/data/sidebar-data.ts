import {
  LayoutDashboard,
  Database,
  Users,
  Settings,
  Palette,
  Command,
  Zap,
  FileCode,
  FolderOpen,
  Radio,
  Key,
  Shield,
  Webhook,
  Activity,
  Server,
  Code,
  Mail,
  ShieldCheck,
  ListTodo,
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
          title: 'SQL Editor',
          url: '/sql-editor',
          icon: Code,
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
        {
          title: 'Jobs',
          url: '/jobs',
          icon: ListTodo,
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
          title: 'Security Settings',
          url: '/security-settings',
          icon: ShieldCheck,
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
          title: 'System Settings',
          url: '/system-settings',
          icon: Server,
        },
        {
          title: 'Email Settings',
          url: '/email-settings',
          icon: Mail,
        },
      ],
    },
    {
      title: 'Settings',
      items: [
        {
          title: 'Account',
          url: '/settings',
          icon: Settings,
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
