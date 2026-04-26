<script setup lang="ts">
definePageMeta({ title: 'Settings' })

const settingsStore = useSettingsStore()
const monacoSettings = useMonacoSettings()
const api = useAstraApi()
const toast = useToast()

onMounted(() => {
  settingsStore.loadPreferences()
  monacoSettings.loadPreferences()
})

function saveSettings() {
  settingsStore.savePreferences()
  monacoSettings.savePreferences()
}

const modeOptions = [
  { label: 'Simulation', value: 'simulation' },
  { label: 'Hardware', value: 'hardware' },
]

const themeOptions = [
  { label: 'Dark', value: 'ASTRA-dark' },
  { label: 'Light', value: 'ASTRA-light' },
]

// TM Import
const selectedFile = ref<File | null>(null)
const uploading = ref(false)
const uploadResult = ref<{ filename: string; stats: { total: number; inserted: number; updated: number; skipped: number; errors: string[] } } | null>(null)

function onFileSelected(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] ?? null
  uploadResult.value = null
}

async function uploadFile() {
  if (!selectedFile.value) return
  uploading.value = true
  uploadResult.value = null
  try {
    const result = await api.uploadTelemetryFile(selectedFile.value)
    uploadResult.value = result
    const s = result.stats
    toast.add({
      severity: s.errors.length > 0 ? 'warn' : 'success',
      summary: 'Import Complete',
      detail: `${s.inserted} inserted, ${s.updated} updated, ${s.skipped} skipped` + (s.errors.length > 0 ? `, ${s.errors.length} errors` : ''),
      life: 5000,
    })
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Import Failed', detail: e.message || 'Unknown error', life: 5000 })
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="p-4 space-y-4">
    <h1 class="text-2xl font-bold text-[var(--astra-text)]">Settings</h1>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
      <!-- Execution Settings -->
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-play" />
            <span>Execution</span>
          </div>
        </template>
        <template #content>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">Execution Mode</label>
              <SelectButton
                v-model="settingsStore.mode"
                :options="modeOptions"
                option-label="label"
                option-value="value"
                :allow-empty="false"
              />
            </div>

            <div class="flex items-center justify-between">
              <label class="text-sm font-medium text-[var(--astra-text)]">Auto-validate on load</label>
              <ToggleSwitch v-model="settingsStore.autoValidate" />
            </div>
          </div>
        </template>
      </Card>

      <!-- Editor Settings -->
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-code" />
            <span>Editor</span>
          </div>
        </template>
        <template #content>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">Font Size</label>
              <InputNumber
                v-model="monacoSettings.editorFontSize"
                :min="10"
                :max="24"
                show-buttons
                button-layout="horizontal"
                :step="1"
              />
            </div>

            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">Editor Theme</label>
              <SelectButton
                v-model="monacoSettings.editorTheme"
                :options="themeOptions"
                option-label="label"
                option-value="value"
                :allow-empty="false"
              />
            </div>
          </div>
        </template>
      </Card>

      <!-- Project Settings -->
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-folder" />
            <span>Project</span>
          </div>
        </template>
        <template #content>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">Global Project Name</label>
              <InputText v-model="settingsStore.globalProject" class="w-full" placeholder="e.g. gsat7r" />
            </div>
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">Username</label>
              <InputText v-model="settingsStore.username" class="w-full" placeholder="e.g. user1" />
            </div>
          </div>
        </template>
      </Card>

      <!-- Telemetry Settings -->
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-wave-pulse" />
            <span>Telemetry</span>
          </div>
        </template>
        <template #content>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">TM Poll Interval (ms)</label>
              <InputNumber
                v-model="settingsStore.tmPollInterval"
                :min="500"
                :max="10000"
                :step="500"
                show-buttons
                button-layout="horizontal"
                suffix=" ms"
              />
            </div>
          </div>
        </template>
      </Card>

      <!-- TM Import -->
      <Card class="panel-card">
        <template #title>
          <div class="flex items-center gap-2">
            <i class="pi pi-upload" />
            <span>TM Import</span>
          </div>
        </template>
        <template #content>
          <div class="space-y-4">
            <div>
              <label class="block text-sm font-medium mb-2 text-[var(--astra-text)]">
                Upload TM Parameter Table (.xlsx or .out)
              </label>
              <input
                type="file"
                name="file"
                accept=".xlsx,.out"
                class="block w-full text-sm text-[var(--astra-text)] file:mr-3 file:py-1.5 file:px-3 file:rounded file:border-0 file:text-sm file:bg-[var(--astra-accent)]/20 file:text-[var(--astra-accent)] hover:file:bg-[var(--astra-accent)]/30"
                @change="onFileSelected"
              />
            </div>
            <Button
              label="Upload & Import"
              icon="pi pi-upload"
              :loading="uploading"
              :disabled="!selectedFile"
              @click="uploadFile"
            />

            <div v-if="uploadResult" class="text-sm space-y-1 p-3 rounded bg-[var(--astra-border)]/20">
              <div class="font-medium text-[var(--astra-text)]">{{ uploadResult.filename }}</div>
              <div class="grid grid-cols-2 gap-x-4 gap-y-1 text-xs">
                <span class="text-muted">Total:</span>
                <span>{{ uploadResult.stats.total }}</span>
                <span class="text-muted">Inserted:</span>
                <span class="text-emerald-400">{{ uploadResult.stats.inserted }}</span>
                <span class="text-muted">Updated:</span>
                <span class="text-amber-400">{{ uploadResult.stats.updated }}</span>
                <span class="text-muted">Skipped:</span>
                <span>{{ uploadResult.stats.skipped }}</span>
              </div>
              <div v-if="uploadResult.stats.errors.length > 0" class="mt-2">
                <div class="text-xs text-red-400 font-medium">Errors ({{ uploadResult.stats.errors.length }}):</div>
                <div class="max-h-32 overflow-auto text-xs text-red-300 mt-1">
                  <div v-for="(err, i) in uploadResult.stats.errors" :key="i">{{ err }}</div>
                </div>
              </div>
            </div>
          </div>
        </template>
      </Card>
    </div>

    <!-- Save Button -->
    <div class="flex justify-end">
      <Button label="Save Preferences" icon="pi pi-save" @click="saveSettings" />
    </div>
  </div>
</template>
