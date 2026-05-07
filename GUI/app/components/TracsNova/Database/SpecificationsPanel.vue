<template>
  <div class="specs-panel" :class="{ 'single-view': isSingleParameterView }">
    <div class="specs-header">
      <h2>Specifications</h2>
     </div>

    <div
      v-for="parameter in visibleParameters"
      :key="parameter"
      class="spec-section"
      :class="{ 'fill-height': isSingleParameterView }"
    >
      <div class="spec-section-header">
        <div>
          <h3>{{ sectionTitles[parameter] }}</h3>
        
        </div>
        <div class="actions">
          <Button label="Refresh" size="small" severity="secondary" @click="loadParameter(parameter)" />
          <Button
            label="Save"
            size="small"
            :loading="saving[parameter]"
            @click="saveParameter(parameter)"
          />
        </div>
      </div>

      <ag-grid-vue
        class="spec-grid"
        :style="{ width: '100%', height: isSingleParameterView ? '100%' : '320px' }"
        :gridOptions="gridOptions"
        :theme="isDark
          ? themeQuartz.withPart(colorSchemeDarkBlue)
          : themeQuartz.withPart(colorSchemeLightCold)"
        :columnDefs="columnDefsByParameter[parameter]"
        :rowData="tableRows[parameter]"
        :enableRangeSelection="true"
        :enableFillHandle="true"
        :getRowHeight="(params) => getRowHeight(parameter, params)"
        :defaultColDef="defaultColDef"
        :rowGroupPanelShow="'always'"
        :groupDisplayType="'singleColumn'"
        :cellSelection="cellSelectionConfig"
        :suppressClickEdit="true"
        :suppressColumnVirtualisation="true"
        :suppressContextMenu="false"
        :suppressMovableColumns="true"
        :undoRedoCellEditing="true"
        :undoRedoCellEditingLimit="20"
        @cell-double-clicked="onCellDoubleClickedParameter(parameter, $event)"
        @cell-key-down="onCellKeyDownParameter(parameter, $event)"
        @grid-ready="onGridReadyParameter(parameter, $event)"
        @first-data-rendered="onFirstDataRenderedParameter(parameter, $event)"
        @model-updated="onModelUpdatedParameter(parameter, $event)"
      />
    </div>

    <Dialog
      v-model:visible="showFbtDialog"
      modal
      :header="fbtDialogTitle"
      :style="{ width: '620px' }"
      :dismissableMask="true"
    >
      <div class="editor-wrap">
        <HotTable :settings="fbtHotSettings" :data="fbtEditingData" />
      </div>

      <template #footer>
        <Button label="Cancel" icon="pi pi-times" text @click="closeFbtEditor" />
        <Button label="Save" icon="pi pi-check" @click="saveFbtEditor" />
      </template>
    </Dialog>
  </div>
</template>

<script lang="ts" setup>
import { HotTable } from '@handsontable/vue3';
import { registerAllModules } from 'handsontable/registry';
import 'handsontable/styles/handsontable.css';
import 'handsontable/styles/ht-theme-main-no-icons.css';
import Dialog from 'primevue/dialog';
import { useToast } from 'primevue/usetoast';
import { ModuleRegistry } from 'ag-grid-community';
import { AllEnterpriseModule } from 'ag-grid-enterprise';
import {
  colorSchemeDarkBlue,
  colorSchemeLightCold,
  themeQuartz,
} from 'ag-grid-community';
import type {
  CellClickedEvent,
  ColDef,
  ColGroupDef,
  FirstDataRenderedEvent,
  GridApi,
  GridReadyEvent,
  RowHeightParams,
  ModelUpdatedEvent,
} from 'ag-grid-community';
import { AgGridVue } from 'ag-grid-vue3';
import PskPmSpuriousFbtCellAgGrid from '@/components/tracsNova/ModulationForms/PskPmSpuriousFbtCellAgGrid.vue';
import {
  useTransmitterApi,
  type CatalogSpecRow,
  type ParameterName,
} from '@/composables/tracsNova/useTransmitterApi';
import { useUiStatePersistence } from '@/composables/tracsNova/useUiStatePersistence';

ModuleRegistry.registerModules([AllEnterpriseModule]);
registerAllModules();

type FbtMatrix = (string | number)[][];

const api = useTransmitterApi();
const toast = useToast();
const isDark = useDark();
const ui = useUiStatePersistence('ui_state:tracsNova:db:specifications');

const props = withDefaults(
  defineProps<{
    activeParameter?: ParameterName | 'all';
  }>(),
  {
    activeParameter: 'all',
  },
);

const parameterOrder: ParameterName[] = ['power', 'frequency', 'modulation_index', 'spurious', 'command_threshold'];
const visibleParameters = computed<ParameterName[]>(() => {
  if (props.activeParameter === 'all') return parameterOrder;
  return [props.activeParameter];
});
const isSingleParameterView = computed(() => visibleParameters.value.length === 1);

const sectionTitles: Record<ParameterName, string> = {
  power: 'Power',
  frequency: 'Frequency',
  modulation_index: 'Modulation Index',
  spurious: 'Spurious',
  command_threshold: 'Command Threshold',
};

const spuriousFbtRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });
const spuriousFbtHotRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });
const spuriousFbtColdRenderer = markRaw({ ...PskPmSpuriousFbtCellAgGrid });

type GridColumnDef = ColDef | ColGroupDef;

const tableRows = reactive<Record<ParameterName, Record<string, any>[]>>({
  power: [],
  frequency: [],
  modulation_index: [],
  spurious: [],
  command_threshold: [],
});

