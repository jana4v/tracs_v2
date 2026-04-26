
<script setup lang="ts">
// const { $api } = useContext();
// api.get('/api/database').then((res) => {
//     console.log(res);
// });

//import {useAPIFetch} from '~/composables/useAPIFetch'
import { ref } from "vue";
import { tracs_table_config_func } from "@/components/Tracs/Database/tableConfigurations";



const selectedSource = ref();
const selectedSubsystem = ref();
const source = ref([]);
const subsystem = ref([]);
const selected_db_table = ref("");
const title = ref("");
const componentKey = ref(0);
const is_systems_group = ref(true);
const system_group_tbl_name = ref("");
const group_name = ref("");
const table_name = ref("");
const forceRerender = () => {
    componentKey.value += 1;
};

const { data: telemetry_sources } = await useAPIFetch(`gui/getTelemetrySources`)
source.value = telemetry_sources.value
console.log(source.value)
const { data: sub_system_names } = await useAPIFetch(`gui/getSubSystemNames`)
subsystem.value = sub_system_names.value


const db_table_selected_from_menu = (path: string) => {
    selected_db_table.value = path;
    title.value = path.replaceAll('#', '/');
    group_name.value = path.split("#")[0];
    table_name.value = path.split("#")[1];
    if (path.indexOf("systems#") == 0) {
        is_systems_group.value = true;
        // if(table_name.value == 'transponder'){
        // is_systems_group.value = false;
        // forceRerender();
        //  }

} else {
    is_systems_group.value = false;
forceRerender();
    }
}

onMounted(async () => {

})





// import { storeToRefs } from 'pinia'
// const store = useCounterStore()
// const { name, doubleCount } = storeToRefs(store)
// const { increment } = store


</script>


<template>
    <div class="layout-sidebar"><tracs-side-nav></tracs-side-nav></div>

    <div class="grid">
        <div class="col-2 ml-0 pl-0">
            <tracs-database-tables-menu @table-selected="db_table_selected_from_menu"></tracs-database-tables-menu>
        </div>
        <!-- <ScrollPanel style="height: calc(100vh - 9rem);padding: 0.5rem;" class="card col-10"> -->
        <div class="col-10 card mt-2" style="height: calc(100vh - 9rem);padding: 0.5rem;">
            <div class="my-3">
                <span class="text-4xl font-bold font-italic text-primary capitalize">{{ title }}</span>
            </div>
            <AgGridTable v-if="!is_systems_group" :key="componentKey" table_height="90vh" table_width="100%"
                app_name="tracs" :table_name="selected_db_table"
                :table_configuration_func="tracs_table_config_func"></AgGridTable>
            <h3 v-if="group_name.length == 0">No table selected </h3>
            <TracsDatabaseSystemsTransmitters v-if="is_systems_group && table_name == 'transmitter'">
            </TracsDatabaseSystemsTransmitters>
            <TracsDatabaseSystemsReceivers v-if="is_systems_group && table_name == 'receiver'">
            </TracsDatabaseSystemsReceivers>
            <TracsDatabaseSystemsTransponders v-if="is_systems_group && table_name == 'transponder'">
            </TracsDatabaseSystemsTransponders>

            <!-- </ScrollPanel> -->
        </div>
        <!-- </div> -->

        <!-- </div> -->

    </div>
</template>

