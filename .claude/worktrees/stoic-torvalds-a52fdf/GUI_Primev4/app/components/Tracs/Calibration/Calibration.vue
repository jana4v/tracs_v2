
<script setup>
import { ref, watch } from "vue";
import { tracs_table_config_func } from "@/components/Tracs/Database/tableConfigurations";

const { $wamp } = useNuxtApp();
const home_store = useMyHomeStore();
const calibration_store = useMyCalibrationStore();

const selected_cal_type = ref(calibration_store.selected_cal_type);
const selected_cal_id = ref(calibration_store.selected_cal_id);
const cal_ids = ref([]); 
const cal_types = ref([]);

const get_url = ref("/tracs/calibration/channel_selection/get");

const table_name = ref("calibration#channel_selection#up_link");

const componentKey = ref(0);




const filteredResults = ref([])


const search = (event) => {

      // Filter the results based on the query
      filteredResults.value = cal_ids.value.filter(option => {
        return option.name.toLowerCase().search(event.query.toLowerCase()) != -1;
      });

    }

useAPIFetch("tracs/get_cal_types").then(res => {
  if (res.error.value == null && res.data.value.length > 0) {
    cal_types.value = res.data.value.map((opt) => ({ value: opt }));
  }
})

useAPIFetch("tracs/get_cal_ids").then(res => {
  if (res.error.value == null && res.data.value.length > 0) {
    cal_ids.value = res.data.value.map((opt) => ({ value: opt }));
  }
})


watch(() => selected_cal_type.value, (newValue, oldValue) => {
  calibration_store.set_selected_cal_type(selected_cal_type.value);
  get_url.value = `/tracs/calibration/channel_selection/get?cal_type=${newValue}&cal_id=${selected_cal_id.value}`;
  if (selected_cal_type.value.search("up_link") == 0){
      table_name.value = "calibration#channel_selection#up_link";
  } else if((selected_cal_type.value.search("down_link") == 0)){
    table_name.value = "calibration#channel_selection#down_link";
  } else{
    table_name.value = "calibration#channel_selection#up_link";
  }

  
  componentKey.value +=1;
})

watch(() => selected_cal_id.value, (newValue, oldValue) => {
  get_url.value = `/tracs/calibration/channel_selection/get?cal_type=${selected_cal_type.value}&cal_id=${newValue}`;
  componentKey.value +=1;
});




onMounted(async () => {

})

const tx_tbl_cell_value_changed = () => {
  console.log("Hello from tx_tbl_cell_value_changed")

}



</script>


<template>
  <div class="card">
    <div class="grid">
      <div class="col-12">
        <h1 class="text-2xl text-primary-600 font-bold">Calibration Channel Selection</h1>
      </div>
    </div>
    <div class="grid pt-4">
      <div class="col-4">
        <!-- <div class="xcard" style="min-height: calc(100vh - 9rem);"> -->
        <p class="mb-0 text-xl w-10">CAL ID</p>
        <AutoComplete v-model="selected_cal_id" 
        :suggestions="filteredResults" 
        @complete="search" 
        field="name" 
        placeholder=" Enter/Select Cal ID"
        :virtualScrollerOptions="{ itemSize: 38 }"
        dropdown
        ></AutoComplete>
      
      </div>
      <div class="col-4">
        <p class="mb-0 text-xl w-10">CAL Type</p>
        <Select v-model="selected_cal_type" :options="cal_types" filter optionLabel="value"
          optionValue="value" inputId="Select_cal_type" placeholder="Select Cal Type" class="w-full">
        </Select>
      </div>

      <div class="col-4">
        <p class="mb-0 text-xl w-10">DUT Label</p>
        <Select v-model="selected_cal_type" :options="sub_test_phase_names" filter optionLabel="value"
          optionValue="value" inputId="Select_sub_test_phase" placeholder="Enter DUT Label" class="w-full">
        </Select>
      </div>
     

    </div>
    <div class="grid pt-4">
      <div class="col-12">
        <div>
          <AgGridTable ref="tx_table_ref" :key="componentKey" table_height="80vh" table_width="100%" app_name="tracs"
            :use_local_data="false" :table_name="table_name" :get_url="get_url"
            @cell-value-changed="tx_tbl_cell_value_changed"
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

