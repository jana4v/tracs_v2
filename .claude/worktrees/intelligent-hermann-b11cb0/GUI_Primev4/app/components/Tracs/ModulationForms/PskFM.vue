<template>
    <div class="grid" style="min-height: 10px; min-width: 100%">
        <div class="col-3">
            <HandsonTableHandSonTbl :data="ports" :hotSettings="port_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
        <div class="col-4">
            <HandsonTableHandSonTbl :data="frequencies" :hotSettings="frequency_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
        <div class="col-5">
            <HandsonTableHandSonTbl :data="env_data" :hotSettings="env_data_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
    </div>
</template>
  
<script lang="ts" setup>

type DataPropType = {
    ports: string[][];
    env_data: string[][];
    frequencies: string[][];
};

const isEditable = inject<Ref<boolean>>("isEditable", ref(false));
const isReadOnly = ref(true);
const counter = ref(0);

const change_editing_mode = ()=>{
    isReadOnly.value = (isEditable.value) ? false : true;
    port_table_hotSettings.value.readOnly = isReadOnly.value;
    env_data_table_hotSettings.value.readOnly = isReadOnly.value;
    frequency_table_hotSettings.value.readOnly = isReadOnly.value;
    counter.value += 1;
}

onMounted(async () => {
    change_editing_mode();
});

watch(isEditable, (n) => {
    change_editing_mode();

})



//watch(()=>{})

const props = defineProps({
    data: {
        type: Object as () => DataPropType,
        default: () => ({
            ports: [["EV"], ["AEV"], ["GLOBAL"]],
            env_data: [["MAX_INPUT_POWER(dBm)", -60], ["FM_DEVIATION(kHz)", "400"]],
            frequencies: [["DF", ""]],
        }),
    },
});
// if(props.data.ports == undefined) props.data.ports = [["EV"], ["AEV"], ["GLOBAL"]];
// if(props.data.env_data == undefined) props.data.env_data = [["MAX_INPUT_POWER(dBm)","-60"], ["FM DEVIATION(kHz)","400"]];
// if(props.data.frequencies == undefined) props.data.frequencies = [["DF",""]];

const ports = ref(props.data.ports);

let port_table_hotSettings = ref({ height: 150, width: '100%', stretchH: 'all', colHeaders: ["PORTS"], readOnly: true });

const env_data = ref(props.data.env_data);
let env_data_table_hotSettings = ref({ contextMenu: false, height: 150, width: '100%', stretchH: 'all', colHeaders: ["Parameter", "Value"], readOnly: true ,columns: [{ data: 0, readOnly: true },{ data: 1}]});

const frequencies = ref(props.data.frequencies);
let frequency_table_hotSettings = ref({ height: 150, width: '100%', stretchH: 'all', colHeaders: ["Frequency Label", "Frequency(MHz)"], readOnly: true });

const get_data = () => {
    return {
        ports: ports.value,
        env_data: env_data.value,
        frequencies: frequencies.value
    }
}
defineExpose({
    get_data
})

</script>