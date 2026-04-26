
<script setup>
// const { $api } = useContext();
// api.get('/api/database').then((res) => {
//     console.log(res);
// });

//import {useAPIFetch} from '~/composables/useAPIFetch'
import { ref, watch } from "vue";
import { tracs_table_config_func } from "@/components/Tracs/Database/tableConfigurations";
// const { $wamp } = useNuxtApp();
// const home_store = useMyHomeStore();

// const selected_test_phase = ref(home_store.selected_test_phase);
// const selected_sub_test_phase = ref(home_store.selected_sub_test_phase);
// const selected_test_plan = ref(home_store.selected_test_plan);
const test_phase_names = ref([]);
const sub_test_phase_names = ref([]);
const test_plan_names = ref([]);
const tx_plan_url = ref("");
const rx_plan_url = ref("");
const tp_plan_url = ref("");

const tx_plan_tbl = "test_plan#transmitter#test_selection";
const rx_plan_tbl = "test_plan#receiver#test_selection";
const tp_plan_tbl = "test_plan#transponder#test_selection";
const componentKey = ref(0);
const counter = ref(0);

const tx_table_ref = ref(null);
const rx_table_ref = ref(null);
const tp_table_ref = ref(null);


useAPIFetch(`tracs/get_test_phase_names`, { method: "GET" }).then(
  async (res) => {
    if (res.error == null && res.data?.length > 0) {
      test_phase_names.value = res.data.value.map((opt) => ({ value: opt }));
    } else {
      const errorDetail = res.error?.data?.detail || res.error?.message || 'Backend service is not available';
      let msg = {
        summary: "Failed to Get Test Phase Names from Database",
        status: `Failed to Get Test Phase Names from Database Error: ${errorDetail}`,
        progress: "0",
      };
      await publishToWampTopic(msg, wamp_topic).catch(e => console.error('WAMP publish failed:', e));
    }
  }
).catch(async (error) => {
  console.error('Error fetching test phase names:', error);
  let msg = {
    summary: "Failed to Get Test Phase Names",
    status: `Network error: ${error.message || 'Backend service is not available'}`,
    progress: "0",
  };
  await publishToWampTopic(msg, wamp_topic).catch(e => console.error('WAMP publish failed:', e));
});


// useAPIFetch("tracs/get_test_phase_names").then((res) => {
//   console.log(res);
//   if (res.error.value == null && res.data.value.length > 0) {
//     test_phase_names.value = res.data.value.map((opt) => ({ value: opt }));
//   }
// });

// useAPIFetch("tracs/get_sub_test_phase_names").then((res) => {
//   if (res.error.value == null && res.data.value.length > 0) {
//     sub_test_phase_names.value = res.data.value.map((opt) => ({ value: opt }));
//   }
// });

// useAPIFetch("tracs/test_plan/plan_names/get").then((res) => {
//   if (res.error.value == null && res.data.value.length > 0) {
//     test_plan_names.value = res.data.value.map((opt) => ({ value: opt.plan_name }));
//   }
// });

// const transmitter_test_names = ref([]);
// const receiver_test_names = ref([]);
// const transponder_test_names = ref([]);
// useAPIFetch("tracs/get_test_paramater_names").then((res) => {
//   if (res.error.value == null) {
//     transmitter_test_names.value = res.data.value["transmitter"];
//     receiver_test_names.value = res.data.value["receiver"];
//     transponder_test_names.value = res.data.value["transponder"];
//   }
// });


const active_tab_number = ref(0);
const tab_menu_items = ref([
  { label: 'Transmitter', icon: 'pi pi-home' },
  { label: 'Receiver', icon: 'pi pi-chart-line' },
  { label: 'Transponder', icon: 'pi pi-list' },

]);

const checkboxStates = ref([true,false,false]);

// watch(() => active_tab_number.value, (newValue, oldValue) => {
//   counter.value += 1;
// });

// watch(() => selected_test_phase.value, (newValue, oldValue) => {
//   home_store.set_selected_test_phase(selected_test_phase.value);
// })

// watch(() => selected_sub_test_phase.value, (newValue, oldValue) => {
//   home_store.set_selected_sub_test_phase(selected_sub_test_phase.value);
// })

