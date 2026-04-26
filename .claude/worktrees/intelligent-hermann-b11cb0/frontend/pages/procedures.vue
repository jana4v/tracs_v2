<script setup lang="ts">
import * as monaco from 'monaco-editor'
import editorWorker from 'monaco-editor/esm/vs/editor/editor.worker?worker'
import jsonWorker from 'monaco-editor/esm/vs/language/json/json.worker?worker'
import cssWorker from 'monaco-editor/esm/vs/language/css/css.worker?worker'
import htmlWorker from 'monaco-editor/esm/vs/language/html/html.worker?worker'
import tsWorker from 'monaco-editor/esm/vs/language/typescript/ts.worker?worker'
import type { Procedure, ProcedureVersion } from '~/types/astra'

definePageMeta({ title: 'Procedures' })

const api = useAstraApi()
const settingsStore = useSettingsStore()
const monacoSettings = useMonacoSettings()
const procStore = useProceduresStore()
const { setup: setupMonaco } = useMonaco()
// LSP disabled - using backend API completions instead
// const { connect: connectLsp, disconnect: disconnectLsp, isConnected: lspConnected } = useLspClient()
const lspConnected = ref(false)

// Dropdown selections
const selectedProject = ref<string>('')
const selectedProcedure = ref<string>('')
const selectedVersionOption = ref<string>('')

// Compare pane selections
const compareProject = ref<string>('')
const compareProcedure = ref<string>('')
const compareVersion = ref<string>('')

// Save remarks
const saveRemarks = ref<string>('')

// Toast
const toast = useToast()

// Editor instances
const primaryEditorContainer = ref<HTMLElement>()
const secondEditorContainer = ref<HTMLElement>()
let primaryEditor: monaco.editor.IStandaloneCodeEditor | null = null
let secondEditor: monaco.editor.IStandaloneCodeEditor | null = null
let monacoInitialized = false

function ensureMonacoEnv() {
  if (monacoInitialized) return
  const globalScope = window as unknown as {
    MonacoEnvironment?: { getWorker: (_: string, label: string) => Worker }
  }
  if (!globalScope.MonacoEnvironment) {
    globalScope.MonacoEnvironment = {
      getWorker: (_: string, label: string) => {
        if (label === 'json') return new jsonWorker()
        if (label === 'css' || label === 'scss' || label === 'less') return new cssWorker()
        if (label === 'html' || label === 'handlebars' || label === 'razor') return new htmlWorker()
        if (label === 'typescript' || label === 'javascript') return new tsWorker()
        return new editorWorker()
      },
    }
  }
  setupMonaco(monaco)
  monacoInitialized = true
}

function createEditor(container: HTMLElement, readOnly: boolean = false) {
  return monaco.editor.create(container, {
    value: '',
    language: 'astra',
    theme: monacoSettings.editorTheme,
    automaticLayout: true,
    minimap: { enabled: false },
    fontSize: monacoSettings.editorFontSize,
    lineNumbers: 'on',
    scrollBeyondLastLine: false,
    wordWrap: 'on',
    glyphMargin: true,
    folding: true,
    renderLineHighlight: 'line',
    cursorBlinking: 'smooth',
    readOnly,
  })
}

// Computed dropdown options
const projectOptions = computed(() => {
  const projects = new Set<string>()
  procStore.versionedList.forEach(p => {
    if (p.project) projects.add(p.project)
  })
  return Array.from(projects).map(p => ({ label: p, value: p }))
})

const procedureOptions = computed(() => {
  if (!selectedProject.value) return []
  return procStore.versionedList
    .filter(p => p.project === selectedProject.value)
    .map(p => ({ 
      label: `${p.test_name} (v${p.latest_version})`, 
      value: p.test_name 
    }))
})

const versionOptions = computed(() => {
  return procStore.availableVersions.map(v => ({
    label: `v${v.version} - ${v.created_by} (${new Date(v.created_at).toLocaleDateString()})`,
    value: `${v.version}|${v.created_by}`,
  }))
})

// Compare dropdown options
const compareProcedureOptions = computed(() => {
  if (!compareProject.value) return []
  return procStore.versionedList
    .filter(p => p.project === compareProject.value)
    .map(p => ({ 
      label: `${p.test_name} (v${p.latest_version})`, 
      value: p.test_name 
    }))
})

