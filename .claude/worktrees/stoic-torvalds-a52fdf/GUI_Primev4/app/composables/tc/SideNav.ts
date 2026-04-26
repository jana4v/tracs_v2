import { useSideNav, type MenuItem } from '../useSideNav';

const menuItems: MenuItem[] = [
  {
    label: "CFG Based Commands",
    icon: "pi pi-microsoft",
    route: "/tc",
  },
  {
    label: "Data Commands",
    icon: "pi pi-calculator",
    route: "/tc/dataCommands",
  },
  {
    label: "Manual Commands",
    icon: "pi pi-eye",
    route: "/tc/manualCommands",
  },
  {
    label: "TC Files",
    icon: "i-carbon-document",
    route: "/tc/files",
  },
  {
    label: "Utility",
    icon: "pi pi-box",
    route: "/tc/utility",
  },
];

export const wamp_topic = "com.tc_file.status";

export const { initMenu, side_nav_config } = useSideNav(
  "TeleCommand",
  "/tc.gif",
  menuItems,
  "60%" // Logo width reduced to 50%
);
