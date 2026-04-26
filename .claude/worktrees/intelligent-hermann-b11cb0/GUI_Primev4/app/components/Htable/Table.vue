<template>
    <div :class="[isDarkMode ? 'ht-theme-main-dark' : 'ht-theme-main', 'mt-5']" style="width: 600px">
        <div v-if="show_banner">
            <Message :severity="banner_severity">{{ banner_msg }}</Message>
        </div>
        <hot-table :key="counter" ref="hotTableRef" :settings="hotTblSettings" :data="tableData"
            licenseKey="non-commercial-and-evaluation"></hot-table>
    </div>
</template>
<script setup lang="ts">
import { ref } from "vue";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
import { HotTable } from "@handsontable/vue3";
import { registerAllModules } from "handsontable/registry";
import "handsontable/styles/handsontable.min.css";
import "handsontable/styles/ht-theme-main.min.css";
import "handsontable/styles/ht-theme-horizon.min.css";
import { useColorModeStore } from "~/stores/colorMode";
registerAllModules();
const hotTableRef = ref(null);
const colorModeStore = useColorModeStore();
const counter = ref(0);
const isDarkMode = ref(colorModeStore.currentMode === "dark");
watch(colorModeStore, () => {
    isDarkMode.value = colorModeStore.currentMode === "dark";
    counter.value++;
});
const banner_severity = ref("success");
const banner_msg = ref("");
const show_banner = ref(false);
const BANNER_TIMEOUT_ERROR = 5000; // 5 seconds for error messages
const BANNER_TIMEOUT_SUCCESS = 3000; // 3 seconds for success messages

const props = defineProps({
    data: {
        type: Array,
        default: () => []
    },
    hotSettings: {
        type: Object,
        default: () => ({})
    },
    get_url: {
        type: String,
        default: ""
    },
    save_url: {
        type: String,
        default: ""
    },
    update_url: {
        type: String,
        default: ""
    },
    delete_url: {
        type: String,
        default: ""
    },
    db_name: {
        type: String,
        default: ""
    },
    collection_name: {
        type: String,
        default: ""
    },
    document_key: {
        type: String,
        default: ""
    },
    enable_column_auto_size: {
        type: Boolean,
        default: true
    },
    generic_backend: {
        type: Boolean,
        default: false
    },
    use_local_data: {
        type: Boolean,
        default: false
    },
    print_url: {
        type: String,
        default: ""
    }
});



const tableData = ref(props.data);

const defaultSettings = {
    contextMenu: ["row_above", "row_below", "remove_row"],
    rowHeaders: true,
    width: "100%",
    mergeCells: false,
    colHeaders: ["A"],
    licenseKey: "non-commercial-and-evaluation",
};

const hotTblSettings = ref({ ...defaultSettings, ...props.hotSettings });

const init_table = (settings, data) => {
    if (hotTableRef.value?.hotInstance) {
        if (settings !== null) {
            hotTblSettings.value = ref({ ...defaultSettings, ...settings });
            hotTableRef.value.hotInstance.updateSettings(settings); // Update table settings
        }
        if (data !== null) hotTableRef.value.hotInstance.loadData(data); // Update table data
        if (counter.value > 1000) {
            counter.value = 0;
        } else {
            counter.value++;
        }
    }
};



async function loadTableFromBackend(key:string) {
    try {
        if(key === undefined || key === null){
        key = props.document_key;
    }
    if(key==="") return;
        tableData.value = [];
        let get_url = props.get_url;
        const rpc_args: DbRequestArgs = {
            db_name: props.db_name,
            collection_name: props.collection_name,
            _key: key
        };
        if (props.generic_backend) {
            get_url = wamp_url_db_get_document;
        }
        rpc(get_url, [rpc_args]).then(
            async (res) => {
                if (res?.error == null) {
                    let tbl_data = res.data.data;
                    if (tbl_data.length === 0) {
                        throw new Error("No data received from the backend");
                    }
                    // Save data to the local database if required
                    if (props.use_local_data) {
                        await $dbUtils.add(props.db_name + props.collection_name, tbl_data);
                    }
                    tableData.value = tbl_data;
                    if (hotTableRef.value?.hotInstance) {
                        hotTableRef.value.hotInstance.loadData(tbl_data);
                    }
                    // Auto-size columns and refresh cells if required
                    if (props.enable_column_auto_size) {
                        // setTimeout(() => {
                        //   gridApi.value?.autoSizeAllColumns();
                        //   gridApi.value?.refreshCells();
                        // }, 100); // Small delay to ensure the grid processes the transaction
                    }

                } else {
                    throw new Error(res.error || "Failed to fetch data from the backend");
                }
            }
        );

        // Handle errors in the response
    } catch (error) {
        // Handle any errors during the load process
        console.error("Failed to load table data:", error);
        banner_severity.value = "error";
        banner_msg.value = error.message || "Failed to get data from Database";
        show_banner.value = true;
        setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR);
    }
}

