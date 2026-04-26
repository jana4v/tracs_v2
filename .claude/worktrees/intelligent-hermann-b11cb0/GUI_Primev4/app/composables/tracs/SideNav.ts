import { ref } from 'vue';
const side_nav_config = useState("side_nav_config");

type MenuItem = {
    label: string;
    icon?: string;
    route?: string;
    items?: MenuItem[]; // Optional nested children
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
  
  // Example usage:
  const menuItems: MenuItem[] = [
    {
      label: 'Home',
      icon: 'pi pi-fw pi-home',
      route: '/tracs'
    },
    {
      label: 'Calibration',
      icon: 'fa-solid fa-scale-balanced',
      items: [
        {
          label: 'Calibration',
          icon: 'fa-solid fa-ruler-horizontal',
          route: '/tracs/calibration'
        },
        {
          label: 'Copy Cal',
          icon: 'fa-solid fa-copy',
          route: '/tracs/calibration/copy_cal'
        },
        {
          label: 'View Cal Data',
          icon: 'fa-solid fa-binoculars',
          route: '/tracs/calibration/view_cal'
        },
        {
          label: 'TVAC Reference Cal',
          icon: 'fa-solid fa-ruler-horizontal',
          route: '/tracs/calibration/tvac_ref_cal'
        }
      ]
    },
    {
      label: 'Database',
      icon: 'pi pi-fw pi-database',
      route: '/tracs/database'
    },
    {
      label: 'Results Generation',
      icon: 'pi pi-fw pi-exclamation-circle',
      route: '/tracs/results'
    },
    {
      label: 'Documentation',
      icon: 'pi pi-fw pi-question',
      route: '/documentation'
    }
  
  ];
  
   


export function initMenu(index: number) {
    
    side_nav_config.value = {
        show_side_nav: true,
        app_name: "TRACS",
        logo_url: "/tc.gif",
        logo_width: "50%",
        items: menuItems,
        selected_item_label:getLabelByIndex(menuItems, index)
      };
}

export const wamp_topic = "com.tracs.status"