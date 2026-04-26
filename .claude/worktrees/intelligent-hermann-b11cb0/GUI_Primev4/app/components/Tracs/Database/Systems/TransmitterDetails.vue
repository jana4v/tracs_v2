
<template>
    <div>
        <div class="grid mt-1">
            <p class="mb-0 text-2xl w-10 font-semibold text-cyan-400">Transmitter</p>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Name</p>
                <InputText id="transmitter_name" placeholder="Example: C Transmitter 1" v-model="transmitter_name"
                    type="text" :class="{ 'p-invalid': tx_name_err_msg }" aria-describedby="text-error"
                    :disabled="!isEditable" />
                <div><span class="p-error" id="text-error">{{ tx_name_err_msg || '&nbsp;' }}</span></div>

            </div>
            <div class="col-4">
                <p class="mb-0 text-xl w-10">Code</p>
                <InputText :disabled="!isEditable" id="transmitter_code" placeholder="Example: CTX1"
                    v-model="transmitter_code" type="text" :class="{ 'p-invalid': tx_code_err_msg }"
                    aria-describedby="text-error" />
                <div><span class="p-error" id="text-error">{{ tx_code_err_msg || '&nbsp;' }}</span></div>
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
import PskPm from '@/components/Tracs/ModulationForms/PskPM.vue'
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
            name: "C Transmitter 1",
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
const transmitter_name = ref(name);
const tx_name_err_msg = ref('');
const transmitter_code = ref(code);
const tx_code_err_msg = ref('');
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

const rest_api_save = (tx_data: any) => {
    useAPIFetch("tracs/saveTransmitterData", { method: 'post', body: tx_data }).then((res) => {

        if (res.error.value == null) {
           let toast_msg = { severity: 'success', summary: 'Status', detail: 'Database Updated', life: 3000 };
            props.onEvent(toast_msg);
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Update Database', life: 5000 });
        }


    });

}

const delete_record = () => {
    useAPIFetch(`tracs/deleteTransmitterData/${transmitter_code.value}`, { method: 'delete' }).then((res) => {
        if (res.error.value == null) {
            props.onEvent({ severity: 'success', summary: 'Status', detail: 'Record Deleteted From Database', life: 3000 });
        } else {
            props.onEvent({ severity: 'error', summary: 'Error', detail: 'Failed to Delete Record From Database', life: 5000 });
        }


    });
}



watch(selectedModulationType, async (new_value, old_value) => {
   
    if (new_value == 'PSK_PM') {
        modulation_component_name.value = markRaw(PskPm);
        is_form_not_found.value = false;
    } else if (new_value == 'FSK_FM') {
        modulation_component_name.value = markRaw(CDMA);
        is_form_not_found.value = false;
    } else {
        modulation_component_name.value = markRaw(FormNotFound);
        is_form_not_found.value = true;
    }

})


const names_and_code = inject('tx_names_and_code', { names: [""], codes: [""] });

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


    if (names_and_code_local.names.includes(transmitter_name.value)) {
        tx_name_err_msg.value = "Transmitter name already exists"
        return;
    } else { tx_name_err_msg.value = ""; }

    if (names_and_code_local.codes.includes(transmitter_code.value)) {

        tx_code_err_msg.value = "Transmitter code already exists"
        return;
    }


    if (is_form_not_found.value) return;
    let tx_name_is_valid = await name_validator.isValid(transmitter_name.value);
    let tx_code_is_valid = await code_validator.isValid(transmitter_code.value);
    if (!tx_name_is_valid) {
        tx_name_err_msg.value = "Atleast 3 characters required"
    } else { tx_name_err_msg.value = ""; }
    if (!tx_code_is_valid) {
        tx_code_err_msg.value = "Atleast 3 characters required(Space not allowed)"
    } else { tx_code_err_msg.value = ""; }


    let data = {
        name: transmitter_name.value,
        code: transmitter_code.value,
        modulation: selectedModulationType.value,
        modulation_details: modulation_form.value.get_data()
    }
    console.log(data);

    if (tx_name_err_msg.value.length == 0 && tx_code_err_msg.value.length == 0 && modulation_form_not_found.value == false) {
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
