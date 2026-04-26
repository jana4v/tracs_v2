<script setup lang="ts">
import { ref } from "vue";
import { initMenu } from "@/composables/mainframe/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

definePageMeta({
  title: "Mainframe",
});
initMenu(1);

const wamp_topic = "com.mainframe.utils.print_tm_template";

const hTable = ref(null);
const contextMenu = [
  "row_above",
  "row_below",
  "remove_row",
  "clear_column",
  "mergeCells",
];
const mergeCells = ref({});

const db_name = "mainframe";
const collection_name = "TelemetryPrintTemplates";
const document_key = ref("");
const reloadTable = ref(0);

const mnemonics = ref([]);
rpc("scg.tm.get_tm_mnemonics", []).then(async (res) => {
  if (res?.error == null) {
    if (res?.data.length > 0) {
      mnemonics.value = res?.data;
      console.log(mnemonics.value);
    }
    loading.value = false;
  } else {
    let msg = {
      summary: "Failed to Get Mnemonic Names from Backend",
      status: `Error:${res.error}`,
      progress: "0",
    };
    wamp_publish(wamp_topic, [], msg);
    loading.value = false;
  }
});

// Reactive data
const tableData = ref([[], [], [], []]);
const blankTable = [[], [], [], []];
const ColHeaders = ["MNEMONICS"];
const columns = ref([
  { type: "autocomplete", source: [], allowEmpty: false, width: 500 }, // Use the custom editor for the first column
  {},{},{},{},{}
]);

// Watch for mnemonics update and apply new source to Handsontable column
watch(mnemonics, (newVal) => {
  columns.value[0].source = [...newVal]; // Update dropdown dynamically
});
const hotSettings = {
  contextMenu: contextMenu,
  mergeCells: false,
  colHeaders: ["MNEMONICS", "TM1","TM2","SMON","ADC","DTM"],
  hiddenColumns: {
    columns: [1,2,3,4,5],
    indicators: false,
    copyPasteEnabled: false,
  },
  columns: columns.value,
  readOnly: false,
  autoWrapRow: true,
  autoWrapCol: true,
};

const hotSettings2 = {
  contextMenu: contextMenu,
  mergeCells: false,
  colHeaders: ["MNEMONICS", "TM1","TM2","SMON","ADC","DTM"],
  // hiddenColumns: {
  //   columns: [],
  //   indicators: false,
  // },
  readOnly: false,
  autoWrapRow: true,
  autoWrapCol: true,
};



const tab_items = ref([
  { id: 0, label: "Edit Data", icon: "pi pi-chart-line" },
  { id: 1, label: "Add/Edit Telemetry Template", icon: "pi pi-list" },
  { id: 2, label: "Add/Edit Result Template", icon: "pi pi-list" },
  
]);
const active = ref(0);
const tabChanged = (val) => {
  active.value = val;
};

const templateNames = ref([]);
const templateName = ref("");
const templateNameOptions = ref([]);
const selectedTemplateName = ref({});
const loading = ref(true);

const allTemplateNames = ref([]); // Store the original list


const get_template_names = () =>{
const rpc_args: DbRequestArgs = {
            db_name: db_name,
            collection_name: collection_name,
            query: `FOR doc IN ${collection_name} RETURN doc._key`,
            bindvars: {},
        };
rpc(wamp_url_db_get_documentsWithQuery, [rpc_args]).then(async (res) => {
  if (res?.error == null) {
    if (res?.data.length > 0) {
      allTemplateNames.value = res?.data;
      templateNameOptions.value = res?.data.map((opt) => ({ name: opt }));
      templateNames.value = [...allTemplateNames.value]; // Copy to display list
      templateName.value = res?.data[0];
      selectedTemplateName.value = {name:res?.data[0]};
      loadTemplate();
    }
    loading.value = false;
  } else {
    let msg = {
      summary: "Failed to Get Template Names from Backend",
      status: `Error:${res.error}`,
      progress: "0",
    };
    wamp_publish(wamp_topic, [], msg);
    loading.value = false;
  }
});
}
const search = (event) => {
 const query = event.query?.toLowerCase().trim();
  if (query) {
    templateNames.value = allTemplateNames.value.filter((opt) =>
      opt.toLowerCase().includes(query)
    );
  } else {
    templateNames.value = [...allTemplateNames.value]; // Reset when empty
  }
};
const loadTemplate = () => {
  
  if (templateName.value.length == 0) {
    return;
  }
  if (
    allTemplateNames.value.filter((opt) => opt == templateName.value).length ==  0) {
      document_key.value = "";
      reloadTable.value += 1;
    return;
  }
 
  //hTable.value.loadTableFromBackend(templateName.value);
  document_key.value = templateName.value;
  reloadTable.value += 1;
  

};