const columnDefsByParameter = reactive<Record<ParameterName, GridColumnDef[]>>({
  power: [],
  frequency: [],
  modulation_index: [],
  spurious: [],
  command_threshold: [],
});

const saving = reactive<Record<ParameterName, boolean>>({
  power: false,
  frequency: false,
  modulation_index: false,
  spurious: false,
  command_threshold: false,
});

const gridApis = shallowRef<Partial<Record<ParameterName, GridApi>>>({});
const showFbtDialog = ref(false);
const fbtEditingData = ref<FbtMatrix>([['', '']]);
const activeFbtField = ref('');
const activeFbtNode = shallowRef<any>(null);
const fbtDialogTitle = computed(() => {
  const displayName = activeFbtField.value === 'fbt'
    ? 'FBT'
    : activeFbtField.value === 'fbt_hot'
      ? 'FBT Hot'
      : 'FBT Cold';

  const row = (activeFbtNode.value?.data ?? {}) as Record<string, unknown>;
  const code = String(row.code ?? '').trim();
  const port = String(row.port ?? '').trim();
  const frequencyLabel = String(row.frequency_label ?? row.frequency ?? '').trim();
  const rowIdentity = [code, port, frequencyLabel].filter(Boolean).join('_');
  return rowIdentity ? `Edit ${displayName} - ${rowIdentity}` : `Edit ${displayName}`;
});

const fbtHotSettings = computed(() => ({
  licenseKey: 'non-commercial-and-evaluation',
  colHeaders: ['Offset (kHz)', 'Value (dBc)'],
  columns: [
    {
      type: 'numeric',
      locale: 'en-US',
      numericFormat: {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
        useGrouping: true,
      },
    },
    {
      type: 'numeric',
      locale: 'en-US',
      numericFormat: {
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
        useGrouping: true,
      },
    },
  ],
  rowHeaders: true,
  stretchH: 'all',
  width: '100%',
  minRows: 1,
  minSpareRows: 1,
  contextMenu: true,
  height: 380,
  autoWrapRow: true,
  autoWrapCol: true,
  copyPaste: true,
  fillHandle: {
    direction: 'vertical',
    autoInsertRow: true,
  },
  enterMoves: { row: 1, col: 0 },
  tabMoves: { row: 0, col: 1 },
}));

const defaultColDef: ColDef = {
  editable: true,
  sortable: true,
  filter: true,
  resizable: true,
  enableRowGroup: true,
  minWidth: 120,
};

const gridOptions = {
  singleClickEdit: false,
  suppressClickEdit: true,
  suppressColumnVirtualisation: true,
  suppressAnimationFrame: true,
  processCellForClipboard: (params: any) => {
    if (isFbtField(params.column?.getColId?.() ?? params.column?.colId ?? params.colDef?.field)) {
      return toFbtClipboardText(params.value);
    }
    return params.value;
  },
  processCellFromClipboard: (params: any) => {
    if (isFbtField(params.column?.getColId?.() ?? params.column?.colId ?? params.colDef?.field)) {
      return parseFbtClipboardValue(params.value);
    }
    return params.value;
  },
};

const cellSelectionConfig = {
  mode: 'range' as const,
  handle: {
    mode: 'fill' as const,
    direction: 'xy' as const,
    suppressClearOnFillReduction: true,
    setFillValue: (params: any) => {
      const values = params?.values;
      if (Array.isArray(values) && values.length > 0) {
        const sourceValue = values[values.length - 1];
        if (Array.isArray(sourceValue)) {
          return sourceValue.map((row: any) => Array.isArray(row) ? [...row] : row);
        }
        return sourceValue;
      }

      const currentValue = params?.currentCellValue;
      if (Array.isArray(currentValue)) {
        return currentValue.map((row: any) => Array.isArray(row) ? [...row] : row);
      }
      return currentValue;
    },
  },
};

const metaKeys = new Set(['transmitter_code', 'transmitter_name', 'modulation_type']);
const readOnlyKeys = new Set(['code', 'port', 'frequency_label', 'frequency']);
const structuralKeys = new Set([
  'row_id',
  'system_kind',
  'system_code',
  'parameter_type',
  'port_id',
  'frequency_id',
  'sort_order',
  'code',
  'port',
  'frequency_label',
  'frequency',
  'frequency_hz',
]);
const hiddenGridKeys = new Set([
  'row_id',
  'system_kind',
  'system_code',
  'parameter_type',
  'port_id',
  'frequency_id',
  'sort_order',
]);

function isFbtField(field: unknown): field is 'fbt' | 'fbt_hot' | 'fbt_cold' {
  return field === 'fbt' || field === 'fbt_hot' || field === 'fbt_cold';
}

function toFbtClipboardText(value: unknown): string {
  const matrix = ensureFbtMatrix(value);
  const rows = matrix
    .map((row) => row.map((cell) => `${cell ?? ''}`.trim()).filter(Boolean))
    .filter((row) => row.length > 0);

  if (!rows.length) return '';
  return rows.map((row) => row.join(', ')).join('; ');
}

function parseFbtClipboardValue(value: unknown): FbtMatrix {
  if (Array.isArray(value)) {
    return ensureFbtMatrix(value).map((row) => [...row]);
  }

  const text = `${value ?? ''}`.trim();
  if (!text) return [['', '']];

  const tabularRows = text
    .split(/\r?\n/)
    .map((line) => line.split('\t').map((cell) => cell.trim()))
    .filter((row) => row.some(Boolean));

  if (tabularRows.length > 1 || (tabularRows[0]?.length ?? 0) > 1) {
    return tabularRows.map((row) => normalizeFbtRow(row));
  }

  return text
    .split(';')
    .map((segment) => segment.trim())
    .filter(Boolean)
    .map((segment) => normalizeFbtRow(segment.split(',').map((cell) => cell.trim())));
}

