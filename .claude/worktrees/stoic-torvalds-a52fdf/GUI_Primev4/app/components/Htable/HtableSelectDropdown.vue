<template>
  <div>
    <select v-model="selectedValue" @change="onChange">
      <option v-for="option in options" :key="option.code" :value="option.name">
        {{ option.name }}
      </option>
    </select>
  </div>
</template>

<script setup lang="ts">
interface Option {
  code: string;
  name: string;
}

interface Props {
  modelValue?: string;
  options?: Option[];
}

const props = withDefaults(defineProps<Props>(), {
  modelValue: '',
  options: () => []
});

const emit = defineEmits<{
  'update:modelValue': [value: string];
}>();

const selectedValue = ref(props.modelValue);

onMounted(() => {
  console.log("✅ HtableSelectDropdown Mounted, Initial Value:", props.modelValue);
});

const onChange = (event: Event) => {
  const target = event.target as HTMLSelectElement;
  console.log("🔥 onChange Event Triggered, Value:", target.value);
  emit("update:modelValue", selectedValue.value);
};
</script>
