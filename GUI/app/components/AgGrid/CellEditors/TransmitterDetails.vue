<script>
import { useToast } from 'primevue/usetoast'
import { defineComponent, reactive, ref, toRefs } from 'vue'
import { useAPIFetch } from '@/composables/restApi'

export default defineComponent({
  name: 'TransmitterDetails',
  components: {
    // executionStatus: executionStatus,
  },
  props: ['params'],
  setup(props) {
    const dropdownRef = ref(null)
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props
    const style = ref({ width: '100%' })
    style.value = { width: params.eGridCell.style.width }
    const data = reactive({
      transmitter_name: '',
      transmitter_code: '',
      modulation_type: '',
      selectedModulation: '',
      downLinkModulationTypes: [],
      ports: [],
      selectedPort: '',
      errorMessage: ' bvcbcb cvb',
      opoptions: [],
      option: {},
    })

    const toast = useToast()

    console.log(params)
    const getValue = () => {
      // return data.option?.value || params.value.name;
      return 'OK'
    }

    const getGui = () => {

    }
    const afterGuiAttached = () => {
      useAPIFetch('tracs/getModulationTypes').then((res) => {
        if (res.error.value == null && res.data.value.length > 0) {
          data.downLinkModulationTypes = res.data.value.map(opt => ({ value: opt }))
        }
      })
    }
    const removeEditor = () => {

    }
    const isPopup = () => {
      // and we could leave this method out also, false is the default
      return false
    }
    const ValueChanged = () => {
      setTimeout(() => params.api.redrawRows(), 3000)
    }

    const rows = ref([])

    const addNewRow = () => {
      rows.value.push({
        id: Date.now(), // unique id for each new row
        data: 'Hello',
        editing: true,
      })
    }
    const editRow = (row) => {
      row.editing = true
    }
    const saveRow = (row) => {
      row.editing = false
    }
    const deleteRow = (row) => {
      rows.value = rows.value.filter(r => r !== row)
    }

    return {
      ...toRefs(data),
      getValue,
      style,
      ValueChanged,
      props,
      dropdownRef,
      afterGuiAttached,
      isPopup,
      removeEditor,
      getGui,
      addNewRow,
      editRow,
      saveRow,
      deleteRow,

    }
  },
  mounted() {
    // nextTick(() => {
    //   this.$refs.container.focus();
    // });
    // console.log(this.$refs.dropdownRef);
    // this.$refs.dropdownRef.showPopup('');
  },
})
</script>

<template>
  <Card>
    <template #title>
      Transmitter Details
    </template>
    <template #content>
      <div class="grid mt-1">
        <div class="col">
          <h6 class="mb-0">
            Transmitter Name
          </h6>
          <InputText
            id="transmitter_name" v-model="transmitter_name" placeholder="Example: C Transmitter 1" type="text" :class="{ 'p-invalid': errorMessage }"
            aria-describedby="text-error"
          />
          <div><span id="text-error" class="p-error">{{ errorMessage || '&nbsp;' }}</span></div>
        </div>
        <div class="col">
          <h6 class="mb-0">
            Transmitter Code
          </h6>
          <InputText
            id="transmitter_code" v-model="transmitter_code" placeholder="Example: CTX1" type="text" :class="{ 'p-invalid': errorMessage }"
            aria-describedby="text-error"
          />
          <div><span id="text-error" class="p-error">{{ errorMessage || '&nbsp;' }}</span></div>
        </div>
      </div>

      <div class="grid">
        <div class="col">
          <h6 class="mb-0">
            Down Link Ports
          </h6>
          <Button label="Add New Row" @click="addNewRow" />
          <DataTable :value="rows">
            <Column field="data" header="Data">
              <template #body="slotProps">
                <div class="flex align-items-center justify-content-between">
                  <div>
                    <InputText v-if="slotProps.data.editing" v-model="slotProps.data.data" />
                    <span v-else>{{ slotProps.data.data }}</span>
                  </div>
                  <div>
                    <Button
                      v-if="!slotProps.data.editing" icon="pi pi-pencil" severity="success" text rounded
                      aria-label="Edit" @click="() => editRow(slotProps.data)"
                    />
                    <Button
                      v-if="slotProps.data.editing" icon="pi pi-check" severity="success" text rounded
                      aria-label="Save" @click="() => saveRow(slotProps.data)"
                    />
                    <Button
                      icon="pi pi-trash" severity="success" text rounded aria-label="Delete"
                      class="p-button-danger" @click="() => deleteRow(slotProps.data)"
                    />
                  </div>
                </div>
              </template>
            </Column>
          </DataTable>
        </div>

        <div class="col">
          <h6 class="mb-0">
            Select Modulation Type
          </h6>
          <Dropdown
            v-model="selectedModulation" :options="downLinkModulationTypes" filter input-id="select_modulation_type" option-label="value" placeholder=" Select Modulation Type"
            class="w-full"
          />
        </div>
      </div>

      <br>
      <h1>TEST</h1>

      <span class="p-buttonset">
        <Button label="Save" icon="pi pi-check" @click="ValueChanged" />
        <Button label="Cancel" icon="pi pi-times" @click="ValueChanged" />
      </span>
    </template>
  </Card>

  <!--
  <Dropdown ref="dropdownRef" v-model="option" :options="options" filter optionLabel="value"
    @hide="ValueChanged" :placeholder='props.params.placeholder' :style=style />  -->
</template>
