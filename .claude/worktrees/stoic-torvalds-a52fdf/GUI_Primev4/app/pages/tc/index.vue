<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/tc/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

definePageMeta({
  title: "Telecommand",
});

initMenu(0);
const cfgNumberString = ref("");
const testProcedure = ref("");
const showTestProcedure = ref(false);
const showExecuteTestProcedureButton = ref(false);
const test_parameter_names = ref([]);
const selected_test_parameter = ref("");
const boa = ref([]);
const selected_boa = ref("");
const generated_test_procedure = ref({});
const statusStore = tCstore();

// Load test parameter names
(async () => {
  const data = await useSimpleAPIFetch(
    `/tc/get_test_parameter_names`, 
    { method: "GET" },
    "Failed to Get Test Parameter Names from Database",
    wamp_topic
  );
  if (data && Array.isArray(data) && data.length > 0) {
    test_parameter_names.value = data.map((opt) => ({ name: opt }));
  }
})();

// Load BOA column names
(async () => {
  const data = await useSimpleAPIFetch(
    `/tc/get_cfg_boa_col_names`, 
    { method: "GET" },
    "Failed to Get Boa Column Names from Database",
    wamp_topic
  );
  if (data && Array.isArray(data) && data.length > 0) {
    boa.value = data.map((opt) => ({ name: opt }));
  }
})();

const gnerateTestProcedureToTurnOff = async () => {
  testProcedure.value = "";
  showTestProcedure.value = false;
  showExecuteTestProcedureButton.value = false;
  let body = {
    configs_str: cfgNumberString.value,
    request_is_to_turn_on: false,
  };
  const data = await useSimpleAPIFetch(
    `/tc/set_cfgs_off`,
    { method: "post", body },
    "Failed to Generate Procedure",
    wamp_topic
  );
  if (data) {
    generated_test_procedure.value = data;
    testProcedure.value = data["file_content"];
    showTestProcedure.value = data["file_content"].length > 0;
    showExecuteTestProcedureButton.value = data["file_content"].length > 0;
  }
};

const gnerateTestProcedureToTurnOn = async () => {
  console.log(cfgNumberString.value);
  if (cfgNumberString.value.length == 0) {
    alert("Please Enter Config Numbers...");
    return;
  }
  testProcedure.value = "";
  showTestProcedure.value = false;
  showExecuteTestProcedureButton.value = false;
  showTestProcedure.value = false;
  let body = {
    configs_str: cfgNumberString.value,
    request_is_to_turn_on: true,
    boa_column_name: selected_boa.value?.name,
    parameter: selected_test_parameter.value?.name,
  };
  const data = await useSimpleAPIFetch(
    `/tc/set_cfgs_on`,
    { method: "post", body },
    "Failed to Generate Procedure",
    wamp_topic
  );
  if (data) {
    generated_test_procedure.value = data;
    testProcedure.value = data["file_content"];
    showTestProcedure.value = data["file_content"].length > 0;
    showExecuteTestProcedureButton.value = data["file_content"].length > 0;
  }
};

const gnerateTestProcedureToConfigurSwitches = async () => {
  testProcedure.value = "";
  showTestProcedure.value = false;
  showExecuteTestProcedureButton.value = false;
  showTestProcedure.value = false;
  let body = {
    configs_str: cfgNumberString.value,
    request_is_to_turn_on: false,
    change_only_switches: true,
  };
  const data = await useSimpleAPIFetch(
    `/tc/set_cfgs_on`,
    { method: "post", body },
    "Failed to Generate Procedure",
    wamp_topic
  );
  if (data) {
    generated_test_procedure.value = data;
    testProcedure.value = data["file_content"];
    showTestProcedure.value = data["file_content"].length > 0;
    showExecuteTestProcedureButton.value = data["file_content"].length > 0;
  }
};

const gnerateTestProcedureToTurnOffAllPayload = async () => {
  testProcedure.value = "";
  showTestProcedure.value = false;
  showExecuteTestProcedureButton.value = false;
  showTestProcedure.value = false;
  let body = {};
  const data = await useSimpleAPIFetch(
    `/tc/set_all_payload_off`,
    { method: "post", body },
    "Failed to Generate Procedure",
    wamp_topic
  );
  if (data) {
    generated_test_procedure.value = data;
    testProcedure.value = data["file_content"];
    showTestProcedure.value = data["file_content"].length > 0;
    showExecuteTestProcedureButton.value = data["file_content"].length > 0;
  }
};

const executeTestProcedure = async () => {
  const data = await useSimpleAPIFetch(
    `/tc/executeTestProcedure`,
    {
      method: "post",
      body: generated_test_procedure.value,
    },
    "Failed to Execute Procedure",
    wamp_topic
  );
  if (data && data.length > 0) {
    showExecuteTestProcedureButton.value = false;
  }
};
</script>

<template>
  <div class="content">
    <AppName appname="Telecommand"></AppName>
    <div class="grid grid-cols-4 gap-4 pt-4">
      <div>
        <div class="">
          <label for="cfgNumberString">
            <h3>Enter Config Numbers</h3>
          </label>
          <InputText id="cfgNumberString" v-model="cfgNumberString" aria-describedby="cfgNumberString-help"
            class="w-full" />
        </div>

        <div class="">
          <label for="cfgNumberString">
            <h3>Test Parameter</h3>
          </label>
          <Select v-model="selected_test_parameter" :autoFilterFocus="true" :options="test_parameter_names" filter
            showClear optionLabel="name" placeholder="Select Test Parameter" class="w-full" />
        </div>
        <div class="">
          <label for="cfgNumberString">
            <h3>BOA</h3>
          </label>
          <Select v-model="selected_boa" :autoFilterFocus="true" :options="boa" filter showClear optionLabel="name"
            placeholder="Select Boa" class="w-full" />
        </div>

        <div>
          <div class="grid grid-cols-2 gap-2 mt-4">
            <div class="">
              <Button class="w-full" @click="gnerateTestProcedureToTurnOn" label="Turn on" severity="warning" raised />
            </div>

            <div class="">
              <Button class="w-full" @click="gnerateTestProcedureToTurnOff" label="Turn off" severity="info" raised />
            </div>
          </div>
        </div>

        <div>
          <div class="grid grid-cols-2 gap-2 mt-4">
            <div class="">
              <Button class="w-full" @click="gnerateTestProcedureToConfigurSwitches" label="Configure Switches"
                severity="info" raised />
            </div>

            <div class="">
              <Button class="w-full" @click="gnerateTestProcedureToTurnOffAllPayload" label="All Payload Off"
                severity="info" raised />
            </div>
          </div>
        </div>

        <div>

          <div class="mt-4">

            <Button class="w-full" v-if="showExecuteTestProcedureButton" label="Execute Test Procedure"
              @click="executeTestProcedure" severity="warn" raised />
          </div>
        </div>

      </div>
      <div class="col-2 gap-3">
        <tc-test-procedure-display :rows="11" :testProcedure="testProcedure"></tc-test-procedure-display>
      </div>
    </div>
   
    <div class="grid mt-1">
      <div class="col-12">
        <ExecutionStatus :store="statusStore" :height="'150px'"></ExecutionStatus>
      </div>
    </div>
  </div>
</template>
<style lang="scss"></style>
