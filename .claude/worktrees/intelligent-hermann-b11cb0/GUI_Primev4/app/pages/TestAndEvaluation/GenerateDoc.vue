<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/TandE/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
//import { useDownloadBase64File } from "@/composables/useDownloadBase64File.ts";
definePageMeta({
    title: "Test And Evaluation",
});
initMenu(2);
const excelOptions = ref([]);
const selectedExcels = ref([]);
const docx =  ref();
const docxUrl= "http://" + window.location.hostname + "/file_n/report.docx";
rpc("com.te.get_excel_names").then(async (res) => {
    excelOptions.value = res.map((opt) => ({ v: opt }));
    selectedExcels.value = toRaw(excelOptions.value);
});

const generate_document = () => {
    rpc("com.te.generate_doc", [{
        type: "generate_document",
        selected_excels: selectedExcels.value.map(opt=>opt.v),
    }]).then(
        async (res) => {
            if (res?.error != null) {
                let msg = {
                    summary: "Failed to Generate Document",
                    status: `Error:${res.error}`,
                    progress: "0",
                };
                wamp_publish(wamp_topic, [], msg);
            } else {
                // console.log(res.data)
                docx.value.click();
            }

        }
    );
};
    


const abort_test = () => {
    rpc("com.te.abort", [{
        type: "abort_test",
        sensor_name: "no sensor",
    }]);
};
const doc = ref(null);
</script>
<template>
<AppName appname="Test And Evaluation"></AppName>
        <div class="flex mt-10 gap-6">

            <div class="w-1/4">
                <label text-cyan-300> Select Excels </label>
                <MultiSelect class="w-full" v-model="selectedExcels" :options="excelOptions" optionLabel="v" filter placeholder="Select Data Excels"
    :maxSelectedLabels="3"  />
            </div>
        </div>

        <div class="flex gap-4 mt-8">
            <div class="w-1/6">
                <Button class="w-full" label="Generate Document" @click="generate_document" severity="primary" raised />
            </div>
            <div class="w-1/6">
                <Button class="w-full" label="Abort" @click="abort_test" severity="warn" raised />
            </div>
        </div>
        <a v-bind:href="docxUrl" ref="docx"></a>


        <div class="flex">
      <div class="w-full">
        <ExecutionStatus :topic="wamp_topic"></ExecutionStatus>
      </div>
    </div>
    <div class="for_doc_down_load" ref="doc"></div>
</template>