<template>
    <div class="mt-50" ref="container" style="height: 400px; width: 1000px;"></div>
    <Button @click="get_data">GET Data</Button>
</template>

<script setup>
import { onMounted, onBeforeUnmount, ref, toRaw } from "vue";
import {
    createUniver,
    defaultTheme,
    LocaleType,
    merge,
} from "@univerjs/presets";
import { UniverSheetsCorePreset } from "@univerjs/presets/preset-sheets-core";
import UniverPresetSheetsCoreEnUS from "@univerjs/presets/preset-sheets-core/locales/en-US";
import "@univerjs/presets/lib/styles/preset-sheets-core.css";

const container = ref(null);
const univerAPIRef = ref(null);

onMounted(() => {
    const { univerAPI } = createUniver({
        locale: LocaleType.EN_US,
        locales: {
            [LocaleType.EN_US]: merge({}, UniverPresetSheetsCoreEnUS),
        },
        theme: defaultTheme,
        presets: [
            UniverSheetsCorePreset({
                container: container.value,
            }),
        ],
    });

    univerAPI.createWorkbook({
        id: "gyI0JO",
        sheetOrder: ["RSfWjJFv4opmE1JaiRj80"],
        name: "",
        appVersion: "0.5.0",
        locale: "enUS",
        styles: {},
        sheets: {
            RSfWjJFv4opmE1JaiRj80: {
                id: "RSfWjJFv4opmE1JaiRj80",
                name: "Test",
                rowCount: 10,
                columnCount: 5,
                zoomRatio: 1,
                showGridlines: 1,
                rowHeader: { width: 46, hidden: 0 },
                columnHeader: { height: 20, hidden: 0 },
            },
        },
        resources: [
            {
                name: "SHEET_DEFINED_NAME_PLUGIN",
                data: "",
            },
        ],
    });

    univerAPIRef.value = univerAPI;

    // 🔹 Remove "Insert Row", "Insert Column", and "Delete" from the context menu
    setTimeout(() => {
        removeContextMenuItems(univerAPI);
    }, 1000); // Ensure UniverSheet is initialized first
});

const removeContextMenuItems = (univerAPI) => {
    const pluginManager = univerAPI.getPluginManager();
    const contextMenuService = pluginManager.getPluginByName(
        "ui-plugin-sheets"
    )?.contextMenuService;

    if (contextMenuService) {
        // 🔹 Get default context menu items
        const defaultMenuItems = contextMenuService.getContextMenuItems();

        // 🔹 Remove specific menu items
        const filteredMenuItems = defaultMenuItems.filter(
            (item) =>
                item.name !== "insertRow" &&
                item.name !== "insertColumn" &&
                item.name !== "deleteRow" &&
                item.name !== "deleteColumn"
        );

        // 🔹 Apply new context menu settings
        contextMenuService.setContextMenuItems(filteredMenuItems);
    }
};

const get_data = () => {
    const fWorkbook = univerAPIRef.value.getActiveWorkbook();
    const fWorksheet = fWorkbook.getActiveSheet();
    const sheetSnapshot = fWorksheet.getSheet().getSnapshot();
    console.log(sheetSnapshot);
};

onBeforeUnmount(() => {
    toRaw(univerAPIRef.value)?.dispose();
    univerAPIRef.value = null;
});
</script>
