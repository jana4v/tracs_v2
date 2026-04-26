<script setup>
import { ref,watch } from 'vue';
import { initMenu, wamp_topic } from "@/composables/papert/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
import { papert_table_config_func } from '@/components/papert/tableConfigurations';

definePageMeta({
  title: "PAPERT Application",
});

initMenu(1);

const { $db } = useNuxtApp();
const active = ref(0);

const table_name = ref('test_phase_colors');
const componentKey = ref(0);
const tab_items = ref([
    { id:0, label: 'Plot Colors', icon: 'pi pi-list' },
    { id:1, label: 'CFG Groups', icon: 'pi pi-chart-line' },
    { id:2, label: 'Report Configuration', icon: 'pi pi-list' },
    { id:3, label: 'Axis & Lables', icon: 'pi pi-inbox' }
]);

watch(active, (val) => {
    console.log(val)
if(val === 0){
    table_name.value = 'test_phase_colors';
    componentKey.value = 0;
} else if(val === 1){
    table_name.value = 'config_groups';
    componentKey.value = 1;
} else if(val === 2){
    table_name.value = 'report_config';
    componentKey.value = 2;
} else if(val === 3){
    componentKey.value = 3;
    table_name.value = 'param_table';
}
});





// Data fetched from the backend
const tableData = ref([]);

// Fetch data when the component is mounted
onMounted(() => {});

await $db.collection('users').add(
            {
                id: "sdasd",
                name: 'x Bill',
                age: 47
            },
            'mykey-2'
        );

await $db.collection('users')
            .doc('mykey-2')
            .get()
            .then((document) => {
                console.log(document);
            });
const value = ref(0);
const tabChanged = (val) => { active.value =val;  console.log(val)}   

</script>

<template>
     <div class="mt-10 min-h-screen w-full">
        <AppName appname="PAPERT"></AppName>
    <div class="">
        <Tabs  :value="value">
            <TabList>
                <Tab v-for="tab in tab_items" :key="tab.label"  :value="tab.id" @click="tabChanged(tab.id)">
                  <span>{{ tab.label }}</span>
                </Tab>
            </TabList>
           
        </Tabs>
    </div>

    <div v-if="active === 0 || active === 1 || active === 2 || active === 3" class="grid pt-4">
        <div class="">
            <div >
                      
            <AgGridTable
                ref="tx_table_ref"
                
                table_height="70vh"
                table_width="100%"
                app_name="papert"
                :use_local_data="false"
                :table_name="table_name"
                row_group_panel_show="always"
                :table_configuration_func="papert_table_config_func"
            ></AgGridTable>
        </div>
        </div>
    </div>
</div>
    
   
</template>

<style>


</style>
