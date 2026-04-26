import { ref } from 'vue';
import CellStyles from '@/components/AgGrid/CellStyles/Validators';
interface TableColumnDef {
  [key: string]: any;
}

interface ToolBarType {
  [key: string]: boolean;
}

const defaultColConfig = ref<Record<string, TableColumnDef>>({});

defaultColConfig.value['db#transmitter'] = {
  sortable: true,
  filter: true,
  flex: 1,
};

const tableConfig = ref<Record<string, TableColumnDef[]>>({});

tableConfig.value['telemetry#live'] = [
  {
    headerName: 'PID',
    field: 'pid',
    flex: 1,
    editable: false,
  },
  {
    headerName: 'Mnemonic',
    field: 'mnemonic',
    flex: 2,
    editable: false,
  },
  {
    headerName: 'Value',
    field: 'value',
    flex: 2,
    editable: false,
  },
];

defaultColConfig.value['telemetry#database'] = {
  sortable: true,
  filter: true,
  flex: 1,
};

tableConfig.value['telemetry#database'] = [
  {
    headerName: 'SubSytem',
    field: 'sub_system',
    editable: false,
    hide: true,
  },
  {
    headerName: 'PID',
    field: 'pid',
    flex: 1,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(1),
    },
  },
  {
    headerName: 'Mnemonic',
    field: 'mnemonic',
    flex: 2,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(6),
    },
  },
  {
    headerName: 'Possible Values',
    field: 'possible_values',
    flex: 3,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.telemetry_possible_values,
    },
  },
  {
    headerName: 'Allowed Values',
    field: 'allowed_values',
    flex: 3,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.telemetry_allowed_values,
    },
  },
  {
    headerName: 'Ignore Changes',
    field: 'ignore_change',
    flex: 3,
    editable: true,
    cellRenderer: 'agCheckboxCellRenderer',
    
    },
  
];

defaultColConfig.value['telemetry#simulate'] = {
  sortable: true,
  filter: true,
  flex: 1,
};

tableConfig.value['telemetry#simulate'] = [
  {
    headerName: 'SubSytem',
    field: 'sub_system',
    hide: true,
    editable: false,
  },
  {
    headerName: 'PID',
    field: 'pid',
    flex: 1,
    editable: false,
  },
  {
    headerName: 'Mnemonic',
    field: 'mnemonic',
    flex: 2,
    editable: false,
  },
  {
    headerName: 'Possible Values',
    field: 'possible_values',
    flex: 3,
    editable: false,
  },

  {
    headerName: 'Value(s) to be Simulated',
    field: 'value_to_be_simulated',
    flex: 3,
    editable: true,
  },
  {
    headerName: 'Simulation Mode',
    field: 'simulation_mode',
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: { values: ['Fixed', 'Random', 'Monotonic'] },
    flex: 1,
    editable: true,
  },
];

const ToolBarDisplays = ref<Record<string, ToolBarType>>({});
ToolBarDisplays.value['telemetry#simulate'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};

ToolBarDisplays.value['telemetry#database'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};

ToolBarDisplays.value['telemetry#live'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

export function cmacs_table_config_func(group_table_name: string) {
  const ColDefs = tableConfig.value[group_table_name];
  const DefaultDef = defaultColConfig.value[group_table_name];
  const ToolBarConf = ToolBarDisplays.value[group_table_name];
  return { DefaultDef, ColDefs, ToolBarConf };
}
//export { defaultColConfig, allow_add_remove_rows, getTableConfig };
