<script setup>
import { ref } from 'vue';
//import AppMenuItem from '@/layouts/AppMenuItem.vue';
//import { useLayout } from '@/layouts/composables/layout';
//const { layoutConfig, layoutState, isSidebarActive } = useLayout();

const sidebar_minimal = computed(() => layoutState.staticMenuDesktopInactive.value && layoutConfig.menuMode.value === 'static');
const model = ref([
    {
        label: 'TRACS',
        items: [
            {
                label: 'Home',
                icon: 'pi pi-fw pi-home',
                to: '/tracs'
            },
            { label: 'Calibration',
              icon: 'fa-solid fa-scale-balanced', 
              
             items:[
                {
                    label: 'Calibration',
                    icon: 'fa-solid fa-ruler-horizontal',
                    to: '/tracs/calibration'
                },
                {
                    label: 'Copy Cal',
                    icon: 'fa-solid fa-copy',
                    to: '/tracs/calibration/copy_cal'
                },
                {
                    label: 'View Cal Data',
                    icon: 'fa-solid fa-binoculars',
                    to: '/tracs/calibration/view_cal'
                },
                {
                    label: 'TVAC Reference Cal',
                    icon: 'fa-solid fa-ruler-horizontal',
                    to: '/tracs/calibration/tvac_ref_cal'
                },
                
             ]
            
            },
            {
                label: 'Database',
                icon: 'pi pi-fw pi-database',
                to: '/tracs/database'
            },
            {
                label: 'Results Generation',
                icon: 'pi pi-fw pi-exclamation-circle',
                to: '/tracs/results'
            },



        ]
    },
    {
        label: 'Documentation',
        items: [
            {
                label: 'Documentation',
                icon: 'pi pi-fw pi-question',
                to: '/documentation'
            },
            {
                label: 'View Source',
                icon: 'pi pi-fw pi-search',
                url: 'https://github.com/primefaces/sakai-nuxt',
                target: '_blank'
            }
        ]
    }
]);


const { $wamp } = useNuxtApp();
const test_execution_status_store = useTestExecutionStatusStore();
$wamp.subscribe('com.tracs.status', (args) => {
test_execution_status_store.set_store(args[0]);
});



</script>

<template>
    
        <div v-if="!sidebar_minimal" class="layout-menu">
            <ul>
            <template v-for="(item, i) in model" :key="item">
                <app-menu-item v-if="!item.separator" :item="item" :index="i"></app-menu-item>
                <li v-if="item.separator" class="menu-separator"></li>
            </template>
        </ul>
        </div>
        <div v-else class="sidebar-minimal">
            <ul>
            <template v-for="(item, i) in model" :key="item">
                <app-menu-item v-if="!item.separator" :item="item" :index="i"></app-menu-item>
                
            </template>
        </ul>
        </div>
        <!-- <li>
            <a href="https://www.primefaces.org/primeblocks-vue/#/" target="_blank">
                <img src="/layout/images/banner-primeblocks.png" alt="Prime Blocks" class="w-full mt-3" />
            </a>
        </li> -->
   
</template>

<style lang="scss" scoped></style>
