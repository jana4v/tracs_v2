<template>
    <div class="grid" style="min-height: 10px; min-width: 100%">
        <div class="col-3">
            <HandsonTableHandSonTbl :data="ports" :hotSettings="dl_port_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
        <div class="col-3">
            <HandsonTableHandSonTbl :data="sub_carriers" :hotSettings="down_link_sub_carriers_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
        <div class="col-6">
            <HandsonTableHandSonTbl :data="frequencies" :hotSettings="dl_frequency_table_hotSettings" :key="counter">
            </HandsonTableHandSonTbl>
        </div>
        
    </div>
</template>
  
<script lang="ts" setup>

type DataPropType = {
    ports: string[][];
    sub_carriers: number[][];
    frequencies: string[][];
};

const isEditable = inject<Ref<boolean>>("isEditable", ref(false));
const isReadOnly = ref(true);
const counter = ref(0);


const change_editing_mode = ()=>{
    isReadOnly.value = (isEditable.value)? false:true;
    dl_port_table_hotSettings.value.readOnly = isReadOnly.value;
    down_link_sub_carriers_table_hotSettings.value.readOnly = isReadOnly.value;
    dl_frequency_table_hotSettings.value.readOnly = isReadOnly.value;
    console.log(dl_port_table_hotSettings.value.readOnly);
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
            sub_carriers: [[32], [128]],
            frequencies: [["DF",""], ["F1",""], ["F2",""]],
        }),
    },
});
// if(props.data.ports == undefined) props.data.ports = [["EV"], ["AEV"], ["GLOBAL"]];
// if(props.data.sub_carriers == undefined) props.data.sub_carriers = [[32], [128]];
// if(props.data.frequencies == undefined) props.data.frequencies = [["DF",""], ["F1",""], ["F2",""]];

const ports = ref(props.data.ports);

let dl_port_table_hotSettings = ref({ height: 150, width: '100%', stretchH: 'all', colHeaders: ["DL PORTS"], readOnly: true });

const sub_carriers = ref(props.data.sub_carriers);
let down_link_sub_carriers_table_hotSettings = ref({ contextMenu: false, height: 150, width: '100%', stretchH: 'all', colHeaders: ["Sub Carriers(kHz)"], readOnly: true });

const frequencies = ref(props.data.frequencies);
let dl_frequency_table_hotSettings = ref({ height: 150, width: '100%', stretchH: 'all', colHeaders: ["Frequency Label","Frequency(MHz)"], colWidths: [200, 200], readOnly: true });

const get_data = () => {
    return {
        ports: ports.value,
        sub_carriers: sub_carriers.value,
        frequencies: frequencies.value
    }
}
defineExpose({
    get_data
})

</script>