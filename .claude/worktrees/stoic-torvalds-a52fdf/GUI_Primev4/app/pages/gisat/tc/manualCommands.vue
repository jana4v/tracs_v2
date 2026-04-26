<script setup>

//import {useAPIFetch} from '~/composables/useAPIFetch'
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/gisat/tc_sidenav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

definePageMeta({
  title: "Telecommand",
});

initMenu(3);


const selectedcommands = ref([]);
const commandsList = ref([]);
const testProcedure = ref("");
const showTestProcedure = ref(false);
const showExecuteTestProcedureButton = ref(false);

useAPIFetch(`tc/getTelecommands`, { method: "GET" }).then(async (res) => {
  if (res.error == null && res.data.length > 0) {
    commandsList.value = res.data.map((opt) => ({ name: opt }));
  } else {
    let msg = {
      summary: "Failed to Get Testcommands List Names from Database",
      status: `Error:${res.error.data.detail}`,
      progress: "0",
    };
    await publishToWampTopic(msg,wamp_topic)
  }
});

const gnerateTestProcedure = () => {
    showTestProcedure.value = false;
    console.log(selectedcommands.value.map(tc => tc.tc));
    useAPIFetch(`/tc/generate_manual_commands_file`, { method: 'post', body: selectedcommands.value.map(tc => tc.tc) }).then((res) => {
        if (res.error == null && res.data.length > 0) {
            testProcedure.value = res.data.replace(/\\n/g, '\n');;
            showTestProcedure.value = true;
            showExecuteTestProcedureButton.value = true;
        }
    });
}

const executeTestProcedure = () => {
    useAPIFetch(`/tc/executeTestProcedure`, { method: 'post', body: { file: testProcedure.value, wait_for_complete: true } }).then((res) => {
        if (res.error == null && res.data.length > 0) {
            showExecuteTestProcedureButton.value = false;
        }
    });
}


onMounted(async () => {
})



</script>


<template>

    <div class="grid pt-4">
        <div class="col-2">
            <h3>Select Commands</h3>
            <MultiSelect v-model="selectedcommands" :options="commandsList" filter optionLabel="tc"
                placeholder="Select Commands" :maxSelectedLabels="3" class="w-full md:w-20rem" />

        </div>


    </div>

    <div class="grid pt-4">
        <div class="col-6 gap-2">
            <Button @click="gnerateTestProcedure" label="Generate Test Procedure" severity="info" raised />
        </div>
    </div>

    <div class="grid pt-4">
        <div class="col-6 gap-2" v-if="showTestProcedure">
            <tc-test-procedure-display :testProcedure="testProcedure"></tc-test-procedure-display>
        </div>
    </div>

    <div class="grid pt-4">
        <div class="col-6 gap-2">
            <Button v-if="showExecuteTestProcedureButton" label="Execute Test Procedure" @click="executeTestProcedure"
                severity="primary" raised />
        </div>
    </div>


    <div class="grid mt-4">
        <div class="col-12">
            <ExecutionStatus :topic="wamp_topic"></ExecutionStatus>
        </div>
    </div>
</template>