function normalizeFbtRow(row: unknown[]): (string | number)[] {
  const normalized = row.slice(0, 2).map((cell) => {
    const text = `${cell ?? ''}`.trim();
    if (text === '') return '';
    const parsed = Number(text);
    return Number.isFinite(parsed) ? parsed : text;
  });

  while (normalized.length < 2) normalized.push('');
  return normalized;
}

function getSelectedFbtTargets(api: GridApi, fallbackEvent?: any): Array<{ rowNode: any; field: 'fbt' | 'fbt_hot' | 'fbt_cold' }> {
  const ranges = api.getCellRanges?.() ?? [];
  const targets: Array<{ rowNode: any; field: 'fbt' | 'fbt_hot' | 'fbt_cold' }> = [];

  for (const range of ranges) {
    const start = Math.min(range.startRow?.rowIndex ?? 0, range.endRow?.rowIndex ?? 0);
    const end = Math.max(range.startRow?.rowIndex ?? 0, range.endRow?.rowIndex ?? 0);
    const fields = (range.columns ?? [])
      .map((column: any) => column.getColId?.() ?? column.colId)
      .filter(isFbtField);

    for (let rowIndex = start; rowIndex <= end; rowIndex += 1) {
      const rowNode = api.getDisplayedRowAtIndex?.(rowIndex);
      if (!rowNode) continue;
      for (const field of fields) {
        targets.push({ rowNode, field });
      }
    }
  }

  if (targets.length > 0) return targets;

  const eventField = fallbackEvent?.colDef?.field;
  if (isFbtField(eventField) && fallbackEvent?.node) {
    return [{ rowNode: fallbackEvent.node, field: eventField }];
  }

  const focused = api.getFocusedCell?.();
  const focusedField = focused?.column?.getColId?.() ?? focused?.column?.colId;
  if (isFbtField(focusedField)) {
    const rowNode = api.getDisplayedRowAtIndex?.(focused.rowIndex);
    if (rowNode) return [{ rowNode, field: focusedField }];
  }

  return [];
}

function createBaseColumn(key: string, parameter?: ParameterName): ColDef {
  // Map of field-key → display header (with units where applicable).
  // Falls back to title-cased key when not listed.
  const headerOverrides: Record<string, string> = {
    frequency: 'Frequency (MHz)',
  };
  // Identifier fields that never carry a value-unit suffix.
  const identifierKeys = new Set([
    'code',
    'port',
    'frequency',
    'frequency_label',
    'modulation_type',
    'transmitter_code',
    'transmitter_name',
    'receiver_code',
    'receiver_name',
  ]);
  // Per-parameter unit suffix applied to all non-identifier value columns.
  const parameterUnitSuffix: Partial<Record<ParameterName, string>> = {
    power: '(dBm)',
    command_threshold: '(dBm)',
  };
  // Per-parameter per-field unit override (takes precedence over the
  // blanket parameterUnitSuffix above). Use this when different columns
  // in the same table carry different units (e.g. Frequency table:
  // FBT* in MHz, tolerance in ppm).
  const parameterFieldUnits: Partial<Record<ParameterName, Record<string, string>>> = {
    frequency: {
      specification: '(MHz)',
      fbt: '(MHz)',
      fbt_hot: '(MHz)',
      fbt_cold: '(MHz)',
      tolerance: '(ppm)',
    },
    modulation_index: {
      tolerance: '(%)',
    },
    spurious: {
      specification: '(dBc)',
    },
  };

  let headerName = headerOverrides[key]
    ?? key.replaceAll('_', ' ').replace(/\b\w/g, (s) => s.toUpperCase());
  const fieldUnit = parameter ? parameterFieldUnits[parameter]?.[key] : undefined;
  const unit = fieldUnit ?? (parameter ? parameterUnitSuffix[parameter] : undefined);
  if (unit && !identifierKeys.has(key) && !headerName.includes('(')) {
    headerName = `${headerName} ${unit}`;
  }
  return {
    field: key,
    headerName,
    editable: !(metaKeys.has(key) || readOnlyKeys.has(key)),
    valueFormatter: (params) => {
      const v = params.value;
      if (Array.isArray(v) || (v && typeof v === 'object')) {
        return JSON.stringify(v);
      }
      return v as any;
    },
    valueParser: (params) => {
      const oldValue = params.oldValue;
      const newValue = params.newValue;

      if (Array.isArray(oldValue) || (oldValue && typeof oldValue === 'object')) {
        try {
          return JSON.parse(newValue);
        } catch {
          return oldValue;
        }
      }

      if (typeof oldValue === 'number') {
        const parsed = Number(newValue);
        return Number.isFinite(parsed) ? parsed : oldValue;
      }

      return newValue;
    },
    cellEditor: 'agTextCellEditor',
  };
}

