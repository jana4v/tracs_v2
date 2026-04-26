
<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/gisat/tc_sidenav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

definePageMeta({
  title: "Telecommand",
});

initMenu(2);


const selectedFileName = ref("");
const fileNames = ref([]);


const { data:files } = await useAPIFetch(`tc/getTcFileNames`);
try {
    fileNames.value = files.value.map((opt) => ({ name: opt }));    
} catch (error) {
    
}



const executeTestProcedure = () => {
     useAPIFetch(`/tc/triggerFileExecution`,{ method: 'post', body: {file :selectedFileName.value.name, wait_for_complete: true } }).then((res) => {
        if (res.error.value == null && res.data.value.length > 0) {
            selectedFileName.value = "";
        }
    });
}


</script>


<template>
    
    <div class="grid pt-4">
        <div class="col-2">
           <h3>Select File</h3>
            <Dropdown v-model="selectedFileName" :options="fileNames" filter optionLabel="name" placeholder="Select File"
    class="w-full md:w-20rem" />
        </div>
    </div>

    <div class="grid pt-4" v-if="selectedFileName">
        <div class="col-6 gap-2">
            <Button  @click="executeTestProcedure" label="Trigger Execution" severity="info" raised />
        </div>
    </div>

    
    <div class="grid mt-4">
        <div class="col-12">
            <ExecutionStatus :topic="wamp_topic"></ExecutionStatus>
        </div>
    </div>
</template>