const saveTemplate = () => {
  hTable.value.saveTable(templateName.value);
  setTimeout(() => {
    get_template_names();
  }, 100);

};
const deleteTemplate = () =>{
  hTable.value.deleteTable(templateName.value);
  setTimeout(() => {
    get_template_names();
  }, 100);
}

const refreshTable = () => {
  reloadTable.value += 1;
};
const hTable2 = ref(null);
const printTable = () => {
  hTable2.value.printTable(selectedPrinter.value);
};

onMounted(() => {
  get_template_names();
 
})

const selectedPrinter = ref("");
const printerOptions = ref([]);

rpc("scg.mainframe.get_printer_names", []).then(async (res) => {
  if (res?.error == null) {
    if (res?.data.length > 0) {
      printerOptions.value = res?.data.map((opt) => ({ name: opt }));
      selectedPrinter.value = res?.data[0];
    }
  } else {
    let msg = {
      summary: "Failed to Get Printer Names from Backend",
      status: `Error:${res.error}`,
      progress: "0",
    };
    wamp_publish(wamp_topic, [], msg);
  }
});

</script>

<template>
  <div class="content">
    <AppName appname="Result Templates"></AppName>
    <div class="">
      <Tabs>
        <TabList>
          <Tab
            v-for="tab in tab_items"
            :key="tab.label"
            :value="tab.id"
            @click="tabChanged(tab.id)"
          >
            <span>{{ tab.label }}</span>
          </Tab>
        </TabList>
      </Tabs>
    </div>

   

    <div v-if="active === 0">
      <div>
      <Select
      class=" mt-5"
        v-model="selectedTemplateName"
        :options="templateNameOptions"
        optionLabel="name" 
        @update:modelValue="loadTemplate"
        placeholder="Select Template"
      />
      <div
        v-if="templateName.length"
        class="ht-theme-main-dark mt-5"
        style="width: 600px"
      >
      <htable-table ref="hTable2" :key="reloadTable" 
      print_url="scg.mainframe.print_table"
      get_url="scg.mainframe.get_filled_tm_template" :db_name="db_name" :collection_name="collection_name" :hotSettings="hotSettings2" :document_key="document_key" ></htable-table>

        <div class="flex gap-2 pt-4">
          <div>
            <Button
              label="Refresh Data"
              @click="refreshTable"
              severity="primary"
              raised
            />
          </div>
          
        </div>

        <div class="flex gap-2 mt-10">
          <div>
            <Select
      class=""
        v-model="selectedPrinter"
        :options="printerOptions"
        optionLabel="name" 
        placeholder="Select Printer"
      />
          </div>
          <div>
            <Button
              label="Print"
              @click="printTable"
              severity="primary"
              raised
            />
          </div>
        </div>
      </div>
    </div>
    </div>

    <div v-if="active === 1">
      <h3>Add/Edit Template</h3>
      <AutoComplete
        v-model="templateName"
        dropdown
        :suggestions="templateNames"
        @complete="search"
        :loading="loading"
        @update:modelValue="loadTemplate"
      />
      <div
        v-if="templateName.length"
        class="ht-theme-main-dark mt-5"
        style="width: 600px"
      >
      <htable-table ref="hTable" :key="reloadTable" :hotSettings="hotSettings" :data="tableData" :generic_backend="true" :db_name="db_name" :collection_name="collection_name" :document_key="document_key" ></htable-table>
        <div class="flex gap-2 pt-4">
          <div class="">
            <Button
              label="Save Template"
              @click="saveTemplate"
              severity="primary"
              raised
            />
          </div>
          <div class="">
            <Button
              label="Delete Template"
              @click="deleteTemplate"
              severity="warn"
              raised
            />
          </div>
        </div>
      </div>
    </div>


    <div class="grid mt-4">
      <div class="col-12">
       
        <ExecutionStatus :topic="wamp_topic"></ExecutionStatus>
      </div>
    </div>
  </div>
</template>
<style lang="scss"></style>
