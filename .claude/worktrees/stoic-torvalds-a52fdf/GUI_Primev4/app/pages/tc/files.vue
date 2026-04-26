<script setup>
//import {useAPIFetch} from '~/composables/useAPIFetch'
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/tc/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
const statusStore = tCstore();
initMenu(3);
const selectedFileName = ref("");
const fileNames = ref([]);
const trigger_request = ref({});

// Load TC file names
(async () => {
  const data = await useSimpleAPIFetch(
    `/tc/get_tc_file_names`,
    { method: "GET" },
    "Failed to Get TC File Names from Database",
    wamp_topic
  );
  if (data && Array.isArray(data) && data.length > 0) {
    fileNames.value = data.map((opt) => ({ name: opt }));
  }
})();

const executeTestProcedure = async () => {
  const data = await useSimpleAPIFetch(
    `/tc/trigger_file_execution`,
    {
      method: "post",
      body: { file_name: selectedFileName.value.name, file_content: "" },
    },
    "Failed to Trigger File",
    wamp_topic
  );
  if (data && data.length > 0) {
    selectedFileName.value = "";
  }
};
</script>

<template>
   <AppName appname="Telecommand"></AppName>
  <div class="grid pt-4">
    <div class="col-2">
      <h3>Select File</h3>
      <Select
        v-model="selectedFileName"
        :autoFilterFocus="true"
        :options="fileNames"
        filter
        showClear
        optionLabel="name"
        placeholder="Select File"
        class="w-full md:w-20rem"
      />
    </div>
  </div>

  <div class="grid pt-4" v-if="selectedFileName">
    <div class="col-6 gap-2">
      <Button
        @click="executeTestProcedure"
        label="Trigger Execution"
        severity="info"
        raised
      />
    </div>
  </div>

  <div class="grid mt-4">
    <div class="col-12">
     <ExecutionStatus :store="statusStore" :height="'150px'"></ExecutionStatus>
    </div>
  </div>
</template>
