<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/gisat/tc_sidenav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";

definePageMeta({
  title: "Telecommand",
});

const cfgNumberString = ref("");
const testProcedure = ref("");
const showTestProcedure = ref(false);
const showExecuteTestProcedureButton = ref(false);
initMenu(0);

const gnerateTestProcedureToTurnOfDt = () => {
  showTestProcedure.value = false;
  let body = {
    cfg_string: cfgNumberString.value,
    turn_on: false,
    phase_cal: false,
    turn_off_data_transmitter: true,
    turn_off_eo_payload: false,
  };
  useAPIFetch(`/tc/set_configuration`, { method: "post", body: body }).then(
    async (res) => {
      if (res.error == null && res.data.length > 0) {
        //console.log(res.data);
        testProcedure.value = res.data.replace(/\\n/g, "\n");
        showTestProcedure.value = true;
        showExecuteTestProcedureButton.value = true;
      } else {
        let msg = {
          summary: "Failed to Generate Test Procedure",
          status: `Error:${res.error.data.detail}`,
          progress: "0",
        };
        await publishToWampTopic(msg, wamp_topic);
      }
    }
  );
};

const gnerateTestProcedure = () => {
  showTestProcedure.value = false;
  let body = {
    cfg_string: cfgNumberString.value,
    turn_on: true,
    phase_cal: false,
    turn_off_data_transmitter: false,
    turn_off_eo_payload: false,
  };
  useAPIFetch(`/tc/set_configuration`, { method: "post", body: body }).then(
   async (res) => {
      if (res.error == null && res.data.length > 0) {
        console.log(res.data);
        testProcedure.value = res.data.replace(/\\n/g, "\n");
        showTestProcedure.value = true;
        showExecuteTestProcedureButton.value = true;
      }else {
        let msg = {
          summary: "Failed to Generate Test Procedure",
          status: `Error:${res.error.data.detail}`,
          progress: "0",
        };
        await publishToWampTopic(msg, wamp_topic);
      }

    }
  );
};

const executeTestProcedure = () => {
  useAPIFetch(`/tc/executeTestProcedure`, {
    method: "post",
    body: { file: testProcedure.value, wait_for_complete: true },
  }).then(async (res) => {
    if (res.error == null && res.data.length > 0) {
      showExecuteTestProcedureButton.value = false;
    }else {
        let msg = {
          summary: "Failed to Execute Test Procedure",
          status: `Error:${res.error.data.detail}`,
          progress: "0",
        };
        await publishToWampTopic(msg, wamp_topic);
      }
  });
};
</script>

<template>
  <div class="content">
    <div class="pt-4">
      <div class="flex flex-col gap-2">
        <label for="cfgNumberString">
          <h3>Enter Config Numbers</h3>
        </label>
        <InputText
          id="cfgNumberString"
          v-model="cfgNumberString"
          aria-describedby="cfgNumberString-help"
          class="w-100"
        />
      </div>
    </div>

    <div class="flex gap-4 pt-4">
      <div class="">
        <Button
          @click="gnerateTestProcedure"
          label="Generate Test Procedure to Turn on DT"
          severity="warning"
          raised
        />
      </div>

      <div class="">
        <Button
          @click="gnerateTestProcedureToTurnOfDt"
          label="Generate Test Procedure to Turn off DT"
          severity="info"
          raised
        />
      </div>
    </div>

    <div class="grid pt-4">
      <div class="gap-2" v-if="showTestProcedure">
        <tc-test-procedure-display
          :testProcedure="testProcedure"
        ></tc-test-procedure-display>
      </div>
    </div>

    <div class="grid pt-4">
      <div class="gap-2">
        <Button
          v-if="showExecuteTestProcedureButton"
          label="Execute Test Procedure"
          @click="executeTestProcedure"
          severity="primary"
          raised
        />
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
