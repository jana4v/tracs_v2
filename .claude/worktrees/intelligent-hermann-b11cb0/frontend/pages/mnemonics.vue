<script setup lang="ts">
import type { TMRef, TCRef, SCORef } from '~/types/astra'

definePageMeta({ title: 'Mnemonics' })

const api = useAstraApi()
const mnemonicsStore = useMnemonicsStore()
const loading = ref(true)
const activeTab = ref(0)

onMounted(async () => {
  await fetchAll()
})

async function fetchAll() {
  loading.value = true
  try {
    const [tm, tc, sco] = await Promise.allSettled([
      api.getAllTMMnemonics(),
      api.getAllTCMnemonics(),
      api.getAllSCOCommands(),
    ])

    if (tm.status === 'fulfilled') mnemonicsStore.setTMMnemonics(tm.value)
    if (tc.status === 'fulfilled') mnemonicsStore.setTCMnemonics(tc.value)
    if (sco.status === 'fulfilled') mnemonicsStore.setSCOCommands(sco.value)
  } catch (e) {
    console.error('Mnemonics fetch error:', e)
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="p-4 space-y-4">
    <div class="flex items-center justify-between">
      <h1 class="text-2xl font-bold text-[var(--astra-text)]">Mnemonics Database</h1>
      <Button icon="pi pi-refresh" label="Refresh" severity="secondary" @click="fetchAll" />
    </div>

    <Card class="panel-card">
      <template #content>
        <TabView v-model:active-index="activeTab">
          <!-- TM Mnemonics -->
          <TabPanel header="Telemetry (TM)">
            <DataTable
              :value="mnemonicsStore.tmMnemonics"
              :loading="loading"
              :rows="20"
              paginator
              striped-rows
              :global-filter-fields="['full_ref', 'description', 'subsystem']"
              class="text-sm"
            >
              <Column field="full_ref" header="Reference" sortable />
              <Column field="bank" header="Bank" sortable />
              <Column field="mnemonic" header="Mnemonic" sortable />
              <Column field="description" header="Description" />
              <Column field="data_type" header="Type">
                <template #body="{ data }">
                  <Tag :value="data.data_type" severity="info" />
                </template>
              </Column>
              <Column field="unit" header="Unit" />
              <Column field="subsystem" header="Subsystem">
                <template #body="{ data }">
                  <Tag :value="data.subsystem" severity="secondary" />
                </template>
              </Column>
              <template #empty>
                <div class="text-center text-muted py-4">No TM mnemonics found.</div>
              </template>
            </DataTable>
          </TabPanel>

          <!-- TC Mnemonics -->
          <TabPanel header="Telecommand (TC)">
            <DataTable
              :value="mnemonicsStore.tcMnemonics"
              :loading="loading"
              :rows="20"
              paginator
              striped-rows
              class="text-sm"
            >
              <Column field="full_ref" header="Reference" sortable />
              <Column field="command" header="Command" sortable />
              <Column field="description" header="Description" />
              <Column field="subsystem" header="Subsystem">
                <template #body="{ data }">
                  <Tag :value="data.subsystem" severity="secondary" />
                </template>
              </Column>
              <Column field="category" header="Category" />
              <Column header="Parameters">
                <template #body="{ data }">
                  <span v-if="data.parameters?.length">
                    {{ data.parameters.map((p: any) => p.name).join(', ') }}
                  </span>
                  <span v-else class="text-muted">none</span>
                </template>
              </Column>
              <template #empty>
                <div class="text-center text-muted py-4">No TC mnemonics found.</div>
              </template>
            </DataTable>
          </TabPanel>

          <!-- SCO Commands -->
          <TabPanel header="SCO Commands">
            <DataTable
              :value="mnemonicsStore.scoCommands"
              :loading="loading"
              :rows="20"
              paginator
              striped-rows
              class="text-sm"
            >
              <Column field="full_ref" header="Reference" sortable />
              <Column field="command" header="Command" sortable />
              <Column field="description" header="Description" />
              <Column field="subsystem" header="Subsystem">
                <template #body="{ data }">
                  <Tag :value="data.subsystem" severity="secondary" />
                </template>
              </Column>
              <Column field="category" header="Category" />
              <template #empty>
                <div class="text-center text-muted py-4">No SCO commands found.</div>
              </template>
            </DataTable>
          </TabPanel>
        </TabView>
      </template>
    </Card>
  </div>
</template>