const compareVersionOptions = computed(() => {
  return procStore.secondAvailableVersions.map(v => ({
    label: `v${v.version} - ${v.created_by} (${new Date(v.created_at).toLocaleDateString()})`,
    value: `${v.version}|${v.created_by}`,
  }))
})

// Load procedure list
async function loadProcedureList() {
  procStore.setLoading(true)
  try {
    const result = await api.getProcedures(selectedProject.value || undefined)
    procStore.setVersionedList(result.procedures)
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load procedures: ${e.message}`, life: 3000 })
  } finally {
    procStore.setLoading(false)
  }
}

// Watch for project selection changes - reload list for selected project
watch(selectedProject, async () => {
  selectedProcedure.value = ''
  selectedVersionOption.value = ''
  saveRemarks.value = ''
  procStore.resetEditor()
  if (primaryEditor) {
    primaryEditor.setValue('')
  }
  await loadProcedureList()
})

// Watch for procedure selection changes
watch(selectedProcedure, async (testName) => {
  if (testName) {
    saveRemarks.value = ''
    await loadProcedure(testName)
  } else {
    selectedVersionOption.value = ''
  }
})

// Watch for version selection changes
watch(selectedVersionOption, async (versionStr) => {
  if (versionStr && selectedProcedure.value) {
    const version = parseInt(versionStr.split('|')[0])
    await loadProcedureVersion(selectedProcedure.value, version)
  }
})

// Compare watchers
watch(compareProject, () => {
  compareProcedure.value = ''
  compareVersion.value = ''
})

watch(compareProcedure, async (testName) => {
  if (testName) {
    await loadSecondProcedure(testName)
  } else {
    compareVersion.value = ''
  }
})

watch(compareVersion, async (versionStr) => {
  if (versionStr && compareProcedure.value) {
    const version = parseInt(versionStr.split('|')[0])
    await loadSecondProcedureVersion(compareProcedure.value, version)
  }
})

// Load a specific procedure into primary editor
async function loadProcedure(testName: string, version?: number) {
  const project = selectedProject.value || undefined
  try {
    const [proc, versionsRes] = await Promise.all([
      api.getProcedure(testName, project),
      api.getProcedureVersions(testName, project),
    ])
    const versions = versionsRes.versions
    procStore.setAvailableVersions(versions)

    const targetVersion = version
      ? versions.find(v => v.version === version)
      : versions[0]
    const content = targetVersion
      ? targetVersion.content
      : (proc.latest_content ?? '')
    procStore.setSelectedProcedure(testName, targetVersion?.version ?? null)
    procStore.setCurrentContent(content)
    procStore.setOriginalContent(content)
    procStore.currentProject = proc.project
    procStore.currentCreatedBy = (targetVersion?.created_by ?? proc.updated_by ?? proc.created_by) ?? ''
    if (primaryEditor) {
      primaryEditor.setValue(content)
    }
    if (targetVersion) {
      selectedVersionOption.value = `${targetVersion.version}|${targetVersion.created_by}`
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load procedure: ${e.message}`, life: 3000 })
  }
}

// Load specific version
async function loadProcedureVersion(testName: string, version: number) {
  const v = procStore.availableVersions.find(v => v.version === version)
  if (v) {
    procStore.setSelectedProcedure(testName, version)
    procStore.setCurrentContent(v.content)
    procStore.setOriginalContent(v.content)
    procStore.currentProject = v.project
    procStore.currentCreatedBy = v.created_by
    if (primaryEditor) primaryEditor.setValue(v.content)
  }
}

