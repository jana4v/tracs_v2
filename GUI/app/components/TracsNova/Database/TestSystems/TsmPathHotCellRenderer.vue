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
  name: 'TsmPathHotCellRenderer',
  components: { HotTable },
  props: ['params'],
  setup(props) {
    // eslint-disable-next-line vue/no-setup-props-destructure
    const { params } = props;
    const isDark = useDark();

    const style = ref({ width: '100%' });
    style.value = params?.style ?? { width: '100%' };

    const data = reactive({
      tableData: [] as Array<{ path: string }>,
      tableSettings: {
        ...(params?.tbl_settings ?? {}),
        readOnly: false,
        contextMenu: ['row_above', 'row_below', 'remove_row'],
        licenseKey: (params?.tbl_settings?.licenseKey ?? 'non-commercial-and-evaluation'),
        themeName: isDark.value ? 'ht-theme-main-dark' : 'ht-theme-main',
      } as Record<string, any>,
    });

    if (Array.isArray(params?.value)) {
      data.tableData = params.value;
    }

    if (!Array.isArray(data.tableData) || data.tableData.length === 0) {
      data.tableData = [{ path: '' }, { path: '' }, { path: '' }, { path: '' }];
    }

    const rowsCount = Math.max(data.tableData.length || 0, 4);
    const desiredHeight = Math.min(100 + rowsCount * 34, 380);
    params?.node?.setRowHeight?.(desiredHeight);
    setTimeout(() => params?.api?.onRowHeightChanged?.(), 0);

    const getValue = () => data.tableData;

    return {
      ...toRefs(data),
      style,
      getValue,
    };
  },
});
</script>

<template>
  <div class="my-1 p-1" :style="style">
    <HotTable :settings="tableSettings" :data="tableData" />
  </div>
</template>
