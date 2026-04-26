<script setup>
definePageMeta({
    layout: "phase-cal"
})

//import {useAPIFetch} from '~/composables/useAPIFetch'
import { ref } from "vue";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
const { $wamp } = useNuxtApp();

const cfgNumberString = ref("");
const disable_start_button = ref(false);
const disable_stop_button = ref(true);
const disable_pause_button = ref(true);
const pause_resume__button_label = ref("Pause");
const remarks = ref("");
const selectedTestPhase = ref("");
const testPhaseNames = ref([]);
const selectedSubTestPhase = ref("");
const subTestPhaseNames = ref([]);


useAPIFetch(`/tc/getTestPhaseNames`, { method: "GET" }).then(async (res) => {
  if (res.error == null && res.data.length > 0) {
    testPhaseNames.value = res.data.map((opt) => ({ name: opt }));
  } else {
    let msg = {
      summary: "Failed to Get Test Phase Names from Database",
      status: `Failed to Get Test Phase Names from Database Error:${res.error.data.detail}`,
      progress: "0",
    };
    await publishToWampTopic(msg,wamp_topic)
  }
});


useAPIFetch(`/tc/getSubTestPhaseNames`, { method: "GET" }).then(async (res) => {
  if (res.error == null && res.data.length > 0) {
    subTestPhaseNames.value = res.data.map((opt) => ({ name: opt }));
  } else {
    let msg = {
      summary: "Failed to Get Sub Test Phase Names from Database",
      status: `Failed to Get Sub Test Phase Names from Database Error:${res.error.data.detail}`,
      progress: "0",
    };
    await publishToWampTopic(msg,wamp_topic)
  }
});
    




const startPhaseCalTest = () => {

    pause_resume__button_label.value = "Pause";
    test_execution_status_store.set_status([]);
    $wamp.call('phase.cal.start', [cfgNumberString.value,selectedTestPhase.value,selectedSubTestPhase.value,remarks.value ]).then(
    function (result) {
            disable_start_button.value = false;
            disable_stop_button.value = true;
            disable_pause_button.value = true;
        },
        function (error) {
            alert(`Failed to start phase cal test..${error.error}`);
            disable_start_button.value = false;
            disable_stop_button.value = true;
            disable_pause_button.value = true;
        }
    );

    disable_start_button.value = true;
    disable_stop_button.value = false;
    disable_pause_button.value = false;

}

const pausePhaseCalTest = () => {

    let rpc = 'phase.cal.continue';
    if (pause_resume__button_label.value == "Pause")
    {
        rpc = 'phase.cal.pause';
    }
    $wamp.call(rpc).then(
        function (result) {
            if (pause_resume__button_label.value == "Pause") {
                pause_resume__button_label.value = "Resume";
            } else {
                pause_resume__button_label.value = "Pause";
            }
            disable_start_button.value = true;
            disable_stop_button.value = false;
            disable_pause_button.value = false;
        },
        function (error) {
            alert(`Failed to Pause phase cal test..${error.error}`);
            console.log(error);
            if (pause_resume__button_label.value == "Pause") {
                pause_resume__button_label.value = "Pause";
            } else {
                pause_resume__button_label.value = "Resume";
            }
            console.log('RPC Call failed:', error);
            disable_start_button.value = true;
            disable_stop_button.value = false;
            disable_pause_button.value = false;
        }
    );

}

const abortPhaseCalTest = () => {

    $wamp.call('phase.cal.abort').then(
        function (result) {
            disable_start_button.value = false;
            disable_stop_button.value = true;
            disable_pause_button.value = true;
        },
        function (error) {
            alert(`Failed to abort phase cal test..${error.error}`);
            console.log('RPC Call failed:', error);
            disable_start_button.value = true;
            disable_stop_button.value = false;
            disable_pause_button.value = false;
        }
    );

}


onMounted(async () => {
}) 



</script>



<template>


    <div class="grid pt-4">
        
        <div class="flex flex-column gap-2">
            <label for="cfgNumberString">
                <h4>Enter Config Numbers</h4>
            </label>
            <InputText id="cfgNumberString" v-model="cfgNumberString" aria-describedby="cfgNumberString-help" />
        </div>
        
        <div class="pt-3">
            <h4>Select Test Phase</h4>
            <Dropdown v-model="selectedTestPhase" :options="testPhaseNames" filter optionLabel="name"
                placeholder="Select Test Phase" class="w-full" />
        </div>

        <div class="pt-3">
            <h4>Select Sub Test Phase</h4>
            <Dropdown v-model="selectedSubTestPhase" :options="subTestPhaseNames" filter optionLabel="name"
                placeholder="Select Sub Test Phase" class="w-full" />
        </div>
    </div>

    <div class="grid pt-8">
        <div class="flex flex-column gap-2">
            <label for="cfgNumberString">
                <h5>Remarks</h5>
            </label>
            <InputText id="remarks" v-model="remarks" aria-describedby="remarks" />
        </div>

    </div>

    <div class="grid mt-4">
        <Button class="p-button-lg mr-2" @click="startPhaseCalTest" :disabled="disable_start_button"
            label="Start Phase Calibration Test" severity="success" />
        <Button class="p-button-lg mr-2" @click="pausePhaseCalTest" :disabled="disable_pause_button"
            :label="pause_resume__button_label" severity="info" />
        <Button class="p-button-lg mr-2" @click="abortPhaseCalTest" :disabled="disable_stop_button" label="Abort"
            severity="warning" />
    </div>

    <div class="grid mt-1">
        <div class=""">
            <ExecutionStatus topic="com.gisat.phase.cal.status"></ExecutionStatus>
        </div>
    </div>
</template>
