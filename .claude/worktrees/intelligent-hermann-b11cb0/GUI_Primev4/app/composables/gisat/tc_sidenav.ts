import { ref } from 'vue';
const side_nav_config = useState("side_nav_config");

type MenuItem = {
    label: string;
    icon?: string;
    route?: string;
    children?: MenuItem[]; // Optional nested children
  };
  
  /**
   * Flattens a nested menu structure into a single array of menu items.
   * @param menuItems - The array of menu items, which may contain nested children.
   * @returns A flattened array of menu items.
   */
  function flattenMenu(menuItems: MenuItem[]): MenuItem[] {
    const result: MenuItem[] = [];
  
    for (const item of menuItems) {
      result.push(item); // Add the current item
      if (item.children) {
        // Recursively flatten the children
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
      return flattenedMenu[index].label;
    }
  
    return undefined; // Return undefined if the index is out of bounds
  }
  
  const menuItems = [
    {
        label: 'CFG Based Commands',
        icon: 'pi pi-fw pi-id-card',
        route: '/gisat/tc'
    },

    {
        label: 'BOA & PHASE',
        icon: 'pi pi-fw pi-id-card',
        route: '/gisat/tc/boaPhase'
    },
    {
        label: 'TC Files',
        icon: 'pi pi-fw pi-id-card',
        route: '/gisat/tc/files'
    },
    {
        label: 'Manual Commands',
        icon: 'pi pi-fw pi-id-card',
        route: '/gisat/tc/manualCommands'
    }
]
  

export function initMenu(index: number) {
    
    side_nav_config.value = {
        show_side_nav: true,
        app_name: "GISAT TC",
        logo_url: "/tc.gif",
        logo_width: "50%",
        items: menuItems,
        selected_item_label:getLabelByIndex(menuItems, index)
      };
}

export const wamp_topic = "com.tc_file.status"