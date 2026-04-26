<template>
    <div :key="counter">
        <Button class="mx-2" @click="add_transmitter">Add Transmitter</Button>
        <Toast />
        <ScrollPanel style="height: calc(100vh - 18rem);padding: 0.5rem;">
            <div v-for="(component, index) in components" :key="index">
                <component :onEvent="cb_for_child" class="card m-4" :is="component.name" :data="component.data"
                    v-if="component.name != ''" />
            </div>
        </ScrollPanel>
    </div>
</template>

<script setup>
import transmitter from '@/components/Tracs/Database/Systems/TransmitterDetails.vue'
import { useToast } from "primevue/usetoast";

const toast = useToast();
const components = ref([{ "name": "", "data": "" }]);

const tx_names_and_code = ref({ names: [], codes: [] });
provide('tx_names_and_code', tx_names_and_code);

const add_transmitter = () => {
    components.value.push({ "name": markRaw(transmitter), "data": {} });
}

const get_data_from_api = async () => {
    components.value = [];
    let res = await useAPIFetch("tracs/getTransmitters")

        let tx_names = [];
        let tx_codes = [];
        if (res.error.value == null) {

            res.data.value.forEach(data => {
                tx_names.push(data.name);
                tx_codes.push(data.code);
            });
            tx_names_and_code.value = { names: tx_names, codes: tx_codes }
            res.data.value.forEach(data => {
                components.value.push({ "name": markRaw(transmitter), "data": data });
            });


        }
        counter.value += 1;
}


onMounted(async () => {
    await get_data_from_api();
});

const counter = ref(0);
const cb_for_child = async (data) => {
    await get_data_from_api();
    toast.add(data);
}


</script>
