<template>
    <div :key="counter">
        <Button class="mx-2" @click="add_transponder">Add Transponder</Button>
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
import transponder from '@/components/Tracs/Database/Systems/TransponderDetails.vue'
import { useToast } from "primevue/usetoast";

const toast = useToast();

const components = ref([{ "name": "", "data": "" }]);

const tp_names_and_code = ref({ names: [], codes: [] });
provide('tp_names_and_code', tp_names_and_code);

const add_transponder = () => {
    components.value.push({ "name": markRaw(transponder), "data": {} });
}

const get_data_from_api = async () => {
    components.value = [];
    let res = await useAPIFetch("tracs/systems/transponder/get")
    let tp_names = [];
    let tp_codes = [];
    if (res.error.value == null) {

        res.data.value.forEach(data => {
            tp_names.push(data.name);
            tp_codes.push(data.code);
        });
        tp_names_and_code.value = { names: tp_names, codes: tp_codes }
        res.data.value.forEach(data => {
            console.log(data);
            components.value.push({ "name": markRaw(transponder), "data": data });
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