function buildColumns(parameter: ParameterName, rows: Record<string, any>[]): GridColumnDef[] {
  const allKeys = new Set<string>();
  rows.forEach((r) => Object.keys(r).forEach((k) => allKeys.add(k)));

  const preferred = ['code', 'port', 'frequency_label', 'frequency'];

  if (parameter === 'command_threshold') {
    const commandThresholdOrdered = [
      ...preferred.filter((k) => allKeys.has(k)),
      ...['max_input_power', 'specification', 'tolerance', 'fbt', 'fbt_hot', 'fbt_cold'].filter((k) => allKeys.has(k)),
      ...[...allKeys].filter(
        (k) => !preferred.includes(k)
          && !metaKeys.has(k)
          && !hiddenGridKeys.has(k)
          && !['max_input_power', 'specification', 'tolerance', 'fbt', 'fbt_hot', 'fbt_cold'].includes(k),
      ),
    ];

    return commandThresholdOrdered.map((key) => {
      const col = createBaseColumn(key, parameter);
      if (key === 'max_input_power') {
        col.headerName = 'MaxInputPower (dBm)';
      }
      return col;
    });
  }

  if (parameter === 'modulation_index') {
    const toneCols = [...allKeys].filter((k) =>
      /^fbt(_hot|_cold)?_tone_/i.test(k),
    );

    const groupedToneKeys = new Map<string, { fbt?: string; fbtHot?: string; fbtCold?: string }>();
    for (const key of toneCols) {
      const lower = key.toLowerCase();
      if (lower.startsWith('fbt_hot_tone_')) {
        const tone = key.substring('fbt_hot_tone_'.length);
        groupedToneKeys.set(tone, { ...(groupedToneKeys.get(tone) ?? {}), fbtHot: key });
      } else if (lower.startsWith('fbt_cold_tone_')) {
        const tone = key.substring('fbt_cold_tone_'.length);
        groupedToneKeys.set(tone, { ...(groupedToneKeys.get(tone) ?? {}), fbtCold: key });
      } else if (lower.startsWith('fbt_tone_')) {
        const tone = key.substring('fbt_tone_'.length);
        groupedToneKeys.set(tone, { ...(groupedToneKeys.get(tone) ?? {}), fbt: key });
      }
    }

    const nonToneOrdered = [
      ...preferred.filter((k) => allKeys.has(k)),
      ...[...allKeys].filter(
        (k) => !preferred.includes(k) && !metaKeys.has(k) && !toneCols.includes(k) && !hiddenGridKeys.has(k),
      ),
    ];

    const toneGroups: ColGroupDef[] = [...groupedToneKeys.entries()]
      .sort((a, b) => Number(a[0]) - Number(b[0]))
      .map(([tone, fields]) => {
        const children: ColDef[] = [];
        if (fields.fbt) children.push({ ...createBaseColumn(fields.fbt, parameter), headerName: 'FBT' });
        if (fields.fbtHot) children.push({ ...createBaseColumn(fields.fbtHot, parameter), headerName: 'FBT Hot' });
        if (fields.fbtCold) children.push({ ...createBaseColumn(fields.fbtCold, parameter), headerName: 'FBT Cold' });

        return {
          headerName: `Tone ${tone} KHz`,
          children,
        };
      });

    return [...nonToneOrdered.map((k) => createBaseColumn(k, parameter)), ...toneGroups];
  }

  if (parameter === 'spurious') {
    const hiddenSpuriousKeys = new Set(['profile_name', 'profiles', 'tolerance', 'enable']);
    const ordered = [
      ...preferred.filter((k) => allKeys.has(k)),
      ...[...allKeys].filter((k) => !preferred.includes(k) && !metaKeys.has(k) && !hiddenSpuriousKeys.has(k) && !hiddenGridKeys.has(k)),
    ];

    const rendererByField: Record<string, any> = {
      fbt: spuriousFbtRenderer,
      fbt_hot: spuriousFbtHotRenderer,
      fbt_cold: spuriousFbtColdRenderer,
    };

    return ordered.map((key) => {
      if (!rendererByField[key]) return createBaseColumn(key, parameter);

      return {
        ...createBaseColumn(key, parameter),
        editable: true,
        minWidth: 180,
        width: 210,
        maxWidth: 240,
        suppressFillHandle: false,
        cellRenderer: rendererByField[key],
        cellRendererParams: { isEditable: true },
        valueSetter: (params) => {
          params.data[key] = params.newValue;
          return true;
        },
      } as ColDef;
    });
  }

  const ordered = [
    ...preferred.filter((k) => allKeys.has(k)),
    ...[...allKeys].filter((k) => !preferred.includes(k) && !metaKeys.has(k) && !hiddenGridKeys.has(k)),
  ];

  return ordered.map((key) => createBaseColumn(key, parameter));
}

function flattenRows(parameter: ParameterName, payloadRows: CatalogSpecRow[]): Record<string, any>[] {
  return payloadRows.map((item) => {
    const row = { ...(item.payload ?? {}) } as Record<string, any>;

    if (parameter === 'spurious') {
      const hasSpecification = !(row.specification === null || row.specification === undefined || row.specification === '');
      const hasTolerance = !(row.tolerance === null || row.tolerance === undefined || row.tolerance === '');

      // Backward-compatible fallback: old rows used tolerance for default -50.
      if (!hasSpecification && hasTolerance) {
        row.specification = row.tolerance;
      }
    }

    return {
      row_id: item.id,
      system_kind: item.system_kind,
      system_code: item.system_code,
      parameter_type: item.parameter_type,
      port_id: item.port_id,
      frequency_id: item.frequency_id,
      sort_order: item.sort_order,
      transmitter_code: item.system_code,
      transmitter_name: item.system_code,
      modulation_type: '',
      code: item.system_code,
      port: item.port_name,
      frequency_label: item.frequency_label,
      frequency: item.frequency_hz,
      ...row,
    };
  });
}