// watch(() => selected_test_plan.value, (newValue, oldValue) => {
//   home_store.set_selected_test_plan(selected_test_plan.value);
//   sessionStorage.removeItem(tx_plan_tbl);
//   sessionStorage.removeItem(rx_plan_tbl);
//   sessionStorage.removeItem(tp_plan_tbl);
//   tx_plan_url.value = `tracs/test_plan/transmitter/test_selection/${selected_test_plan.value}`
//   rx_plan_url.value = `tracs/test_plan/receiver/test_selection/${selected_test_plan.value}`
//   tp_plan_url.value = `tracs/test_plan/transponder/test_selection/${selected_test_plan.value}`
//   checkboxStates.value = [true, true, true];
//   componentKey.value += 1;
//   active_tab_number.value = 1;
//   setTimeout(() => {
//     active_tab_number.value = 2;
//   }, 200);
//   setTimeout(() => {
//     active_tab_number.value = 0;
//   }, 400);
//   setTimeout(() => {
//     tx_tbl_cell_value_changed();
//     rx_tbl_cell_value_changed();
//     tp_tbl_cell_value_changed();
//   }, 1000);
// });

// const check_box_selection_changed = (index) => {
//   checkboxStates.value[index] = !checkboxStates.value[index];
//   if (!checkboxStates.value[0]) {
//     home_store.set_is_tx_plan_selected(false);
//     home_store.set_tx_plan_selected_count(0);
//   } else {
//     home_store.set_is_tx_plan_selected(true);
//     tx_tbl_cell_value_changed();
//   }
//   if (!checkboxStates.value[1]) {
//     home_store.set_is_rx_plan_selected(false);
//     home_store.set_rx_plan_selected_count(0);
//   } else {
//     home_store.set_is_rx_plan_selected(true);
//     rx_tbl_cell_value_changed();
//   }
//   if (!checkboxStates.value[2]) {
//     home_store.set_is_tp_plan_selected(false);
//     home_store.set_tp_plan_selected_count(0);
//   } else {
//     home_store.set_is_tp_plan_selected(true);
//     tp_tbl_cell_value_changed();
//   }
// };


// onMounted(async () => {
//   setTimeout(() => {
//     tx_tbl_cell_value_changed();
//     rx_tbl_cell_value_changed();
//     tp_tbl_cell_value_changed();
//   }, 1000);
// })

// const tx_tbl_cell_value_changed = () => {
//   console.log("Hello from tx_tbl_cell_value_changed")
//   let tx_tbl_data = JSON.parse(sessionStorage.getItem(tx_plan_tbl) || "[]");
//   home_store.set_tx_plan(tx_tbl_data);
//   let tx_selected_count = 0
//   home_store.tx_plan.forEach(item => {
//     transmitter_test_names.value.forEach(test_name => {
//       if (item[test_name] === 1) {
//         tx_selected_count += 1;
//       }
//     });

//   });
//   home_store.set_tx_plan_selected_count(tx_selected_count);
// }

// const rx_tbl_cell_value_changed = () => {
//   console.log("Hello from rx_tbl_cell_value_changed")
//   let rx_tbl_data = JSON.parse(sessionStorage.getItem(rx_plan_tbl) || "[]");
//   home_store.set_rx_plan(rx_tbl_data);
//   let rx_selected_count = 0
//   home_store.rx_plan.forEach(item => {
//     receiver_test_names.value.forEach(test_name => {
//       if (item[test_name] === 1) {
//         rx_selected_count += 1;
//       }
//     });
//   });
//   home_store.set_rx_plan_selected_count(rx_selected_count);
// }

// const tp_tbl_cell_value_changed = () => {
//   console.log("Hello from tp_tbl_cell_value_changed")
//   let tp_tbl_data = JSON.parse(sessionStorage.getItem(tp_plan_tbl) || "[]");
//   home_store.set_tp_plan(tp_tbl_data);
//   let tp_selected_count = 0
//   home_store.tp_plan.forEach(item => {


//     transponder_test_names.value.forEach(test_name => {
//       if (item[test_name] === 1) {
//         tp_selected_count += 1;
//       }
//     });
//   });
//   home_store.set_tp_plan_selected_count(tp_selected_count);
// }

// const test_execution_status_store = useTestExecutionStatusStore();
// watch(() => test_execution_status_store.$state.reload_test_plan_table, (newValue, oldValue) => {
// let data = test_execution_status_store.$state.updated_test_plan;

