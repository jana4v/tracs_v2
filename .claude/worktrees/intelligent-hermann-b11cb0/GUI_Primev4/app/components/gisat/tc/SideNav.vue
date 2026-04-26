<script setup>
import { ref } from 'vue';
import AppMenuItem from '@/layouts/AppMenuItem.vue';
import { useLayout } from '@/layouts/composables/layout';
const { layoutConfig, layoutState, isSidebarActive } = useLayout();

const sidebar_minimal = computed(() => layoutState.staticMenuDesktopInactive.value && layoutConfig.menuMode.value === 'static');
const model = ref([
    {
        label: 'TC Application',
        items: [
            {
                label: 'CFG Based Commands',
                icon: 'pi pi-fw pi-id-card',
                to: '/gisat/tc'
            },

            {
                label: 'BOA & PHASE',
                icon: 'pi pi-fw pi-id-card',
                to: '/gisat/tc/boaPhase'
            },
            {
                label: 'TC Files',
                icon: 'pi pi-fw pi-id-card',
                to: '/gisat/tc/files'
            },
            {
                label: 'Manual Commands',
                icon: 'pi pi-fw pi-id-card',
                to: '/gisat/tc/manualCommands'
            }
        ]
    }
]);

const environment = ref('');
useAPIFetch(`tc/getEnv`).then((res) => {
    environment.value = res.data.value;
});

const set_env = (data) => {
    useAPIFetch(`/tc/setEnv`, { method: 'post', body: [data] }).then((res) => {
        if (res.error.value != null) {
            environment.value = data == 'ntp' ? 'tvac' : 'ntp';
        }
    });
};
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
        <div class="card flex flex-column mt-5 pt-5">
            <div class="font-bold">Environment</div>
            <div class="col flex flex-wrap gap-2 pt-5">
                <div class="flex align-items-center">
                    <RadioButton v-model="environment" inputId="NTP" name="NTP" value="ntp" @click="set_env('ntp')"> </RadioButton>
                    <label for="NTP" class="ml-2">NTP</label>
                </div>
                <div class="flex align-items-center">
                    <RadioButton v-model="environment" inputId="TVAC" name="TVAC" value="tvac" @click="set_env('tvac')"> </RadioButton>
                    <label for="TVAC" class="ml-2">TVAC</label>
                </div>
            </div>
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
