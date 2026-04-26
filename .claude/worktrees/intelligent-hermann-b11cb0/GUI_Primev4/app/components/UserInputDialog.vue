<template>
  <Dialog
    v-model:visible="visible"
   
    modal
    :closable="false"
    :draggable="false"
    :style="{ width: '380px', borderRadius: '18px', overflow: 'hidden' }"
    contentStyle="padding: 0; background: transparent;"
  >
    <div class="bg-white dark:bg-gray-900 rounded-2xl shadow-xl px-8 py-6 flex flex-col gap-4">
      <!-- Title -->
      <div class="text-2xl font-bold text-gray-900 dark:text-white mb-1 tracking-tight">
        {{ title }}
      </div>
      <!-- Message -->
      <div class="text-base text-gray-600 dark:text-gray-300 mb-2 leading-relaxed">
        {{ message }}
      </div>
      <!-- Input -->
      <div v-if="input && (input.type === 'text' || !input.type)">
        <InputText
          v-model="inputValueText"
          :placeholder="input.placeholder"
          class="w-full bg-transparent text-gray-900 dark:text-white border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-2 mt-1 focus:ring-2 focus:ring-blue-400 transition"
        />
      </div>
      <div v-else-if="input && input.type === 'number'">
        <InputNumber
          v-model="inputValueNumber"
          :placeholder="input.placeholder"
          class="w-full bg-transparent text-gray-900 dark:text-white border border-gray-300 dark:border-gray-700 rounded-lg px-3 py-2 mt-1 focus:ring-2 focus:ring-blue-400 transition"
          inputClass="w-full"
        />
      </div>
      <!-- Divider -->
      <div class="border-t border-gray-200 dark:border-gray-800 my-2"></div>
      <!-- Actions -->
      <div class="flex justify-end gap-4 mt-2">
        <Button
          v-for="option in options"
          :key="option"
          :label="option"
          @click="handleOption(option)"
          :class="[
            'px-5 py-2 rounded-xl font-semibold transition-all duration-200',
            option.toLowerCase() === 'approve'
              ? 'bg-green-500 hover:bg-green-600 text-white'
              : option.toLowerCase() === 'reject'
                ? 'bg-red-500 hover:bg-red-600 text-white'
                : 'bg-gray-100 hover:bg-gray-200 dark:bg-gray-800 dark:hover:bg-gray-700 text-gray-700 dark:text-gray-200'
          ]"
        />
      </div>
    </div>
  </Dialog>
</template>

<script setup lang="ts">
import type { DialogInputOption } from '~/stores/dialogRequest'

interface Props {
  title: string
  message: string
  options: string[]
  input?: DialogInputOption
}
const props = defineProps<Props>()
const emits = defineEmits(['resolved'])
const visible = ref(true)

// For text input
const inputValueText = ref<string>('')

// For number input
const inputValueNumber = ref<number | null>(null)

// When dialog opens, initialize values
onMounted(() => {
  if (props.input) {
    if (props.input.type === 'number') {
      if (typeof props.input.default === 'number') {
        inputValueNumber.value = props.input.default
      } else if (typeof props.input.default === 'string' && props.input.default !== '') {
        inputValueNumber.value = Number(props.input.default)
      }
    } else {
      if (props.input.default !== undefined && props.input.default !== null) {
        inputValueText.value = props.input.default.toString()
      }
    }
  }
})

function handleOption(option: string) {
  visible.value = false
  if (props.input) {
    if (props.input.type === 'number') {
      emits('resolved', { option, input: inputValueNumber.value })
    } else {
      emits('resolved', { option, input: inputValueText.value })
    }
  } else {
    emits('resolved', option)
  }
}
</script>
