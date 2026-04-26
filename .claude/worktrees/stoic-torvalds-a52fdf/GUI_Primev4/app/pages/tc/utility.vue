<script setup>
import { initMenu, wamp_topic } from "@/composables/tc/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
const statusStore = tCstore();
definePageMeta({
  title: "Telecommand",
});

initMenu(4)
const cfgNumberString = ref("");
const testProcedure = ref("");
const showTestProcedure = ref(false);
const showExecuteTestProcedureButton = ref(false);
const test_parameter_names = ref([]);
const selected_test_parameter = ref("");
const boa = ref([]);
const selected_boa = ref("");
const generated_test_procedure = ref({});

const docx =  ref();
const docxUrl= "http://" + window.location.hostname + "/file_n/procedures.docx";

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

const gnerateTestProcedureDocument = async () => {
  if (cfgNumberString.value.length == 0) {
    alert("Please Enter Config Numbers...");
    return;
  }

  let body = {
    configs_str: cfgNumberString.value,
    request_is_to_turn_on: true,
    boa_column_name: selected_boa.value?.name,
    parameter: selected_test_parameter.value?.name,
  };
  const data = await useSimpleAPIFetch(
    `/tc/generate_test_procedure_document`,
    {
      method: "post",
      body: body,
    },
    "Failed to Generate Test Procedure Document",
    wamp_topic
  );
  if (data && data.length > 0) {
    docx.value.click();
  }
};


const gnerateTestProcedureTurnOffDocument = async () => {
  if (cfgNumberString.value.length == 0) {
    alert("Please Enter Config Numbers...");
    return;
  }

  let body = {
    configs_str: cfgNumberString.value,
    request_is_to_turn_on: true,
    boa_column_name: selected_boa.value?.name,
    parameter: selected_test_parameter.value?.name,
  };
  const data = await useSimpleAPIFetch(
    `/tc/generate_test_procedure_turn_off_document`,
    {
      method: "post",
      body: body,
    },
    "Failed to Generate Test Procedure Document",
    wamp_topic
  );
  if (data && data.length > 0) {
    docx.value.click();
  }
};

</script>

<template>
  <div class="content">
    <AppName appname="Telecommand"></AppName>
  <div class="grid grid-cols-3 gap-4 pt-4">
     <div>
        <label for="cfgNumberString">
          <h3>Enter Config Numbers</h3>
        </label>
        <InputText
          id="cfgNumberString"
          v-model="cfgNumberString"
          aria-describedby="cfgNumberString-help"
          class="w-full"
        />
      </div>

      <div >
        <label for="cfgNumberString">
          <h3>Test Parameter</h3>
        </label>
        <Select
          v-model="selected_test_parameter"
          :autoFilterFocus="true"
          :options="test_parameter_names"
          filter
          showClear
          optionLabel="name"
          placeholder="Select Test Parameter"
          class="w-full"
        />
      </div>
      <div >
        <label for="cfgNumberString">
          <h3>BOA</h3>
        </label>
        <Select
          v-model="selected_boa"
          :autoFilterFocus="true"
          :options="boa"
          filter
          showClear
          optionLabel="name"
          placeholder="Select Boa"
          class="w-full md:w-20rem"
        />
      </div>
    </div>

    <div class="mt-4 flex gap-2">
      <div class="">
        <Button
          @click="gnerateTestProcedureDocument"
          label="Generate Test Procedures Document"
          severity="info"
          raised
        />
      </div>
      <div class="">
        <Button
          @click="gnerateTestProcedureTurnOffDocument"
          label="Generate Test Procedures OFF Document"
          severity="info"
          raised
        />
      </div>
    </div>
<a v-bind:href="docxUrl" ref="docx"></a>
    <div class="grid mt-4"> 
      <div class="col-12">
       <ExecutionStatus :store="statusStore" :height="'150px'"></ExecutionStatus>
      </div>
    </div>
  </div>

</template>
<style lang="scss"></style>
