import { useSideNav, type MenuItem } from '../useSideNav';

const menuItems: MenuItem[] = [
  {
    label: "Generate PPT",
    icon: "pi pi-microsoft",
    route: "/papert",
  },
  {
    label: "Settings",
    icon: "pi pi-cog",
    route: "/papert/settings",
  },
];

export const wamp_topic = "com.papert.status";

export const { initMenu, side_nav_config } = useSideNav(
  "PAPERT",
  "/presentation.webp",
  menuItems
);