function buildReceiverCommandThresholdRows(existingRows: Record<string, any>[], receivers: any[], portsByCode: Record<string, any[]>, frequenciesByCode: Record<string, any[]>): Record<string, any>[] {
  const existingMap = new Map<string, Record<string, any>>();
  for (const row of existingRows) {
    const key = `${String(row.code ?? '')}|${String(row.port ?? '')}|${String(row.frequency_label ?? '')}|${String(row.frequency ?? '')}`;
    existingMap.set(key, row);
  }

  const out: Record<string, any>[] = [];

  for (const receiver of receivers) {
    const code = String(receiver?.code ?? '').trim();
    if (!code) continue;

    const ports = portsByCode[code] ?? [];
    const frequencies = frequenciesByCode[code] ?? [];

    for (const port of ports) {
      const portId = Number(port?.port_id ?? 0);
      const portName = String(port?.port_name ?? '').trim();
      if (!portId || !portName) continue;

      for (const frequency of frequencies) {
        const frequencyId = Number(frequency?.frequency_id ?? 0);
        const frequencyLabel = String(frequency?.frequency_label ?? '').trim();
        const frequencyHz = String(frequency?.frequency_hz ?? '').trim();
        if (!frequencyId || !frequencyLabel) continue;

        const key = `${code}|${portName}|${frequencyLabel}|${frequencyHz}`;
        const existing = existingMap.get(key);

        out.push({
          row_id: existing?.row_id,
          system_kind: 'receiver',
          system_code: code,
          parameter_type: 'command_threshold',
          port_id: portId,
          frequency_id: frequencyId,
          sort_order: existing?.sort_order ?? 0,
          transmitter_code: code,
          transmitter_name: code,
          modulation_type: 'PSK_FM',
          code,
          port: portName,
          frequency_label: frequencyLabel,
          frequency: frequencyHz,
          max_input_power: existing?.max_input_power ?? -60,
          specification: existing?.specification ?? null,
          tolerance: existing?.tolerance ?? 0.5,
          fbt: existing?.fbt ?? null,
          fbt_hot: existing?.fbt_hot ?? null,
          fbt_cold: existing?.fbt_cold ?? null,
        });
      }
    }
  }

  return out;
}

/**
 * Extract per-transmitter ranging tones from each transmitter's
 * `modulation_details.sub_carriers` definition. Returns a map of
 * transmitter code → list of tone strings (in kHz, trimmed). The tone set
 * comes from the transmitter itself (typically 32 kHz / 128 kHz for
 * PSK_PM systems) — NOT from the global `/system-catalog/ranging-tones`
 * catalog (which is for ranging-threshold tests, e.g. 22 / 27.777 kHz).
 */
function extractTonesByCode(transmitters: any[]): Record<string, string[]> {
  const out: Record<string, string[]> = {};
  for (const tx of transmitters) {
    const code = String(tx?.code ?? '').trim();
    if (!code) continue;
    const details = tx?.modulation_details ?? {};
    const subCarriers: any[] = Array.isArray(details?.sub_carriers) ? details.sub_carriers : [];
    const tones: string[] = [];
    const seen = new Set<string>();
    for (const entry of subCarriers) {
      const raw = Array.isArray(entry) ? entry[0] : entry;
      if (raw === null || raw === undefined) continue;
      const text = String(raw).trim();
      if (text === '' || seen.has(text)) continue;
      const num = Number(text);
      if (!Number.isFinite(num) || num <= 0) continue;
      seen.add(text);
      tones.push(text);
    }
    out[code] = tones;
  }
  return out;
}