const getAllErrors = () => {
    const hot = hotTableRef.value.hotInstance;
    const errors = [];

    for (let row = 0; row < hot.countRows(); row++) {
        for (let col = 0; col < hot.countCols(); col++) {
            const cellMeta = hot.getCellMeta(row, col);
            if (cellMeta.valid === false) {
                errors.push({ row, col, value: hot.getDataAtCell(row, col) });
            }
        }
    }
    return errors;
};
async function saveTable(key: string) {
    // Validate for errors in the grid
    if(key === undefined || key === null){
        key = props.document_key;
    }
    let errors = getAllErrors();
    if (errors.length) {
        banner_severity.value = "error";
        banner_msg.value = "Please correct errors before saving";
        show_banner.value = true;
        setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR);
        return;
    }

    try {
        const allData = hotTableRef.value.hotInstance.getSourceData();
        let data = {
            _key: key,
            data: allData,
        };
        let save_url = props.save_url;
        const rpc_args: DbRequestArgs = {
            db_name: props.db_name,
            collection_name: props.collection_name,
            _key: props.key,
            document:data
        };
        if (props.generic_backend) {
            save_url = wamp_url_db_create_update_document;
        }

       
        // // Send the data to the backend
        // const res = await useAPIFetch(save_url, { method: "post", body: data });

        // useAPIFetch(save_url, { method: "post", body: data })
        //   .then(async (res: any) => {
        //     if (res.error) {
        //       throw new Error(res.error);
        //     } else {
        //       banner_severity.value = "success";
        //       banner_msg.value = "Database Updated";
        //       show_banner.value = true;
        //       setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS);

        //     }
        //   })
        //   .catch((err) => {
        //     console.error('Unexpected error:', err);
        //   });
        
        rpc(save_url, [rpc_args]).then(
            async (res) => {
                if (res?.error === null) {
                    banner_severity.value = "success";
                    banner_msg.value = "Database Updated";
                    show_banner.value = true;
                    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS);
                } else {
                    throw new Error(res.error || "Failed to save data to the backend");
                }
            }
        );

    } catch (error) {
        // Handle any errors during the save process
        console.error("Failed to save data:", error);
        banner_severity.value = "error";
        banner_msg.value = error.message || "Failed to Save to Database";
        show_banner.value = true;
        setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR);
    }
}

async function deleteTable(key:string) {
    try{
        if(key === undefined || key === null){
        key = props.document_key;
    }

        let delete_url = props.delete_url;
        const rpc_args: DbRequestArgs = {
            db_name: props.db_name,
            collection_name: props.collection_name,
            _key: key,
        };
        if (props.generic_backend) {
            delete_url = wamp_url_db_delete_document;
        }
              
        rpc(delete_url, [rpc_args]).then(
            async (res) => {
                if (res?.error === null) {
                    banner_severity.value = "success";
                    banner_msg.value = "Database Table Deleted";
                    show_banner.value = true;
                    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS);
                } else {
                    throw new Error(res.error || "Failed to save data to the backend");
                }
            }
        );

    } catch (error) {
        // Handle any errors during the save process
        console.error("Failed to save data:", error);
        banner_severity.value = "error";
        banner_msg.value = error.message || "Failed to Save to Database";
        show_banner.value = true;
        setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR);
    }
}
onMounted(() => {
    loadTableFromBackend();
});

async function printTable(printer_name?:string) {
    const allData = hotTableRef.value.hotInstance.getSourceData();
    rpc(props.print_url, [allData,printer_name]).then(
            async (res) => {
                if (res?.error === null) {
                    banner_severity.value = "success";
                    banner_msg.value = "Printed Table";
                    show_banner.value = true;
                    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_SUCCESS);
                } else {
                    banner_severity.value = "error";
                    banner_msg.value = "Failed to Printed Table";
                    show_banner.value = true;
                    setTimeout(() => (show_banner.value = false), BANNER_TIMEOUT_ERROR);
                }
            }
        );
}
// Expose the method to the parent
defineExpose({
    init_table,
    saveTable,
    deleteTable,
    loadTableFromBackend,
    printTable
});
</script>