// Load a procedure into the second pane
async function loadSecondProcedure(testName: string, version?: number) {
  const project = compareProject.value || undefined
  try {
    const [proc, versionsRes] = await Promise.all([
      api.getProcedure(testName, project),
      api.getProcedureVersions(testName, project),
    ])
    const versions = versionsRes.versions
    procStore.setSecondAvailableVersions(versions)

    const targetVersion = version ? versions.find(v => v.version === version) : versions[0]
    const content = targetVersion ? targetVersion.content : (proc.latest_content ?? '')
    procStore.setSecondPane(testName, targetVersion?.version ?? null)
    procStore.setSecondContent(content)
    if (secondEditor) {
      secondEditor.setValue(content)
    }
    if (targetVersion) {
      compareVersion.value = `${targetVersion.version}|${targetVersion.created_by}`
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Error', detail: `Failed to load second procedure: ${e.message}`, life: 3000 })
  }
}

// Load specific version for second pane
async function loadSecondProcedureVersion(testName: string, version: number) {
  const v = procStore.secondAvailableVersions.find(v => v.version === version)
  if (v) {
    procStore.setSecondPane(testName, version)
    procStore.setSecondContent(v.content)
    if (secondEditor) secondEditor.setValue(v.content)
  }
}

// Validate
async function handleValidate() {
  if (!procStore.currentContent.trim()) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'No content to validate', life: 3000 })
    return
  }
  
  // Clear validation state before validating
  procStore.setValidated(false)
  procStore.clearProblems()
  
  try {
    const loadResult = await api.loadProcedure(procStore.currentContent, '<procedure-editor>')
    if (!loadResult.success) {
      const errorMsg = (loadResult as any).error || 'Failed to parse procedure'
      toast.add({ 
        severity: 'error', 
        summary: 'Parse Error', 
        detail: errorMsg, 
        life: 5000 
      })
      procStore.setValidated(false)
      
      // Set markers on editor
      if (primaryEditor) {
        const model = primaryEditor.getModel()
        if (model) {
          monaco.editor.setModelMarkers(model, 'astra', [{
            severity: monaco.MarkerSeverity.Error,
            startLineNumber: 1,
            startColumn: 1,
            endLineNumber: 1,
            endColumn: 1000,
            message: errorMsg,
          }])
        }
      }
      return
    }

    const validateResult = await api.validateProcedure(loadResult.test_name)
    if (validateResult.valid) {
      procStore.clearProblems()
      procStore.setValidated(true)
      toast.add({ severity: 'success', summary: 'Valid', detail: 'Procedure validation passed', life: 3000 })

      // Clear markers on editor
      if (primaryEditor) {
        const model = primaryEditor.getModel()
        if (model) monaco.editor.setModelMarkers(model, 'astra', [])
      }
    } else {
      procStore.setProblems(validateResult.errors)
      procStore.setValidated(false)
      
      const errorCount = validateResult.errors.filter(e => e.severity === 'error').length
      const warnCount = validateResult.errors.length - errorCount
      
      let detail = `Found ${errorCount} error(s)`
      if (warnCount > 0) detail += ` and ${warnCount} warning(s)`
      
      toast.add({ severity: 'error', summary: 'Validation Failed', detail, life: 5000 })

      // Set error markers on editor
      if (primaryEditor) {
        const model = primaryEditor.getModel()
        if (model) {
          const markers = validateResult.errors.map(p => ({
            severity: p.severity === 'error' ? monaco.MarkerSeverity.Error : monaco.MarkerSeverity.Warning,
            startLineNumber: p.line_number,
            startColumn: 1,
            endLineNumber: p.line_number,
            endColumn: 1000,
            message: p.message,
          }))
          monaco.editor.setModelMarkers(model, 'astra', markers)
        }
      }
    }
  } catch (e: any) {
    procStore.setValidated(false)
    toast.add({ severity: 'error', summary: 'Error', detail: `Validation error: ${e.message}`, life: 5000 })
  }
}

