import type { MenuItem } from '../useSideNav'
import { useSideNav } from '../useSideNav'

const menuItems: MenuItem[] = [
  { label: 'Overview', icon: 'pi pi-home', route: '/iam' },
  { label: 'Users', icon: 'pi pi-users', route: '/iam/users' },
  { label: 'Roles', icon: 'pi pi-id-card', route: '/iam/roles' },
  { label: 'Permissions', icon: 'pi pi-lock', route: '/iam/permissions' },
  { label: 'Audit Logs', icon: 'pi pi-history', route: '/iam/audit' },
]

export const { initMenu, side_nav_config } = useSideNav(
  'Identity & Access',
  '/tc.gif',
  menuItems,
  '55%',
)