// if(test_execution_status_store.$state.system == "Transmitter"){
//   sessionStorage.setItem(tx_plan_tbl, JSON.stringify(data));
//   active_tab_number.value = 0;
//   componentKey.value += 1;
 
  

// };
// if(test_execution_status_store.$state.system == "Receiver"){
//   sessionStorage.setItem(rx_plan_tbl, JSON.stringify(data));
//   active_tab_number.value = 1;
//   componentKey.value += 1;

 
// };
// if(test_execution_status_store.$state.system == "Transponder"){
//   sessionStorage.setItem(tp_plan_tbl, JSON.stringify(data));
//   active_tab_number.value = 2;
//   componentKey.value += 1;
 
// }
// })


</script>


<template>
  <div class="">
    <div class="text-2xl text-center text-primary-600 font-bold">Test Selection</div>
    <hr>
     <div class="grid md:grid-cols-3 gap-2 sm:grid-cols-1 mx-2">
      <div class="">
        <!-- <div class="xcard" style="min-height: calc(100vh - 9rem);"> -->
        <p class="mb-0 text-xl ">Test Phase</p>
        <Select v-model="selected_test_phase" :options="test_phase_names" :filter="false" inputId="select_test_phase"
          optionLabel="value" optionValue="value" placeholder=" Select Test Phase" class="w-full">
        </Select>

      </div>
      <div class="">
        <p class="mb-0 text-xl">Sub Test Phase</p>
        <Select v-model="selected_sub_test_phase" :options="sub_test_phase_names" filter optionLabel="value"
          optionValue="value" inputId="Select_sub_test_phase" placeholder="Select Subsystem" class="w-full">
        </Select>
      </div>

      <div class="">
        <p class="mb-0 text-xl">Test Plan</p>
        <Select v-model="selected_test_plan" :options="test_plan_names" filter optionLabel="value" optionValue="value"
          inputId="Select_test_plan" placeholder="Select Test Plan" class="w-full">
        </Select>
      </div>
    </div>
    <div class="grid pt-4">
      <div class="col-12">
        <div class="flex align-items-center ">
          <div v-for="(item, index) in tab_menu_items" :key="index" :class="['flex align-items-center justify-content-center p-2',
            { 'font-bold text-primary-600 border-bottom-2 border-primary-600': active_tab_number === index }]"
            @click="active_tab_number = index">
            <i :class="[item.icon, 'p-mr-2']"></i>
            <span class="ml-2 text-2xl text-secondary-200"> {{ item.label }} </span>
            <Checkbox @click.stop="check_box_selection_changed(index)" class="ml-3 mr-2 sys_type_selection"
              v-model="checkboxStates[index]" :binary="true" />
          </div>
        </div>

        <!-- <h1>{{ active_tab_number }}</h1> -->

        <div v-if="active_tab_number === 0">
          <AgGridTable ref="tx_table_ref" :key="componentKey" table_height="80vh" table_width="100%" app_name="tracs"
            :use_local_data="true" :table_name="tx_plan_tbl" :get_url="tx_plan_url"
            @cell-value-changed="tx_tbl_cell_value_changed"
            :table_configuration_func="tracs_table_config_func"></AgGridTable>
        </div>

        <div v-if="active_tab_number === 1">
          <AgGridTable ref="rx_table_ref" :key="componentKey" table_height="80vh" :use_local_data="true" table_width="100%"
            app_name="tracs" :table_name="rx_plan_tbl" :get_url="rx_plan_url"
            @cell-value-changed="rx_tbl_cell_value_changed"
            :table_configuration_func="tracs_table_config_func"></AgGridTable>

        </div>
        <div v-if="active_tab_number === 2">
          <AgGridTable ref="tp_table_ref" :key="componentKey" table_height="80vh" :use_local_data="true" table_width="100%"
            app_name="tracs" :table_name="tp_plan_tbl" :get_url="tp_plan_url"
            @cell-value-changed="tp_tbl_cell_value_changed"
            :table_configuration_func="tracs_table_config_func"></AgGridTable>
        </div>
      </div>
    </div>
  </div>
</template>

<style lang="scss">
.sys_type_selection .p-checkbox-icon {
  color: #1bd13f !important;
  font-size: xx-large !important;
  font-weight: bolder !important;
  // background-color: #3a453f !important;
}
</style>