// Save
async function handleSave() {
  if (!procStore.currentContent.trim()) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'No content to save', life: 3000 })
    return
  }

  // CRITICAL: Enforce validation before saving
  if (!procStore.isValidated) {
    toast.add({ 
      severity: 'error', 
      summary: 'Cannot Save', 
      detail: 'Please validate the procedure first. Validation must pass before saving.', 
      life: 5000 
    })
    return
  }

  // Additional check: ensure no errors
  if (procStore.hasProblems && procStore.errorCount > 0) {
    toast.add({ 
      severity: 'error', 
      summary: 'Cannot Save', 
      detail: `Cannot save with ${procStore.errorCount} validation error(s). Please fix errors first.`, 
      life: 5000 
    })
    return
  }

  // Check: remarks required
  if (!saveRemarks.value.trim()) {
    toast.add({ 
      severity: 'warn', 
      summary: 'Remarks Required', 
      detail: 'Please enter remarks before saving the procedure.', 
      life: 5000 
    })
    return
  }

  // Extract test_name from content
  const testNameMatch = procStore.currentContent.match(/^TEST_NAME\s+(\S+)/m)
  const testName = testNameMatch ? testNameMatch[1] : procStore.selectedTestName
  if (!testName) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'Could not determine TEST_NAME from content', life: 3000 })
    return
  }

  const project = selectedProject.value || settingsStore.globalProject
  if (!project) {
    toast.add({ severity: 'warn', summary: 'Warning', detail: 'Please select a project first', life: 3000 })
    return
  }

  procStore.saving = true
  try {
    const result = await api.saveProcedure(
      testName,
      procStore.currentContent,
      project,
      settingsStore.username || 'system',
      undefined,
      undefined,
      saveRemarks.value.trim(),
    )

    if (result.saved) {
      procStore.setOriginalContent(procStore.currentContent)
      saveRemarks.value = ''
      toast.add({
        severity: 'success',
        summary: 'Saved',
        detail: `Saved ${testName} v${result.version}`,
        life: 3000,
      })
      await loadProcedureList()
      // Reload versions
      if (procStore.selectedTestName === testName) {
        await loadProcedure(testName)
      }
    } else {
      toast.add({
        severity: 'info',
        summary: 'No Changes',
        detail: result.message || 'Content unchanged, not saved',
        life: 3000,
      })
    }
  } catch (e: any) {
    toast.add({ severity: 'error', summary: 'Save Error', detail: e.message, life: 5000 })
  } finally {
    procStore.saving = false
  }
}

// New procedure
function handleNew() {
  procStore.resetEditor()
  selectedProcedure.value = ''
  selectedVersionOption.value = ''
  saveRemarks.value = ''
  const project = selectedProject.value || 'default'
  const content = `TEST_NAME new-procedure\nPRE_TEST_REQ TM1.STATUS == "OK"\nSEND START\nWAIT 5\n`
  procStore.setCurrentContent(content)
  procStore.setOriginalContent('')
  procStore.currentProject = project
  if (primaryEditor) {
    primaryEditor.setValue(content)
  }
}

// Mount
onMounted(async () => {
  ensureMonacoEnv()
  await loadProcedureList()

  nextTick(() => {
    if (primaryEditorContainer.value) {
      primaryEditor = createEditor(primaryEditorContainer.value)
      primaryEditor.onDidChangeModelContent(() => {
        const newContent = primaryEditor!.getValue()
        procStore.setCurrentContent(newContent)
        // Reset validation on content change
        if (newContent !== procStore.originalContent) {
          procStore.setValidated(false)
        }
      })

      lspConnected.value = true // Completions registered via useMonaco.setup()
    }
  })
})

// Watch dual pane - create/destroy second editor
watch(() => procStore.dualPaneMode, (isDual) => {
  if (isDual) {
    nextTick(() => {
      if (secondEditorContainer.value && !secondEditor) {
        secondEditor = createEditor(secondEditorContainer.value, true)
      }
    })
  } else {
    secondEditor?.dispose()
    secondEditor = null
    compareProject.value = ''
    compareProcedure.value = ''
    compareVersion.value = ''
  }
})

// Watch font size
watch(() => monacoSettings.editorFontSize, (size) => {
  primaryEditor?.updateOptions({ fontSize: size })
  secondEditor?.updateOptions({ fontSize: size })
})

// Watch theme changes
watch(() => monacoSettings.editorTheme, (theme) => {
  monaco.editor.setTheme(theme)
})

