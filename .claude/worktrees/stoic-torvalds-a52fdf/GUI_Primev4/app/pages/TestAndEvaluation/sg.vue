<script setup>
import { ref } from "vue";
import { initMenu, wamp_topic } from "@/composables/TandE/SideNav.ts";
import { publishToWampTopic } from "@/composables/publishToWamp.ts";
definePageMeta({
    title: "Test And Evaluation",
});
initMenu(1);
const selected_sg_name = ref({ v: "SG 1" });
const sg_options = [
    { v: "SG 1" },
    { v: "SG 2" },
    { v: "CAL SG" },
    { v: "TTC SG" },
];
const sensor_options = [
    { v: "A" },
    { v: "B" },
]
const selected_sensor_channel = ref({ v: "A" });
const test_options = [
    { v: "Power Linearity" },
    { v: "Frequency Linearity" },
];

const selected_sg_model = ref({ v: "E8257D" });
const sg_model_options = [
    { v: "E8257D" },
  ];

const selected_powermeter_model = ref({ v: "ML4238" });
const powermeter_options = [
{ v: "ML4238" },    
{ v: "E4419B" },
{ v: "N1914A" },

];

const powermeter_address = ref("GPIB0:13");
const sg_address = ref("GPIB0:1");
const sg_start_power = ref("-40");
const sg_stop_power = ref("10");
const sg_power_step = ref("1");
const selected_channel_numbers = ref("1-10,12,14-20");
const selected_test = ref({ v: "Power Linearity" });
const show_power_settings = ref(true);
watch(selected_test, (newVal) => {
    //console.log("Selected test changed:", newVal);
    if(newVal.v == "Power Linearity") show_power_settings.value = true;
    else show_power_settings.value = false;
    // Perform any additional actions based on the selected test
});
const freq_steps = ref(30);

const start_test = () => {

    let request_data = {
        type: "start_test",
        name: selected_sg_name.value.v,
        sg_address_v: sg_address.value,
        sg_model_v: selected_sg_model.value.v,
        pm_address_v: powermeter_address.value,
        pm_model_v: selected_powermeter_model.value.v,
        pm_channel: selected_sensor_channel.value.v,
        sg_start_power: sg_start_power.value,
        sg_stop_power: sg_stop_power.value,
        sg_power_step: sg_power_step.value,
        channel_numbers: selected_channel_numbers.value,
        test: selected_test.value.v,
        sg_freq_number_of_steps: freq_steps.value
    };

    rpc("com.te.sg_test", [request_data]).then(
        async (res) => {
            if (res?.error != null) {
                let msg = {
                    summary: "Failed to Start Power Meter TE",
                    status: `Error:${res.error}`,
                    progress: "0",
                };
                wamp_publish(wamp_topic, [], msg);
            }
        }
    );
 
};


    

const abort_test = () => {
    rpc(wamp_topic, [{
        type: "abort_test",
        sensor_name: selected_sensor_name.value.v,
    }]);
};

</script>
<template>
    <div class="content">
        <AppName appname="Test And Evaluation"></AppName>
        <div class="flex mt-10 gap-6">

            <div class="w-1/4">
                <label text-cyan-300> Select Signal Generator </label>
                <Select class="w-full" v-model="selected_sg_name" :options="sg_options" optionLabel="v"
                    placeholder="Select SG" />
            </div>

            <div class="w-1/4">
                <label text-cyan-300> Siganl Generator IP/GPIB Address </label>
                <InputText class="w-full" v-model="sg_address" />
            </div>
            <div class="w-1/6">
                <label text-cyan-300>
                    SG Model
                </label>
                <Select class="mt-0 w-full" v-model="selected_sg_model" :options="sg_model_options" optionLabel="v"
                    placeholder="Select SG Model" />
            </div>
            <div class="w-1/4">
                <label text-cyan-300>
                    Power Meter IP/GPIB Address
                </label>
                <InputText class="w-full" v-model="powermeter_address"/>
            </div>
            <div class="w-1/6">
                <label text-cyan-300>
                    PM Model
                </label>
                <Select class="mt-0 w-full" v-model="selected_powermeter_model" :options="powermeter_options" optionLabel="v"
                    placeholder="Select PM Model" />
            </div>
            <div class="w-1/4">
                <label text-cyan-300>
                    Select Sensor
                </label>

                <Select class="w-full" v-model="selected_sensor_channel" :options="sensor_options" optionLabel="v"
                    placeholder="Select Sensor" />
            </div>
        </div>

        
        <div class="flex gap-8 mt-10">
            <div class="w-1/4">
                <label text-cyan-300> Select Test  </label>
                
                <Select class="mt-0 w-full" v-model="selected_test"  :options="test_options" optionLabel="v"
                    placeholder="Select Test" />
            </div>
        </div>

        <div class="flex gap-4 mt-4" v-if="show_power_settings">
            <Card class="w-3/4 mt-4 border-2 border-2 border-cyan-200 border-solid p-4  text-black rounded rounded-lg p-4" overflow: hidden>

                <template #title>
                    <div class="mb-2 text-cyan">SG Power</div>
                </template>
                <template #content>
                    <div class="flex gap-4 justify-between">
                    <div class="">
                        <label text-cyan-300>
                            Start 
                        </label>
                        <InputNumber class="w-full" showButtons buttonLayout="horizontal" v-model="sg_start_power"
                            :min="-50" :max="25" :step="1" mode="decimal" />
                    </div>

                    <div class="">
                        <label text-cyan-300>
                            Stop 
                        </label>
                        <InputNumber class="w-full" showButtons buttonLayout="horizontal" v-model="sg_stop_power"
                            :min="-50" :max="25" :step="1" mode="decimal" />
                    </div>

                    <div class="">
                        <label text-cyan-300>
                             Step
                        </label>
                        <InputNumber class="w-full" showButtons buttonLayout="horizontal" v-model="sg_power_step"
                            :min="0.5" :max="5" :step="0.5" />
                    </div>
                </div>
                </template>
                <template #footer>

                </template>
            </Card>

        </div>

        <div class="flex gap-4 mt-4" v-else>
            <Card
                class="w-3/4 mt-4 border-2 border-2 border-cyan-200 border-solid p-4  text-black rounded rounded-lg p-4"
                overflow: hidden>

                <template #title>
                    <div class="mb-2 text-cyan">SG Frequency</div>
                </template>
                <template #content>
                    <div class="flex gap-4 justify-between">
                        <div class="">
                            <label text-cyan-300>
                                Number Of Steps in BW
                            </label>
                            <InputNumber class="w-full" showButtons buttonLayout="horizontal" v-model="freq_steps"
                                :min="1" :max="500" :step="1" />
                        </div>
                    </div>
                </template>
                <template #footer>

                </template>
            </Card>
        </div>


        <div class="flex gap-4 mt-4 mt-8">
            <div class="w-1/4">
                <label text-cyan-300> Enter Channels  </label>
                <InputText class="w-full" v-model="selected_channel_numbers" />
              
            </div>
           
        </div>


        <div class="flex gap-4 mt-8">
            <div class="w-1/6">
                <Button class="w-full" label="Start" @click="start_test" severity="primary" raised />
            </div>
            <div class="w-1/6">
                <Button class="w-full" label="Abort" @click="abort_test" severity="warn" raised />
            </div>
        </div>

        <div class="flex">
      <div class="w-full">
        <ExecutionStatus :topic="wamp_topic"></ExecutionStatus>
      </div>
    </div>


    </div>
</template>
