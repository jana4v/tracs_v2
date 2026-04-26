/**
 * Shared Side Navigation Composable
 * Eliminates code duplication across modules (TC, PAPERT, TRACS, etc.)
 */

export interface MenuItem {
  label: string;
  icon?: string;
  route?: string;
  url?: string;
  target?: string;
  children?: MenuItem[];
}

export interface SideNavConfig {
  show_side_nav: boolean;
  app_name: string;
  logo_url: string;
  logo_width: string;
  items: MenuItem[];
  selected_item_label?: string;
}

/**
 * Flattens a nested menu structure into a single array of menu items.
 * @param menuItems - The array of menu items, which may contain nested children.
 * @returns A flattened array of menu items.
 */
function flattenMenu(menuItems: MenuItem[]): MenuItem[] {
  const result: MenuItem[] = [];

  for (const item of menuItems) {
    result.push(item);
    if (item.children) {
      result.push(...flattenMenu(item.children));
    }
  }

  return result;
}

/**
 * Retrieves the label of a menu item by its index.
 * @param menuItems - The array of menu items, which may contain nested children.
 * @param index - The index of the menu item to retrieve.
 * @returns The label of the menu item at the specified index, or undefined if the index is out of bounds.
 */
function getLabelByIndex(menuItems: MenuItem[], index: number): string | undefined {
  const flattenedMenu = flattenMenu(menuItems);

  if (index >= 0 && index < flattenedMenu.length) {
    return flattenedMenu[index]?.label;
  }

  return undefined;
}

/**
 * Composable for managing side navigation state
 * @param appName - The name of the application module
 * @param logoUrl - URL/path to the logo image
 * @param menuItems - Array of menu items to display
 * @param logoWidth - Optional logo width (default: "100%")
 * @returns Object containing initMenu function and side_nav_config state
 */
export function useSideNav(appName: string, logoUrl: string, menuItems: MenuItem[], logoWidth: string = "100%") {
  const side_nav_config = useState<SideNavConfig>("side_nav_config");

  /**
   * Initializes the side navigation menu
   * @param index - Index of the menu item to select by default
   */
  const initMenu = (index: number = 0) => {
    side_nav_config.value = {
      show_side_nav: true,
      app_name: appName,
      logo_url: logoUrl,
      logo_width: logoWidth,
      items: menuItems,
      selected_item_label: getLabelByIndex(menuItems, index)
    };
  };

  return {
    initMenu,
    side_nav_config,
    flattenMenu,
    getLabelByIndex
  };
}
