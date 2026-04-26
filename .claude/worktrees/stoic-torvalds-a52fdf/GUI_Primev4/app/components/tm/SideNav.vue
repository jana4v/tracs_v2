<script setup>
import { ref } from 'vue';
//import AppMenuItem from '@/layouts/AppMenuItem.vue';


const sidebar_minimal = computed(() => layoutState.staticMenuDesktopInactive.value && layoutConfig.menuMode.value === 'static');
const model = ref([
    {
        label: 'TM Application',
        items: [
            {
                label: 'Live Data',
                icon: 'pi pi-fw pi-id-card',
                to: '/tm'
            },

            {
                label: 'Database',
                icon: 'pi pi-fw pi-database',
                to: '/tm/database'
            },
            {
                label: 'Simulator',
                icon: 'pi pi-fw pi-id-card',
                to: '/gisat/tc/files'
            },
        ]
    }
]);
</script>

<template>
    <div class="container">
        <div class="layout-menu">
            <ul>
                <template v-for="(item, i) in model" :key="item">
                    <app-menu-item v-if="!item.separator" :item="item" :index="i"></app-menu-item>
                    <li v-if="item.separator" class="menu-separator"></li>
                </template>
            </ul>
        </div>
    </div>
</template>

<style lang="scss" scoped>
.container {
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    height: calc(100vh - 10rem);
}
</style>
