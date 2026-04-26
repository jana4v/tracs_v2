<template>
  <div class="flex items-center w-full h-full gap-1">
    <select
      v-if="!loading && !error && values.length > 0"
      ref="selectRef"
      :value="currentValue"
      @change="onSelectChange"
      class="flex-1 h-full px-2 bg-transparent border-none outline-none"
      style="color: inherit; font: inherit;"
    >
      <option v-for="v in values" :key="v" :value="v">{{ v }}</option>
    </select>
    <input
      v-else
      ref="inputRef"
      v-model="currentValue"
      class="flex-1 h-full px-2 bg-transparent border-none outline-none"
      style="color: inherit; font: inherit;"
      placeholder="Loading or no options..."
    />
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

const props = defineProps<{
  mnemonic?: string
  value?: any
}>()

const emit = defineEmits(['updateValue'])

const api = useAstraApi()
const inputRef = ref<HTMLInputElement | null>(null)
const selectRef = ref<HTMLSelectElement | null>(null)
const currentValue = ref('')
const values = ref<string[]>([])
const loading = ref(true)
const error = ref(false)

onMounted(async () => {
  currentValue.value = props.value || ''
  
  if (props.mnemonic) {
    try {
      const result = await api.getMnemonicRange(props.mnemonic)
      if (result && result.range) {
        values.value = result.range
      }
    } catch (e) {
      console.error('Failed to fetch range for', props.mnemonic, e)
      error.value = true
    }
  }
  
  loading.value = false
  
  setTimeout(() => {
    if (selectRef.value) {
      selectRef.value.focus()
    } else if (inputRef.value) {
      inputRef.value.focus()
    }
  }, 0)
})

function onSelectChange(event: Event) {
  const target = event.target as HTMLSelectElement
  emit('updateValue', target.value)
}
</script>
