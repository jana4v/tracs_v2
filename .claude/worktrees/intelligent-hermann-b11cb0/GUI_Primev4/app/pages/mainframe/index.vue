<script setup>
import { ref } from "vue";
import { initMenu,wamp_topic } from "@/composables/mainframe/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts"

definePageMeta({
  title: "Mainframe",
});

initMenu(0)

const testProcedure = ref("");
const executeTestProcedure = () => {
  useAPIFetch(`/tc/executeTestProcedure`, {
    method: "post",
    body: testProcedure.value,
  }).then(async (res) => {
    if (res.error == null && res.data.length > 0) {
     
    } else {
      let msg={
          summary: "Falied to Ececute Procedure",
          status:  `Error:${res.error.data.detail}`,
          progress: "0",
        };
        await publishToWampTopic(msg,wamp_topic)
    }
  });
};
const tab_items = ref([
    { id:0, label: 'Def Limits', icon: 'pi pi-list' },
    { id:1, label: 'Tolerance', icon: 'pi pi-chart-line' },
    { id:2, label: 'Include', icon: 'pi pi-list' },
    { id:3, label: 'Ignore', icon: 'pi pi-inbox' }
]);
const active = ref(0);
const tabChanged = (val) => { active.value =val; testProcedure.value="" }  
</script>

<template>
  <div class="content">
    <AppName appname="Manual TC Console"></AppName>
    <div class="">
        <Tabs >
            <TabList>
                <Tab v-for="tab in tab_items" :key="tab.label"  :value="tab.id" @click="tabChanged(tab.id)">
                  <span>{{ tab.label }}</span>
                </Tab>
            </TabList>
           
        </Tabs>
    </div>
  
     <div v-if="active === 0">
        <h3>Def Limit</h3>
        <Textarea class="preserve-whitespace" spellcheck="false" :modelValue="testProcedure" variant="filled" rows="15" cols="60" />
    </div>

    <div v-if="active === 1" >
        <h3>Tolerance</h3>
        <Textarea class="preserve-whitespace" spellcheck="false" :modelValue="testProcedure" variant="filled" rows="15" cols="60" />
    </div>

    <div v-if="active === 2">
        <h3>Include</h3>
        <Textarea class="preserve-whitespace" spellcheck="false" :modelValue="testProcedure" variant="filled" rows="15" cols="60" />
    </div>
    <div v-if="active === 3">
        <h3>Ignore</h3>
        <Textarea class="preserve-whitespace" spellcheck="false" :modelValue="testProcedure" variant="filled" rows="15" cols="60" />
    </div>
    

    <div class="grid pt-4">
      <div class="col-6 gap-2">
        <Button
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
