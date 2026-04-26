# Component Structure Guidelines

## 🏗️ Architecture Improvements

### Current Issues Found:
1. **Component Naming Conflict**: `customEditor.vue` vs `customEditor.js` in Htable folder
2. **Missing TypeScript types** in some components
3. **No global error boundary** (now added to app.vue)

### Recommended Structure:

```
app/components/
├── shared/           # Reusable components
│   ├── buttons/
│   ├── forms/
│   └── modals/
├── features/         # Feature-specific components
│   ├── gisat/
│   ├── mainframe/
│   └── tc/
└── layout/          # Layout components
    ├── AppTopbar.vue
    └── AppSidebar.vue
```

### Best Practices:

1. **Use TypeScript**: Add `lang="ts"` to all `<script>` blocks
2. **Component Naming**: Use PascalCase for components, kebab-case for files
3. **Props Validation**: Always define props with types and defaults
4. **Emit Events**: Use `defineEmits` for type safety
5. **Composables**: Extract reusable logic to composables

### Example Component Template:

```vue
<script setup lang="ts">
interface Props {
  title: string
  disabled?: boolean
}

interface Emits {
  click: [event: MouseEvent]
  update: [value: string]
}

const props = withDefaults(defineProps<Props>(), {
  disabled: false
})

const emit = defineEmits<Emits>()

// Component logic here
</script>

<template>
  <!-- Template here -->
</template>

<style scoped>
/* Component styles */
</style>
```