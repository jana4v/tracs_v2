<template>
    <div :key="counter">
        <Button class="mx-2" @click="add_receiver">Add Recceiver</Button>
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
import receiver from '@/components/Tracs/Database/Systems/ReceiverDetails.vue'
import { useToast } from "primevue/usetoast";

const toast = useToast();

const components = ref([{ "name": "", "data": "" }]);

const rx_names_and_code = ref({ names: [], codes: [] });
provide('rx_names_and_code', rx_names_and_code);

const add_receiver = () => {
    components.value.push({ "name": markRaw(receiver), "data": {} });
}

const get_data_from_api = async () => {
    components.value = [];
    let res = await useAPIFetch("tracs/getReceivers")
    let rx_names = [];
    let tx_codes = [];
    if (res.error.value == null) {

        res.data.value.forEach(data => {
            rx_names.push(data.name);
            tx_codes.push(data.code);
        });
        rx_names_and_code.value = { names: rx_names, codes: tx_codes }
        res.data.value.forEach(data => {
            components.value.push({ "name": markRaw(receiver), "data": data });
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