function buildTransmitterSpecRows(
  parameter: ParameterName,
  existingRows: Record<string, any>[],
  transmitters: any[],
  portsByCode: Record<string, any[]>,
  frequenciesByCode: Record<string, any[]>,
  tonesByCode: Record<string, string[]> = {},
): Record<string, any>[] {
  const existingMap = new Map<string, Record<string, any>>();
  for (const row of existingRows) {
    const key = `${String(row.code ?? '')}|${String(row.port ?? '')}|${String(row.frequency_label ?? '')}|${String(row.frequency ?? '')}`;
    existingMap.set(key, row);
  }

  const out: Record<string, any>[] = [];

  for (const transmitter of transmitters) {
    const code = String(transmitter?.code ?? '').trim();
    if (!code) continue;
    const modulationType = String(transmitter?.modulation_type ?? '').trim();

    const ports = portsByCode[code] ?? [];
    const frequencies = frequenciesByCode[code] ?? [];

    for (const port of ports) {
      const portId = Number(port?.port_id ?? 0);
      const portName = String(port?.port_name ?? '').trim();
      if (!portId || !portName) continue;

      for (const frequency of frequencies) {
        const frequencyId = Number(frequency?.frequency_id ?? 0);
        const frequencyLabel = String(frequency?.frequency_label ?? '').trim();
        const frequencyHz = String(frequency?.frequency_hz ?? '').trim();
        if (!frequencyId || !frequencyLabel) continue;

        const key = `${code}|${portName}|${frequencyLabel}|${frequencyHz}`;
        const existing = existingMap.get(key);

        const defaults: Record<string, any> = {
          row_id: existing?.row_id,
          system_kind: 'transmitter',
          system_code: code,
          parameter_type: parameter,
          port_id: portId,
          frequency_id: frequencyId,
          sort_order: existing?.sort_order ?? 0,
          transmitter_code: code,
          transmitter_name: code,
          modulation_type: modulationType,
          code,
          port: portName,
          frequency_label: frequencyLabel,
          frequency: frequencyHz,
        };

        if (parameter === 'power') {
          Object.assign(defaults, {
            specification: existing?.specification ?? null,
            tolerance: existing?.tolerance ?? null,
            fbt: existing?.fbt ?? null,
            fbt_hot: existing?.fbt_hot ?? null,
            fbt_cold: existing?.fbt_cold ?? null,
          });
        } else if (parameter === 'frequency') {
          Object.assign(defaults, {
            tolerance: existing?.tolerance ?? null,
            fbt: existing?.fbt ?? null,
            fbt_hot: existing?.fbt_hot ?? null,
            fbt_cold: existing?.fbt_cold ?? null,
          });
        } else if (parameter === 'modulation_index') {
          Object.assign(defaults, {
            specification: existing?.specification ?? 0.9,
            tolerance: existing?.tolerance ?? 20,
          });
          // Pre-populate dynamic tone columns so the grid always shows
          // FBT / FBT Hot / FBT Cold groups even when no rows have been
          // saved yet. Tones come from each transmitter's own
          // modulation_details.sub_carriers (NOT the global ranging-tones
          // catalog). Existing values are preserved (merged below).
          const txTones = tonesByCode[code] ?? [];
          for (const tone of txTones) {
            const t = String(tone ?? '').trim();
            if (!t) continue;
            const fbtKey = `fbt_tone_${t}`;
            const fbtHotKey = `fbt_hot_tone_${t}`;
            const fbtColdKey = `fbt_cold_tone_${t}`;
            defaults[fbtKey] = existing?.[fbtKey] ?? null;
            defaults[fbtHotKey] = existing?.[fbtHotKey] ?? null;
            defaults[fbtColdKey] = existing?.[fbtColdKey] ?? null;
          }
        } else if (parameter === 'spurious') {
          Object.assign(defaults, {
            specification: existing?.specification ?? null,
            tolerance: existing?.tolerance ?? null,
            fbt: existing?.fbt ?? null,
            fbt_hot: existing?.fbt_hot ?? null,
            fbt_cold: existing?.fbt_cold ?? null,
          });
        }

        // Preserve any extra fields already saved on the existing row
        // (e.g. dynamic tone columns for modulation_index).
        if (existing) {
          for (const k of Object.keys(existing)) {
            if (!(k in defaults)) defaults[k] = existing[k];
          }
        }

        out.push(defaults);
      }
    }
  }

  return out;
}

