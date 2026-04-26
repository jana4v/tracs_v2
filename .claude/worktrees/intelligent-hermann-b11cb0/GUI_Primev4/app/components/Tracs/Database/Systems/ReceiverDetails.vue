
<template>
    <div>
        <div class="grid mt-1">
            <p class="mb-0 text-2xl w-10 font-semibold text-cyan-400">Receiver</p>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Name</p>
                <InputText id="receiver_name" placeholder="Example: C Receiver 1" v-model="receiver_name" type="text"
                    :class="{ 'p-invalid': rx_name_err_msg }" aria-describedby="text-error" :disabled="!isEditable" />
                <div><span class="p-error" id="text-error">{{ rx_name_err_msg || '&nbsp;' }}</span></div>

            </div>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Code</p>
                <InputText :disabled="!isEditable" id="receiver_code" placeholder="Example: CRX1" v-model="receiver_code"
                    type="text" :class="{ 'p-invalid': rx_code_err_msg }" aria-describedby="text-error" />
                <div><span class="p-error" id="text-error">{{ rx_code_err_msg || '&nbsp;' }}</span></div>
            </div>
            <div class="col-4">

                <p class="mb-0 text-xl w-10">Modulation</p>
                <Dropdown :disabled="!isEditable" v-model="selectedModulationType" :options="downLinkModulationTypes" filter
                    inputId="select_modulation_type" optionLabel="value" optionValue="value" :placeholder="placeholder"
                    class="w-full">
                </Dropdown>
            </div>

        </div>

        <div class="grid mt-0">
            <div class="col-12">
                <component ref="modulation_form" :is="modulation_component_name" :data="props.data.modulation_details">
                </component>
                <!-- <tracs-modulation-forms-psk-pm :isEditable="isEditable" :key="componentKey" v-if="selectedModulationType.value == 'psk_pm'"></tracs-modulation-forms-psk-pm>
            <tracs-modulation-forms-cdma v-if="selectedModulationType.value == 'fsk'"></tracs-modulation-forms-cdma> -->
            </div>
        </div>
        <div class="pb-5">
            <InlineMessage v-if="modulation_form_not_found" severity="error">Select Modulation Type before saving
            </InlineMessage>
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


type DataPropType = {
    name: string,
    code: string,
    modulation_type: string,
    modulation_details: any,
    id: string
};
type EventFunction = (event: any) => void;

const props = defineProps({
    data: {
        type: Object as () => DataPropType,
        default: () => ({
            name: "C Receiver 1",
            code: "CTX1",
            modulation_type: '',
            modulation_details: {},
            id: ''
        }),
    },
    onEvent: {
        type: Function as PropType<EventFunction>
    }
});


let name = props.data?.name ? props.data.name : '';
let code = props.data?.code ? props.data.code : '';
let modulation_type = props.data?.modulation_type ? props.data.modulation_type : ''
let is_new_tx = props.data?.name ? false : true;

let name_validator = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9 ]{4,}[a-zA-Z0-9]+$/);
let code_validator = yup.string().matches(/^[a-zA-Z][a-zA-Z0-9]{2,}$/);


const modulation_form = ref(null);
const selectedModulationType = ref();
const receiver_name = ref(name);
const rx_name_err_msg = ref('');
const receiver_code = ref(code);
const rx_code_err_msg = ref('');
const placeholder = ref(' Select Modulation Type');
const downLinkModulationTypes = ref([]);
const ports = ref([{ port: 'EV', code: 'EV' }, { port: 'AEV', code: 'AEV' }, { port: 'GLOBAL', code: 'GLOBAL' }]);
const selectedPort = ref('');
const opoptions = ref([]);
const option = ref({});
const isEditable = ref(false);
const componentKey = ref(0);
const is_form_not_found = ref(false);
const modulation_component_name = ref();
let modulation_form_not_found = ref(false);

watch(modulation_form_not_found, (newValue) => {
    if (newValue) {
        setTimeout(() => {
            modulation_form_not_found.value = false;
        }, 3000);
    }
});

provide("isEditable", computed(() => isEditable.value));

onMounted(async () => {
    isEditable.value = props.data?.code ? false : true;
    afterGuiAttached();
    selectedModulationType.value = modulation_type;
})


const afterGuiAttached = () => {

    useAPIFetch("tracs/getModulationTypes").then((res) => {

        if (res.error.value == null && res.data.value.length > 0) {
            downLinkModulationTypes.value = res.data.value.map((opt: any) => ({ value: opt }));
        }
    });

};

const rest_api_save = (rx_data: any) => {
    useAPIFetch("tracs/saveReceiverData", { method: 'post', body: rx_data }).then((res) => {

        if (res.error.value == null) {
            props.onEvent({ severity: 'success', summary: 'Status', detail: 'Database Updated', life: 3000 });
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Update Database', life: 5000 });
        }


    });

}

const delete_record = () => {
    useAPIFetch(`tracs/deleteReceiverData/${receiver_code.value}`, { method: 'delete' }).then((res) => {
        if (res.error.value == null) {
            props.onEvent({ severity: 'success', summary: 'Status', detail: 'Record Deleteted From Database', life: 3000 });
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Delete Record From Database', life: 5000 });
        }


    });
}



watch(selectedModulationType, async (new_value, old_value) => {
    if (new_value == 'PSK_FM') {
        modulation_component_name.value = markRaw(PskFm);
        is_form_not_found.value = false;
    } else if (new_value == 'FSK_FM') {
        modulation_component_name.value = markRaw(CDMA);
        is_form_not_found.value = false;
    } else {
        modulation_component_name.value = markRaw(FormNotFound);
        is_form_not_found.value = true;
    }

})


const names_and_code = inject('rx_names_and_code', { names: [""], codes: [""] });

const save = async () => {
    let names_and_code_local = JSON.parse(JSON.stringify(names_and_code.value));
    let name_index = names_and_code_local.names.indexOf(name);
    if (name_index != -1) names_and_code_local.names.splice(name_index, 1);
    let code_index = names_and_code_local.codes.indexOf(code);
    if (code_index != -1) names_and_code_local.codes.splice(code_index, 1);

    console.log(names_and_code_local);

    if (is_form_not_found.value == true) {
        modulation_form_not_found.value = true;
    }


    if (names_and_code_local.names.includes(receiver_name.value)) {
        rx_name_err_msg.value = "Receiver name already exists"
        return;
    } else { rx_name_err_msg.value = ""; }

    if (names_and_code_local.codes.includes(receiver_code.value)) {

        rx_code_err_msg.value = "Receiver code already exists"
        return;
    }


    if (is_form_not_found.value) return;
    let rx_name_is_valid = await name_validator.isValid(receiver_name.value);
    let rx_code_is_valid = await code_validator.isValid(receiver_code.value);
    if (!rx_name_is_valid) {
        rx_name_err_msg.value = "Atleast 3 characters required"
    } else { rx_name_err_msg.value = ""; }
    if (!rx_code_is_valid) {
        rx_code_err_msg.value = "Atleast 3 characters required(Space not allowed)"
    } else { rx_code_err_msg.value = ""; }


    let data = {
        name: receiver_name.value,
        code: receiver_code.value,
        modulation: selectedModulationType.value,
        modulation_details: modulation_form.value.get_data()
    }
    console.log(data);

    if (rx_name_err_msg.value.length == 0 && rx_code_err_msg.value.length == 0 && modulation_form_not_found.value == false) {
        rest_api_save(data);
        isEditable.value = false;
        componentKey.value += 1;
    }

}

const edit = () => {
    isEditable.value = true;
    componentKey.value += 1;

}

</script>
