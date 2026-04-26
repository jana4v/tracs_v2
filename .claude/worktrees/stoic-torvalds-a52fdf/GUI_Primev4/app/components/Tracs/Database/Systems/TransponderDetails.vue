
<template>
    <div>
        <div class="grid mt-1">
            <p class="mb-0 text-2xl w-10 font-semibold text-cyan-400">Transponder</p>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Name</p>
                <InputText id="transponder_name" placeholder="Example: C Transponder 1" v-model="transponder_name"
                    type="text" :class="{ 'p-invalid': tp_name_err_msg }" aria-describedby="text-error"
                    :disabled="!isEditable" />
                <div><span class="p-error" id="text-error">{{ tp_name_err_msg || '&nbsp;' }}</span></div>

            </div>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Code</p>
                <InputText :disabled="!isEditable" id="transponder_code" placeholder="Example: CTP1"
                    v-model="transponder_code" type="text" :class="{ 'p-invalid': tp_code_err_msg }"
                    aria-describedby="text-error" />
                <div><span class="p-error" id="text-error">{{ tp_code_err_msg || '&nbsp;' }}</span></div>
            </div>


        </div>

        <div class="grid mt-0">
            <div class="col-12">

                <HandsonTableHandSonTbl :data="transponder_mapping" :hotSettings="transponder_table_hot_settings"
                    :key="counter">
                </HandsonTableHandSonTbl>
            </div>
        </div>

        <div class="grid p-buttonset">
            <Button v-if="isEditable" @click="save" label="Save" icon="pi pi-check" />
            <Button v-if="!isEditable" @click="edit" label="Edit" icon="pi pi-pencil" />
            <Button @click="delete_record" label="Delete" icon="pi pi-trash" />
        </div>

    </div>
</template>


<script lang='ts' setup>
import { useAPIFetch } from '@/composables/restApi';
import * as yup from 'yup';
import FormNotFound from '@/components/Tracs/ModulationForms/FormNotFound.vue'
import PskFm from '@/components/Tracs/ModulationForms/PskFM.vue'
import CDMA from '@/components/Tracs/ModulationForms/CDMA.vue'
import InlineMessage from 'primevue/inlinemessage';


const transponder_mapping = ref([{"up_link_config":"","down_link_config":""}]);
type DataPropType = {
    name: string,
    code: string,
    mapping_details: any,
};
type EventFunction = (event: any) => void;

const props = defineProps({
    data: {
        type: Object as () => DataPropType,
        default: () => ({
            name: "C Transponder 1",
            code: "CTX1",
            mapping_details: {},
        }),
    },
    onEvent: {
        type: Function as PropType<EventFunction>
    }
});

transponder_mapping.value = props.data.mapping_details? props.data.mapping_details:[{"up_link_config":"","down_link_config":""}];

let name = props.data?.name ? props.data.name : '';
let code = props.data?.code ? props.data.code : '';
let is_new_tx = props.data?.name ? false : true;

let name_validator = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9 ]{4,}[a-zA-Z0-9]+$/);
let code_validator = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9]{2,}$/);



const transponder_name = ref(name);
const tp_name_err_msg = ref('');
const transponder_code = ref(code);
const tp_code_err_msg = ref('');

const isEditable = ref(false);
const componentKey = ref(0);



onMounted(async () => {
    isEditable.value = props.data?.code ? false : true;
    transponder_table_hot_settings.value.readOnly = !isEditable.value;
    afterGuiAttached();
    
})


const counter = ref(0);

const transponder_table_hot_settings = ref({
    height: 150,
    width: '100%',
    stretchH: 'all',
    colHeaders: ["Uplink", "Downlink"],
    readOnly: false,
    columns: [
        {
            data:'up_link_config',
            type: 'dropdown',
            source: []
        },
        {
            data:'down_link_config',
            type: 'dropdown',
            source: []
        },
    ]
});
useAPIFetch("tracs/configurations/transmitter/get").then((res) => {
    if (res.error.value == null && res.data.value.length > 0) {
        transponder_table_hot_settings.value.columns[1].source = res.data.value.map((item: any) => item.config_label);
        counter.value += 1;
    }
});


useAPIFetch("tracs/configurations/receiver/get").then((res) => {
    if (res.error.value == null && res.data.value.length > 0) {
        transponder_table_hot_settings.value.columns[0].source = res.data.value.map((item: any) => item.config_label);
        counter.value += 1;
    }
});



const afterGuiAttached = () => {

};

const rest_api_save = (tp_data: any) => {
    useAPIFetch("tracs/systems/transponder/save", { method: 'post', body: tp_data }).then((res) => {

        if (res.error.value == null) {
            props.onEvent({ severity: 'success', summary: 'Status', detail: 'Database Updated', life: 3000 });
            isEditable.value = false;
            transponder_table_hot_settings.value.readOnly = !isEditable.value;
            componentKey.value += 1;
            counter.value += 1;
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Update Database', life: 5000 });
        }


    });

}

const delete_record = () => {
    useAPIFetch(`tracs/systems/transponder/delete/${transponder_code.value}`, { method: 'delete' }).then((res) => {
        if (res.error.value == null) {
            props.onEvent({ severity: 'success', summary: 'Status', detail: 'Record Deleteted From Database', life: 3000 });
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Delete Record From Database', life: 5000 });
        }


    });
}






const names_and_code = inject('tp_names_and_code', { names: [""], codes: [""] });

const save = async () => {
    let names_and_code_local = JSON.parse(JSON.stringify(names_and_code.value));
    let name_index = names_and_code_local.names.indexOf(name);
    if (name_index != -1) names_and_code_local.names.splice(name_index, 1);
    let code_index = names_and_code_local.codes.indexOf(code);
    if (code_index != -1) names_and_code_local.codes.splice(code_index, 1);



    if (names_and_code_local.names.includes(transponder_name.value)) {
        tp_name_err_msg.value = "Transponder name already exists"
        return;
    } else { tp_name_err_msg.value = ""; }

    if (names_and_code_local.codes.includes(transponder_code.value)) {

        tp_code_err_msg.value = "Transponder code already exists"
        return;
    }

    let tp_name_is_valid = await name_validator.isValid(transponder_name.value);
    let tp_code_is_valid = await code_validator.isValid(transponder_code.value);
    if (!tp_name_is_valid) {
        tp_name_err_msg.value = "Atleast 3 characters required"
    } else { tp_name_err_msg.value = ""; }
    if (!tp_code_is_valid) {
        tp_code_err_msg.value = "Atleast 3 characters required(Space not allowed)"
    } else { tp_code_err_msg.value = ""; }


    let data = {
        name: transponder_name.value,
        code: transponder_code.value,
        mapping_details: transponder_mapping.value
    }

    if (tp_name_err_msg.value.length == 0 && tp_code_err_msg.value.length == 0) {
        rest_api_save(data);

    }

}

const edit = () => {
    isEditable.value = true;
    transponder_table_hot_settings.value.readOnly = !isEditable.value;
    componentKey.value += 1;
    counter.value += 1;

}




</script>
