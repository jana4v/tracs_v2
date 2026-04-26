<template>

  <ScrollPanel style="height: calc(100vh - 9rem);padding: 0.5rem;" class="card">
    <div class="flex flex-wrap  gap-3 my-2 ml-2">
      <Button icon="pi pi-plus" text rounded outlined severity="info" @click="expandAll" />
      <Button icon="pi pi-minus" text rounded outlined severity="info" @click="collapseAll" />
    </div>
    <PanelMenu v-model:expandedKeys="expandedKeys" :model="items" class="w-full md:w-17rem" />


  </ScrollPanel>



</template>

<script setup>
import { ref } from "vue";
import { table_details } from "./tableConfigurations"
const emit = defineEmits();
const expandedKeys = ref({});

const select = (item) => {
  let path = handleClick(item.item.key, items.value);
  emit('table-selected', path);
}



const transformTableDetails = (details) => {
  return details.map((item, index) => {
    const [label, value] = Object.entries(item)[0];
    const name = label.toLowerCase().replaceAll(' ', '_');
    const icon = 'pi pi-fw pi-database';

    const items = Array.isArray(value)
      ? value.map((subItem, subIndex) => ({
        key: `${index}_${subIndex}`,
        label: subItem,
        name: subItem.toLowerCase().replaceAll(' ', '_'),
        icon: 'pi pi-fw pi-external-link',
        command: select,
      }))
      : Object.entries(value).map((subItem, subIndex) => {
        const [subLabel, subValue] = subItem;
        return {
          key: `${index}_${subIndex}`,
          label: subLabel,
          name: subLabel.toLowerCase().replaceAll(' ', '_'),
          icon: 'pi pi-fw pi-database',
          items: subValue.map((subSubItem, subSubIndex) => ({
            key: `${index}_${subIndex}_${subSubIndex}`,
            label: subSubItem,
            name: subSubItem.toLowerCase().replaceAll(' ', '_'),
            icon: 'pi pi-fw pi-external-link',
            command: select,
          }))
        };
      });

    return {
      key: index.toString(),
      label,
      name,
      icon,
      items,
    };
  });
};

const items = ref();
items.value = transformTableDetails(table_details);


const expandAll = () => {
  for (let node of items.value) {
    expandNode(node);
  }

  expandedKeys.value = { ...expandedKeys.value };
};

const collapseAll = () => {
  expandedKeys.value = {};
};

const expandNode = (node) => {
  if (node.items && node.items.length) {
    expandedKeys.value[node.key] = true;

    for (let child of node.items) {
      expandNode(child);
    }
  }
};


const handleClick = (selected_key, menu) => {
  const path = findPath(menu, selected_key);
  if (path) {
    return path;
  } else {
    return '';
  }
}
const findPath = (menu, key) => {
  for (const item of menu) {
    if (item.key === key) {
      return item.name;
    }
    if (item.items) {
      const path = findPath(item.items, key);
      if (path) {
        return `${item.name}#${path}`;
      }
    }
  }
  return null;
}


</script>