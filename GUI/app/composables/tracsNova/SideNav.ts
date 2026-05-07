import { useSideNav, type MenuItem } from '../useSideNav';

const menuItems: MenuItem[] = [
  {
    label: 'Measure',
    icon: 'pi pi-chart-line',
    route: '/tracsNova',
  },
  {
    label: 'Calibration',
    icon: 'pi pi-sliders-h',
    route: '/tracsNova/calibration',
  },
  {
    label: 'Test Results',
    icon: 'pi pi-check-square',
    route: '/tracsNova/test-results',
  },
  {
    label: 'Database',
    icon: 'pi pi-database',
    route: '/tracsNova/database',
  },
  {
    label: 'Link Support',
    icon: 'pi pi-link',
    route: '/tracsNova/link-support',
  },
];

export const { initMenu, side_nav_config } = useSideNav(
  'TRACS-Nova',
  '/tc.gif',
  menuItems,
  '55%'
);
