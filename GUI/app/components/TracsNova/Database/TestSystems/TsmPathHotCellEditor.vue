<script lang="ts">
import { defineComponent, reactive, ref, toRefs } from 'vue';
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main.css';
import 'handsontable/styles/ht-icons-main.css';

function suppressHotThemeFalsePositiveWarning() {
  const g = globalThis as any;
  if (g.__hotThemeWarnPatched) return;

  const originalWarn = console.warn.bind(console);
  console.warn = (...args: any[]) => {
    const first = args[0];
    if (
      typeof first === 'string' &&
      first.includes('theme is enabled, but its stylesheets are missing')
    ) {
      return;
    }
    originalWarn(...args);
  };

  g.__hotThemeWarnPatched = true;
}

suppressHotThemeFalsePositiveWarning();
registerAllModules();

export default defineComponent({
  name: 'TsmPathHotCellEditor',
  components: { HotTable },
  props: ['params'],
  setup(props) {
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props;
    const isDark = useDark();

    const tableData = ref<Array<{ path: string }>>([{ path: '' }, { path: '' }, { path: '' }, { path: '' }]);
    if (Array.isArray(params?.value) && params.value.length > 0) {
      tableData.value = params.value;
    }

    const data = reactive({
      tableSettings: {
        ...(params?.tbl_settings ?? {}),
        licenseKey: (params?.tbl_settings?.licenseKey ?? 'non-commercial-and-evaluation'),
        themeName: isDark.value ? 'ht-theme-main-dark' : 'ht-theme-main',
      } as Record<string, any>,
    });

    const getValue = () => tableData.value;
    const isPopup = () => true;
    const closeEditor = () => params?.stopEditing?.();
    const cancelEditor = () => params?.stopEditing?.(true);

    function onKeyDown(event: KeyboardEvent) {
      if (event.key === 'Enter') {
        event.preventDefault();
        event.stopPropagation();
      }
    }

    return {
      ...toRefs(data),
      tableData,
      getValue,
      isPopup,
      closeEditor,
      cancelEditor,
      onKeyDown,
    };
  },
});
</script>

<template>
  <div class="tsm-path-editor" @mousedown.stop @click.stop>
    <HotTable :settings="tableSettings" :data="tableData" @keydown="onKeyDown" />
    <div class="editor-actions">
      <button class="btn btn-secondary" type="button" @click="cancelEditor">Cancel</button>
      <button class="btn btn-primary" type="button" @click="closeEditor">OK</button>
    </div>
  </div>
</template>

<style scoped>
.tsm-path-editor {
  width: 520px;
  max-width: calc(100vw - 3rem);
  background: #0b1220;
  border: 1px solid #1f2937;
  border-radius: 6px;
  padding: 0.5rem;
}

.editor-actions {
  display: flex;
  justify-content: flex-end;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.btn {
  border: 1px solid #334155;
  border-radius: 4px;
  padding: 0.28rem 0.7rem;
  font-size: 0.8rem;
  cursor: pointer;
}

.btn-secondary {
  background: #0f172a;
  color: #cbd5e1;
}

.btn-primary {
  background: #0891b2;
  border-color: #0891b2;
  color: #ecfeff;
}
</style>