onUnmounted(() => {
  primaryEditor?.dispose()
  secondEditor?.dispose()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <!-- Top Toolbar with Dropdowns -->
    <div class="flex items-center gap-3 px-4 py-2.5 border-b border-[var(--astra-border)] bg-[var(--astra-surface)]">
      <!-- File Operations -->
      <div class="flex items-center gap-1">
        <Button
          icon="pi pi-file-plus"
          size="small"
          severity="secondary"
          text
          v-tooltip.bottom="'New Procedure'"
          @click="handleNew"
        />
      </div>

      <Divider layout="vertical" class="mx-1 h-6" />

      <!-- Project Dropdown -->
      <div class="flex items-center gap-2">
        <label class="text-xs font-medium text-[var(--astra-muted)]">Project:</label>
        <Select
          v-model="selectedProject"
          :options="projectOptions"
          option-label="label"
          option-value="value"
          placeholder="Select project..."
          class="w-48"
          size="small"
          filter
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)]' },
            label: { class: 'text-[var(--astra-text)]' },
            input: { class: 'text-[var(--astra-text)]' },
            panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
            item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
          }"
        />
      </div>

      <!-- Procedure Dropdown -->
      <div class="flex items-center gap-2">
        <label class="text-xs font-medium text-[var(--astra-muted)]">Procedure:</label>
        <Select
          v-model="selectedProcedure"
          :options="procedureOptions"
          option-label="label"
          option-value="value"
          placeholder="Select procedure..."
          :disabled="!selectedProject"
          class="w-64"
          size="small"
          filter
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)]' },
            label: { class: 'text-[var(--astra-text)]' },
            input: { class: 'text-[var(--astra-text)]' },
            panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
            item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
          }"
        />
      </div>

      <!-- Version Dropdown -->
      <div class="flex items-center gap-2">
        <label class="text-xs font-medium text-[var(--astra-muted)]">Version:</label>
        <Select
          v-model="selectedVersionOption"
          :options="versionOptions"
          option-label="label"
          option-value="value"
          placeholder="Select version..."
          :disabled="!selectedProcedure || versionOptions.length === 0"
          class="w-80"
          size="small"
          filter
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)]' },
            label: { class: 'text-[var(--astra-text)]' },
            input: { class: 'text-[var(--astra-text)]' },
            panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
            item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
          }"
        />
      </div>

      <div class="flex-1" />

      <!-- Remarks Input -->
      <div class="flex items-center gap-2 mr-4">
        <InputText
          v-model="saveRemarks"
          placeholder="Remarks (required)"
          class="w-48 h-8 text-sm"
          :pt="{
            root: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)] text-[var(--astra-text)]' }
          }"
        />
      </div>

      <!-- Action Buttons -->
      <div class="flex items-center gap-2">
        <Button
          icon="pi pi-check-circle"
          label="Validate"
          size="small"
          severity="info"
          outlined
          :disabled="!procStore.currentContent.trim()"
          @click="handleValidate"
        />
        <Button
          icon="pi pi-save"
          label="Save"
          size="small"
          severity="success"
          :disabled="!procStore.isDirty || !procStore.isValidated || procStore.saving || procStore.errorCount > 0"
          :loading="procStore.saving"
          @click="handleSave"
        />
      </div>

      <!-- Status Tags -->
      <div class="flex items-center gap-2 text-xs ml-2">
        <Tag v-if="lspConnected" severity="success" value="API" icon="pi pi-check-circle" v-tooltip.bottom="'Backend completions active'" />
        <Tag v-if="procStore.isDirty" severity="warn" value="Modified" />
        <Tag v-if="procStore.isValidated" severity="success" value="Valid" icon="pi pi-check" />
        <Tag v-if="procStore.errorCount > 0" severity="danger" :value="`${procStore.errorCount} errors`" />
      </div>
    </div>
    <!-- Main Content: Editor Area -->
    <div class="flex-1 flex overflow-hidden">
      <!-- Primary Editor (70% width) -->
      <div class="w-[70%] flex flex-col overflow-hidden" :class="{ 'border-r border-[var(--astra-border)]': procStore.dualPaneMode }">
        <div ref="primaryEditorContainer" class="h-full w-full" />
      </div>

      <!-- Console Panel (30% width) -->
      <div class="w-[30%] flex flex-col overflow-hidden bg-[var(--astra-surface)] border-l border-[var(--astra-border)]">
        <div class="flex items-center gap-2 px-4 py-2 border-b border-[var(--astra-border)] bg-[var(--astra-surface-2)]">
          <i class="pi pi-terminal text-[var(--astra-accent)]" />
          <span class="text-sm font-medium text-[var(--astra-text)]">Console</span>
          <div class="flex-1" />
          <Tag v-if="procStore.isValidated" severity="success" value="Valid" icon="pi pi-check" />
          <Tag v-if="procStore.errorCount > 0" severity="danger" :value="`${procStore.errorCount} errors`" />
        </div>

        <!-- Validation Output -->
        <div class="flex-1 overflow-auto p-3 font-mono text-xs">
          <div v-if="!procStore.currentContent.trim()" class="text-[var(--astra-muted)] italic">
            No content to validate
          </div>
          <div v-else-if="!procStore.hasProblems && !procStore.isValidated" class="text-[var(--astra-muted)]">
            Click "Validate" to check for errors
          </div>
          <div v-else-if="procStore.isValidated && !procStore.hasProblems" class="text-green-400">
            <i class="pi pi-check-circle mr-1" />
            Validation passed - no errors found
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="(problem, i) in procStore.problems"
              :key="i"
              class="p-2 rounded"
              :class="problem.severity === 'error' ? 'bg-red-900/20 border border-red-500/30' : 'bg-yellow-900/20 border border-yellow-500/30'"
            >
              <div class="flex items-center gap-2 mb-1">
                <i :class="problem.severity === 'error' ? 'pi pi-times-circle text-red-400' : 'pi pi-exclamation-circle text-yellow-400'" />
                <span class="text-[var(--astra-muted)]">Line {{ problem.line_number }}</span>
                <Tag :severity="problem.severity === 'error' ? 'danger' : 'warn'" :value="problem.severity" class="text-[10px]" />
              </div>
              <div class="text-[var(--astra-text)]">{{ problem.message }}</div>
            </div>
          </div>
        </div>
      </div>

      <!-- Second Editor (Compare) -->
      <div v-if="procStore.dualPaneMode" class="flex-1 flex flex-col overflow-hidden bg-[var(--astra-bg)]">
        <!-- Compare controls -->
        <div class="flex items-center gap-3 px-4 py-2 border-b border-[var(--astra-border)] bg-[var(--astra-surface)]">
          <span class="text-xs font-medium text-[var(--astra-text)]">Compare with:</span>
          
          <!-- Compare Project -->
          <div class="flex items-center gap-2">
            <label class="text-xs text-[var(--astra-muted)]">Project:</label>
            <Select
              v-model="compareProject"
              :options="projectOptions"
              option-label="label"
              option-value="value"
              placeholder="Select project..."
              class="w-40"
              size="small"
              filter
              :pt="{
                root: { class: 'bg-[var(--astra-surface-2)]' },
                input: { class: 'text-[var(--astra-text)]' },
                panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
                item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
              }"
            />
          </div>

          <!-- Compare Procedure -->
          <div class="flex items-center gap-2">
            <label class="text-xs text-[var(--astra-muted)]">Procedure:</label>
            <Select
              v-model="compareProcedure"
              :options="compareProcedureOptions"
              option-label="label"
              option-value="value"
              placeholder="Select procedure..."
              :disabled="!compareProject"
              class="w-56"
              size="small"
              filter
              :pt="{
                root: { class: 'bg-[var(--astra-surface-2)]' },
                input: { class: 'text-[var(--astra-text)]' },
                panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
                item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
              }"
            />
          </div>

          <!-- Compare Version -->
          <div class="flex items-center gap-2">
            <label class="text-xs text-[var(--astra-muted)]">Version:</label>
            <Select
              v-model="compareVersion"
              :options="compareVersionOptions"
              option-label="label"
              option-value="value"
              placeholder="Select version..."
              :disabled="!compareProcedure || compareVersionOptions.length === 0"
              class="w-64"
              size="small"
              filter
              :pt="{
                root: { class: 'bg-[var(--astra-surface-2)]' },
                input: { class: 'text-[var(--astra-text)]' },
                panel: { class: 'bg-[var(--astra-surface-2)] border-[var(--astra-border)]' },
                item: { class: 'text-[var(--astra-text)] hover:bg-[var(--astra-accent)]/10' }
              }"
            />
          </div>
        </div>

        <div ref="secondEditorContainer" class="flex-1" />
      </div>
    </div>
  </div>
</template>