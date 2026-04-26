<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/tc/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

const testProcedure = ref("");
const showTestProcedure = ref(false);
const ag_grid_boa_phase = ref(null);
const showExecuteTestProcedureButton = ref(false);
const data_commands = ref([]);
const selected_data_command = ref(null);
const data_cmd_values = ref([]);
const selected_value = ref(null);
const dispaly_value_to_data_code_map = ref({});
const commandsList = ref([]);
const selected_data_command_list = ref([]);
const selectedCommands = ref([]);
const generated_test_procedure = ref({});

const side_nav_config = useState("side_nav_config");
const statusStore = tCstore();
definePageMeta({
  title: "PAPERT Application",
});

initMenu(1);

// Load data commands list
(async () => {
  const data = await useSimpleAPIFetch(
    `/tc/get_data_commands_list`,
    { method: "GET" },
    "Failed to Get Data commands List from Database",
    wamp_topic
  );
  if (data && Array.isArray(data) && data.length > 0) {
    data_commands.value = data.map((opt) => ({ tc: opt }));
  }
})();

const fetchValues = async (newVal) => {
  const data = await useSimpleAPIFetch(
    `/tc/get_data_command_values`,
    {
      method: "POST",
      body: [selected_data_command.value["tc"]],
    },
    "Failed to Get Data command Values from Database",
    wamp_topic
  );
  if (data) {
    data_cmd_values.value = Object.keys(data).map((opt) => ({ tc: opt }));
    dispaly_value_to_data_code_map.value = data;
  }
};

const onValueChanged = (value) => {
  let data = {
    tc: selected_data_command.value["tc"],
    value: value["tc"],
    code: dispaly_value_to_data_code_map.value[value["tc"]],
  };
  data["id"] = data["tc"] + "_" + value["tc"];
  if (selected_data_command_list.value.indexOf(data["id"]) == -1) {
    commandsList.value.push(data);
    selected_data_command_list.value.push(selected_data_command.value["tc"]);
    selected_data_command_list.value.push(data["id"]);
  }
};

const gnerateTestProcedure = async () => {
  showTestProcedure.value = false;
  if (commandsList.value.length == 0) return;
  let cmds = commandsList.value.map(
    (obj) => `${obj.tc} ${obj.code}: ${obj.value}`
  );

  const data = await useSimpleAPIFetch(
    `/tc/generate_data_commands`,
    {
      method: "post",
      body: { commands: cmds },
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

const columns = ref([
  { field: "tc", header: "COMMAND" },
  { field: "code", header: "CODE" },
  { field: "value", header: "VALUE" },
]);
const onRowReorder = (event) => {
  commandsList.value = event.value;
};
const deleteAll = () => {
  commandsList.value = [];
  selected_data_command_list.value = [];
  selected_data_command.value = "";
  selected_value.value = "";
  selectedCommands.value = [];
};
const deleteSelected = () => {
  commandsList.value = commandsList.value.filter(
    (item) =>
      !selectedCommands.value.some(
        (selectedItem) => selectedItem.id === item.id
      )
  );
  if (commandsList.value.length == 0) {
    selected_data_command.value = "";
    selected_value.value = "";
    selected_data_command_list.value = [];
    selectedCommands.value = [];
  }
};
</script>

<template>
  <div>
    <AppName appname="Telecommand"></AppName>
    <div class="grid grid-cols-12 gap-4 pt-4">
      <div class="col-span-5">
        <div class="">
        <div class="">
          <label for="cfgNumberString">
            <h3>Select Command</h3>
          </label>
          <Select
            v-model="selected_data_command"
            @update:model-value="fetchValues"
            :options="data_commands"
            :autoFilterFocus="true"
            filter
            optionLabel="tc"
            placeholder="Select File"
            class="w-full"
          />
        </div>
        <div class="">
          <label for="Value">
            <h3>Select Value</h3>
          </label>
          <Select
            v-model="selected_value"
            :autoFilterFocus="true"
            :options="data_cmd_values"
            @update:model-value="onValueChanged"
            filter
            optionLabel="tc"
            placeholder="Select File"
            class="w-full"
          />
        </div>
        </div>
        <div class="">
        <DataTable
          :value="commandsList"
          @rowReorder="onRowReorder"
          v-model:selection="selectedCommands"
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
          <Column
            rowReorder
            headerStyle="width: 3rem"
            :reorderableColumn="false"
          />
          <Column
            v-for="col of columns"
            :field="col.field"
            :header="col.header"
            :key="col.field"
          ></Column>
        </DataTable>
        <div class="grid grid-cols-2 gap-2">
        <div class="mt-4">
          <Button
            class="w-full"
            @click="gnerateTestProcedure"
            label="Generate Procedure"
            severity="info"
            raised
          />
        </div>
        <div class="mt-4">
            <Button
              class="w-full"
              v-if="showExecuteTestProcedureButton"
              label="Execute Test Procedure"
              @click="executeTestProcedure"
              severity="warn"
              raised
            />
          </div>
          </div>
      </div>

      </div>
      <div class="col-span-7">
        <div class="">
          <tc-test-procedure-display :cols="55" :rows="11"
            :testProcedure="testProcedure"
          ></tc-test-procedure-display>
        </div>
      </div>
    </div>
    <div class="grid grid-cols-2">

    </div>

    <div class="grid">
      <div class="col-6">
        <div class="grid pt-5">
          <div v-if="showTestProcedure" class="col-6 gap-2"></div>
        </div>
      </div>
    </div>

    <div class="grid mt-4">
      <div class="col-12">
        <ExecutionStatus :store="statusStore" :height="'150px'"></ExecutionStatus>
      </div>
    </div>
  </div>
</template>
