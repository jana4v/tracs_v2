<script setup>
import { ref, watch } from "vue";

import { initMenu, wamp_topic } from "@/composables/tc/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

const tc_store = tCstore();
const groupNames = ref([]);
const selectedGroups = ref([]);
const commandsList = ref([]);
const selectedCommands = ref([]);
const tableSelectedCommands = ref([]);
const testProcedure = ref("");
const showTestProcedure = ref(false);
const generated_test_procedure = ref({});
const showExecuteTestProcedureButton = ref(false);
const enable_manual_edit = ref(false);

definePageMeta({ title: "PAPERT Application" });
initMenu(2);


if(tc_store.getManualCommands([]).length === 0) {
  // Fetch manual commands if not already loaded
  (async () => {
    const data = await useSimpleAPIFetch(
      `/tc/get_manual_commands`,
      { method: "GET" },
      "Failed to Get Commands list from Database",
      wamp_topic
    );
    if (data) {
      tc_store.loadManualCommands(data);
      commandsList.value = tc_store.getManualCommands(selectedGroups.value.map((tc) => tc.tc.toLowerCase())).map((opt) => ({ tc: opt }));
    }
  })();
} else {
  commandsList.value = tc_store.getManualCommands(selectedGroups.value.map((tc) => tc.tc.toLowerCase())).map((opt) => ({ tc: opt }));
  groupNames.value = tc_store.getGroupNames().map((opt) => ({ tc: opt }));
}



(async () => {
    const data = await useSimpleAPIFetch(
      `/tc/get_manual_cmd_edit_mode`,
      { method: "GET" },
      "Failed to Get manual_cmd_edit_mode from Redis Database",
      wamp_topic
    );
    if (data) {
      enable_manual_edit.value = data;
    }
  })();


// Fetch commands whenever selectedGroups changes
const fetchCommands = () => {
  if (selectedGroups.value.length == 0) {
    commandsList.value = tc_store.getManualCommands([]).map((opt) => ({ tc: opt }));
    return;
  }
  commandsList.value = tc_store.getManualCommands(selectedGroups.value.map((tc) => tc.tc.toLowerCase())).map((opt) => ({ tc: opt }));
};
watch(selectedGroups, fetchCommands, { immediate: true });

const gnerateTestProcedure = async () => {
  showTestProcedure.value = false;
  const data = await useSimpleAPIFetch(
    `/tc/generate_manual_commands_file`,
    {
      method: "post",
      body: { commands: selectedCommands.value.map((tc) => tc.tc) },
    },
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

function getCustomFileName() {
  const now = new Date();

  // Get parts with leading zeros as needed
  const pad = n => String(n).padStart(2, '0');

  const hours = pad(now.getHours());
  const minutes = pad(now.getMinutes());
  const seconds = pad(now.getSeconds());
  const day = pad(now.getDate());

  // Months as 3-letter lowercase
  const months = ["jan", "feb", "mar", "apr", "may", "jun", 
                  "jul", "aug", "sep", "oct", "nov", "dec"];
  const month = months[now.getMonth()];
  const year = String(now.getFullYear()).slice(-2);

  return `${hours}_${minutes}_${seconds}_${day}${month}${year}.tst`;
}
const executeTestProcedure = async () => {
  if (!generated_test_procedure.value.file_name) {
    generated_test_procedure.value.file_name = getCustomFileName();
  }
  generated_test_procedure.value = {
    file_content: testProcedure.value,
    file_name: generated_test_procedure.value.file_name,
  };
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
    generated_test_procedure.value.file_name = null;
  } else {
    generated_test_procedure.value.file_name = null;
  }
};

const columns = ref([{ field: "tc", header: "COMMAND" }]);
const onRowReorder = (event) => {
  selectedCommands.value = event.value;
};
const deleteAll = () => {
  selectedCommands.value = [];
  tableSelectedCommands.value = [];
};
const deleteSelected = () => {
  selectedCommands.value = selectedCommands.value.filter(
    (item) => !tableSelectedCommands.value.some((selectedItem) => selectedItem.id === item.id)
  );
  tableSelectedCommands.value = [];
};
</script>

<template>
  <AppName appname="Telecommand" />
  <div class="grid grid-cols-12 gap-4 pt-4">
    <!-- LEFT COLUMN: selections + table + buttons -->
    <div class="col-span-5">
      <!-- Group + Command selects stacked -->
      <div>
        <div class="mb-4">
          <h3>Select Groups</h3>
          <MultiSelect
            v-model="selectedGroups"
            :options="groupNames"
            filter
            optionLabel="tc"
            placeholder="Select Groups"
            display="chip"
            class="w-full"
            :maxSelectedLabels="15"
          />
        </div>
        <div class="mb-4">
          <h3>Select Commands</h3>
          <MultiSelect
            v-model="selectedCommands"
            :options="commandsList"
            filter
            optionLabel="tc"
            placeholder="Select Commands"
            display="chip"
            class="w-full"
            :maxSelectedLabels="15"
            :showToggleAll="true"
            :selectionLimit="15"
          />
        </div>
      </div>

      <!-- Table for selected commands -->
      <DataTable
        :value="selectedCommands"
        @rowReorder="onRowReorder"
        v-model:selection="tableSelectedCommands"
        resizableColumns
        columnResizeMode="fit"
      >
        <template #header>
          <div class="flex gap-4">
            <Button
              type="button"
              icon="pi pi-trash"
              label="Delete Selected"
              outlined
              @click="deleteSelected"
            />
            <Button
              type="button"
              icon="pi pi-trash"
              label="Clear All"
              outlined
              @click="deleteAll"
            />
          </div>
        </template>
        <Column selectionMode="multiple" headerStyle="width: 3rem"></Column>
        <Column rowReorder headerStyle="width: 3rem" :reorderableColumn="false" />
        <Column v-for="col of columns" :field="col.field" :header="col.header" :key="col.field"></Column>
      </DataTable>
      <div class="flex gap-2 pt-4">
        <div class="mt-2">
          <Button
            @click="gnerateTestProcedure"
            label="Generate Procedure"
            severity="info"
            raised
          />
        </div>
        <div class="mt-2">
          <Button
            v-if="showExecuteTestProcedureButton || true"
            label="Execute Test Procedure"
            @click="executeTestProcedure"
            severity="warn"
            raised
          />
        </div>
      </div>
    </div>
    <!-- RIGHT COLUMN: procedure display -->
    <div class="col-span-7">
      <tc-test-procedure-display
        :rows="12"
        :cols="55"
        :testProcedure="testProcedure"
        v-model:testProcedure="testProcedure"
        :readonly="!enable_manual_edit"
      />
    </div>
  </div>

  <div class="grid mt-4">
    <div class="col-12">
      <ExecutionStatus :store="tc_store" :height="'150px'" />
    </div>
  </div>
</template>
