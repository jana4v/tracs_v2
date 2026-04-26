<template>
     <AppName appname="TTC RF Automated Checkout Software"></AppName>
     <div class="">
      <Splitter class="h-[80vh] w-full" >
          <SplitterPanel class="" :size="70" :minSize="50"> 
            
            <tracs-test-selection></tracs-test-selection>
        </SplitterPanel>
          <SplitterPanel class="" :size="30" :minSize="30"> Panel 2 </SplitterPanel>
      </Splitter>
  </div>
    <!-- <div class="grid">
        <div class="col-8" style="max-height: 85vh;">
          
        </div>
        <div class="col-4" style="max-height: 85vh;">
           <tracs-test-execution-console style="min-height: 85vh;"></tracs-test-execution-console> 
        </div>
    </div> -->


</template>
<script setup>
import { ref } from "vue";
import { initMenu,wamp_topic } from "@/composables/tracs/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts"

definePageMeta({
  title: "Telecommand",
});

initMenu(0)

</script>