async function loadParameter(parameter: ParameterName) {
  const res = parameter === 'command_threshold'
    ? await api.getSystemCatalogReceiverSpecRows(parameter)
    : await api.getSystemCatalogTransmitterSpecRows(parameter);
  if (res.error.value) {
    toast.add({
      severity: 'error',
      summary: 'Load Failed',
      detail: `Unable to load ${sectionTitles[parameter]} rows.`,
      life: 3500,
    });
    return;
  }

  let rows = flattenRows(parameter, (res.data.value as CatalogSpecRow[]) ?? []);

  if (parameter === 'command_threshold') {
    const receiversRes = await api.getReceivers();
    const receivers = Array.isArray(receiversRes.data.value) ? (receiversRes.data.value as any[]) : [];
    const portsByCode: Record<string, any[]> = {};
    const frequenciesByCode: Record<string, any[]> = {};

    await Promise.all(receivers.map(async (receiver) => {
      const code = String(receiver?.code ?? '').trim();
      if (!code) return;
      const [portsRes, frequenciesRes] = await Promise.all([
        api.getSystemCatalogSystemPorts('receiver', code),
        api.getSystemCatalogSystemFrequencies('receiver', code),
      ]);
      portsByCode[code] = Array.isArray(portsRes.data.value) ? (portsRes.data.value as any[]) : [];
      frequenciesByCode[code] = Array.isArray(frequenciesRes.data.value) ? (frequenciesRes.data.value as any[]) : [];
    }));

    rows = buildReceiverCommandThresholdRows(rows, receivers, portsByCode, frequenciesByCode);
  } else {
    // For transmitter-side parameters synthesize rows for every
    // (transmitter × port × frequency) combo so the grid is never blank
    // when no spec rows have been saved yet. Existing saved rows are merged
    // by (code, port, frequency_label, frequency).
    const transmittersRes = await api.getTransmitters();
    const transmittersRaw = transmittersRes.data.value as any;
    const transmitters: any[] = Array.isArray(transmittersRaw)
      ? transmittersRaw
      : Array.isArray(transmittersRaw?.transmitters)
        ? transmittersRaw.transmitters
        : [];

    if (transmitters.length > 0) {
      const portsByCode: Record<string, any[]> = {};
      const frequenciesByCode: Record<string, any[]> = {};

      await Promise.all(transmitters.map(async (transmitter) => {
        const code = String(transmitter?.code ?? '').trim();
        if (!code) return;
        const [portsRes, frequenciesRes] = await Promise.all([
          api.getSystemCatalogSystemPorts('transmitter', code),
          api.getSystemCatalogSystemFrequencies('transmitter', code),
        ]);
        portsByCode[code] = Array.isArray(portsRes.data.value) ? (portsRes.data.value as any[]) : [];
        frequenciesByCode[code] = Array.isArray(frequenciesRes.data.value) ? (frequenciesRes.data.value as any[]) : [];
      }));

      // For modulation_index the tone-group columns are dynamic. Each
      // transmitter carries its own tone set in
      // `modulation_details.sub_carriers` — use that (NOT the global
      // /system-catalog/ranging-tones list which is for ranging tests).
      let tonesByCode: Record<string, string[]> = {};
      if (parameter === 'modulation_index') {
        tonesByCode = extractTonesByCode(transmitters);
      }

      const synthesized = buildTransmitterSpecRows(parameter, rows, transmitters, portsByCode, frequenciesByCode, tonesByCode);
      if (synthesized.length > 0) {
        rows = synthesized;
      }
    }

    // Final fallback: if the system_catalog tables aren't seeded with ports/
    // frequencies for transmitters, build rows from each transmitter's own
    // modulation_details via the legacy parameter-rows endpoint. This
    // mirrors the working approach used by other transmitter panels.
    if (rows.length === 0) {
      const paramRes = await api.getParameterRows(parameter);
      const paramData = paramRes.data.value as any;
      const paramRows: any[] = Array.isArray(paramData?.rows) ? paramData.rows : [];
      rows = paramRows.map((entry) => {
        const inner = (entry?.row ?? {}) as Record<string, any>;
        return {
          system_kind: 'transmitter',
          system_code: entry?.transmitter_code,
          parameter_type: parameter,
          transmitter_code: entry?.transmitter_code,
          transmitter_name: entry?.transmitter_name ?? entry?.transmitter_code,
          modulation_type: entry?.modulation_type ?? '',
          code: inner.code ?? entry?.transmitter_code,
          port: inner.port,
          frequency_label: inner.frequency_label,
          frequency: inner.frequency,
          ...inner,
        } as Record<string, any>;
      });

      // Legacy data path may not include any fbt_tone_* keys yet — pad
      // each row with the tones from each transmitter's own
      // modulation_details.sub_carriers so FBT tone-group columns always
      // render in the modulation_index grid.
      if (parameter === 'modulation_index' && rows.length > 0) {
        const transmittersRes2 = await api.getTransmitters();
        const transmittersRaw2 = transmittersRes2.data.value as any;
        const transmitters2: any[] = Array.isArray(transmittersRaw2)
          ? transmittersRaw2
          : Array.isArray(transmittersRaw2?.transmitters)
            ? transmittersRaw2.transmitters
            : [];
        const tonesByCode = extractTonesByCode(transmitters2);
        for (const row of rows) {
          const txCode = String(row.transmitter_code ?? row.code ?? '').trim();
          const tones = tonesByCode[txCode] ?? [];
          for (const tone of tones) {
            const fbtKey = `fbt_tone_${tone}`;
            const fbtHotKey = `fbt_hot_tone_${tone}`;
            const fbtColdKey = `fbt_cold_tone_${tone}`;
            if (!(fbtKey in row)) row[fbtKey] = null;
            if (!(fbtHotKey in row)) row[fbtHotKey] = null;
            if (!(fbtColdKey in row)) row[fbtColdKey] = null;
          }
        }
      }
    }
  }

  tableRows[parameter] = rows;
  columnDefsByParameter[parameter] = buildColumns(parameter, rows);
  gridApis.value[parameter]?.setGridOption('columnDefs', columnDefsByParameter[parameter]);
  gridApis.value[parameter]?.setGridOption('rowData', tableRows[parameter]);

  // Auto-size after rows are rendered.
  setTimeout(() => autoSizeDisplayedColumns(parameter), 0);
}

async function saveParameter(parameter: ParameterName) {
  try {
    saving[parameter] = true;
    const gridRows: Record<string, any>[] = [];
    gridApis.value[parameter]?.forEachNode((n) => {
      if (n.data) gridRows.push(n.data as Record<string, any>);
    });

    const rowsToSave = gridRows.filter((r) => String(r.transmitter_code ?? '').trim() !== '');

    const saveResults = await Promise.all(
      rowsToSave.map(async (r, index) => {
        const payload: Record<string, any> = {};
        Object.keys(r).forEach((k) => {
          if (!metaKeys.has(k) && !structuralKeys.has(k)) payload[k] = r[k];
        });

        if (parameter === 'command_threshold') {
          return api.upsertSystemCatalogReceiverSpecRow(String(r.transmitter_code), {
            parameter_type: parameter,
            port_id: Number(r.port_id),
            frequency_id: Number(r.frequency_id),
            payload,
            sort_order: index,
          });
        }

        return api.upsertSystemCatalogTransmitterSpecRow(String(r.transmitter_code), {
          parameter_type: parameter,
          port_id: Number(r.port_id),
          frequency_id: Number(r.frequency_id),
          payload,
          sort_order: index,
        });
      }),
    );

    if (saveResults.some((result) => Boolean(result.error.value))) {
      toast.add({
        severity: 'error',
        summary: 'Save Failed',
        detail: `Unable to save ${sectionTitles[parameter]} rows.`,
        life: 3500,
      });
      return;
    }

    toast.add({
      severity: 'success',
      summary: 'Saved',
      detail: `${sectionTitles[parameter]} updated (${rowsToSave.length} rows).`,
      life: 3000,
    });

    await loadParameter(parameter);
  } finally {
    saving[parameter] = false;
  }
}

function onGridReady(parameter: ParameterName, event: GridReadyEvent) {
  gridApis.value[parameter] = event.api;
  setTimeout(() => autoSizeDisplayedColumns(parameter), 0);
  // Each parameter has its own grid; key by parameter name so layouts persist
  // independently per spec table.
  ui.registerGrid(parameter);
  ui.onGridReady(parameter, event);
}

function onGridReadyParameter(parameter: ParameterName, event: GridReadyEvent) {
  onGridReady(parameter, event);
}

function autoSizeDisplayedColumns(parameter: ParameterName) {
  const gridApi = gridApis.value[parameter];
  if (!gridApi) return;

  const displayed = gridApi.getAllDisplayedColumns();
  const colIds = displayed.map((c) => c.getColId());
  if (colIds.length > 0) {
    gridApi.autoSizeColumns(colIds, false);
  }
}

function onFirstDataRenderedParameter(parameter: ParameterName, _event: FirstDataRenderedEvent) {
  setTimeout(() => autoSizeDisplayedColumns(parameter), 0);
}

function onModelUpdatedParameter(parameter: ParameterName, _event: ModelUpdatedEvent) {
  setTimeout(() => autoSizeDisplayedColumns(parameter), 0);
}

function onCellDoubleClickedParameter(parameter: ParameterName, event: CellClickedEvent) {
  if (parameter !== 'spurious') return;

  const field = String(event.colDef?.field ?? '');
  if (!isFbtField(field)) return;

  const target = event.event?.target as HTMLElement | null;
  if (target?.closest?.('.ag-fill-handle')) return;

  activeFbtField.value = field;
  activeFbtNode.value = event.node;
  fbtEditingData.value = ensureFbtMatrix(event.value).map(row => [...row]);
  showFbtDialog.value = true;
}

function onCellKeyDownParameter(parameter: ParameterName, event: any) {
  if (parameter !== 'spurious') return;

  const keyboardEvent = event?.event as KeyboardEvent | undefined;
  if (!keyboardEvent) return;

  const field = event?.colDef?.field;
  const targets = getSelectedFbtTargets(event.api, event);
  const isFbtTarget = isFbtField(field) || targets.length > 0;
  if (!isFbtTarget) return;

  const key = keyboardEvent.key.toLowerCase();
  const isCopy = (keyboardEvent.ctrlKey || keyboardEvent.metaKey) && key === 'c';
  const isPaste = (keyboardEvent.ctrlKey || keyboardEvent.metaKey) && key === 'v';
  const isDelete = key === 'delete' || key === 'backspace';

  if (isCopy) {
    event.api?.copySelectedRangeToClipboard?.();
    keyboardEvent.preventDefault();
    keyboardEvent.stopPropagation();
    return;
  }

  if (isPaste) {
    event.api?.pasteFromClipboard?.();
    keyboardEvent.preventDefault();
    keyboardEvent.stopPropagation();
    return;
  }

  if (isDelete) {
    for (const target of targets) {
      target.rowNode?.setDataValue?.(target.field, [['', '']]);
    }
    event.api?.resetRowHeights?.();
    keyboardEvent.preventDefault();
    keyboardEvent.stopPropagation();
  }
}

function ensureFbtMatrix(value: unknown): FbtMatrix {
  if (Array.isArray(value) && value.length > 0 && value.every((row) => Array.isArray(row))) {
    return value as FbtMatrix;
  }
  return [['', '']];
}

function closeFbtEditor() {
  showFbtDialog.value = false;
}

async function saveFbtEditor() {
  const cleanedData = fbtEditingData.value.filter((row) =>
    row.some((cell) => cell !== '' && cell !== null && cell !== undefined),
  );
  const finalData = cleanedData.length > 0 ? cleanedData : [['', '']];

  if (activeFbtField.value && activeFbtNode.value) {
    activeFbtNode.value.setDataValue(activeFbtField.value, finalData);
    await nextTick();
    activeFbtNode.value?.gridApi?.resetRowHeights?.();
  }

  showFbtDialog.value = false;
}

function getRowHeight(parameter: ParameterName, params: RowHeightParams): number {
  if (parameter !== 'spurious') return 42;

  const row = (params.data ?? {}) as Record<string, any>;
  const rowCount = (value: unknown): number => (Array.isArray(value) ? Math.max(value.length, 1) : 1);
  const maxRows = Math.max(
    rowCount(row.fbt),
    rowCount(row.fbt_hot),
    rowCount(row.fbt_cold),
  );

  // Grow with row count but cap height; preview itself can scroll for long lists.
  const previewRows = maxRows;
  const headerHeight = 20;
  const tableRowHeight = 12;
  const containerPadding = 1;
  const baseHeight = containerPadding + headerHeight + (previewRows * tableRowHeight);
  return Math.max(45, baseHeight);
}

onMounted(async () => {
  for (const parameter of visibleParameters.value) {
    await loadParameter(parameter);
  }
  await ui.load();
});

watch(
  () => props.activeParameter,
  async () => {
    for (const parameter of visibleParameters.value) {
      await loadParameter(parameter);
    }
  },
);
</script>

<style scoped>
.specs-panel {
  padding: 1rem;
}

.specs-panel.single-view {
  height: calc(100vh - 2rem);
  display: flex;
  flex-direction: column;
}

.specs-header {
  margin-bottom: 1rem;
}

.specs-header h2 {
  margin: 0;
  color: #22d3ee;
}

.specs-header p {
  margin: 0.35rem 0 0;
  color: #94a3b8;
}

.spec-section {
  border: 1px solid #1e3a5f;
  border-radius: 8px;
  padding: 0.75rem;
  margin-bottom: 1rem;
  background: #0b182a;
}

.spec-section.fill-height {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

.spec-section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 0.65rem;
}

.spec-section-header h3 {
  margin: 0;
  color: #e2e8f0;
}

.spec-section-header small {
  color: #94a3b8;
}

.actions {
  display: flex;
  gap: 0.5rem;
}

.spec-grid {
  min-height: 0;
}
</style>
