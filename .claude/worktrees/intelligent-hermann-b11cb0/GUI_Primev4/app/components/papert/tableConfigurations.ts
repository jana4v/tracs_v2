import { ref } from 'vue';
import CellStyles from '@/components/AgGrid/CellStyles/Validators';
import ColourPickerCellEditor from '@/components/AgGrid/CellEditors/ColourPickerCellEditor.vue';
import DropDownCellEditor from '@/components/AgGrid/CellEditors/DropDownCellEditor.vue';
import MultiSelectDropDownCellEditor from '@/components/AgGrid/CellEditors/MultiSelectDropDownCellEditor.vue';
import TransmitterDetails from '@/components/AgGrid/CellEditors/TransmitterDetails.vue';
import multiLineTextRenderer from '@/components/AgGrid/CellRenderers/multiLineTextRenderer.vue';
import basic from '@/components/AgGrid/CellToolTips/basic.vue';
import HandsonTableCellEditor from '@/components/AgGrid/CellEditors/HandsonTableCellEditor.vue';
import HandsonTableCellRenderer from '@/components/AgGrid/CellRenderers/HandsonTableCellRenderer.vue';
import TriStateCheckboxRenderer from '@/components/AgGrid/CellRenderers/TriStateCheckboxRenderer.vue';
import ColourPickerCellRenderer from '@/components/AgGrid/CellRenderers/ColourPickerCellRenderer.vue';

interface TableColumnDef {
    [key: string]: any;
}

interface ToolBarType {
    [key: string]: boolean;
}

const defaultColConfig: Record<string, TableColumnDef> = {};
const tableConfig: Record<string, TableColumnDef[]> = {};
const ToolBarDisplays: Record<string, ToolBarType> = {};

defaultColConfig['test_phase_colors'] = {
    sortable: true,
    filter: true,
    flex: 1,
    cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_phase_colors'] = {
    is_addRowsAllowed: true,
    is_removeRowsAllowed: true,
    is_saveAllowed: true
};

tableConfig['test_phase_colors'] = [
    {
        headerName: 'Test Phase',
        field: 'TEST_PHASE',
        flex: 2,
        editable: false
    },
    {
        headerName: 'Label',
        field: 'LABEL',
        flex: 1,
        editable: true
    },
    {
        headerName: 'Color',
        field: 'COLOR',
        flex: 1,
        editable: true
    },
    {
        headerName: 'Color',
        field: 'COLOR',
        flex: 1,
        editable: true,
        cellEditor: ColourPickerCellEditor,
        cellRenderer: ColourPickerCellRenderer,
        cellEditorPopup: false,
    }
];

defaultColConfig['config_groups'] = {
    sortable: true,
    filter: true,
    flex: 1,
    cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['config_groups'] = {
    is_addRowsAllowed: true,
    is_removeRowsAllowed: true,
    is_saveAllowed: true
};

tableConfig['config_groups'] = [
    {
        headerName: 'Group Name',
        field: 'GROUP_NAME',
        flex: 1,
        editable: true
    },
    {
        headerName: 'CFG Numbers',
        field: 'CONFIG_NUMBERS',
        flex: 1,
        editable: true
    }
];

defaultColConfig['report_config'] = {
    sortable: true,
    filter: true,
    flex: 1,
    cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['report_config'] = {
    is_addRowsAllowed: true,
    is_removeRowsAllowed: true,
    is_saveAllowed: true
};

tableConfig['report_config'] = [
    {
        headerName: 'Test Plan',
        field: 'REPORT_CONFIG_NAME',
        flex: 1,
        editable: true
    },
    {
        headerName: 'Group Name',
        field: 'GROUP_NAME',
        flex: 1,
        editable: true,
        cellEditor: DropDownCellEditor,
        cellEditorPopup: true,
        cellEditorParams: { url: "/papert/get_cfg_group_names", placeholder: 'Select Code' },
               
    },
    {
        headerName: 'Group Priority',
        field: 'GROUP_PRIORITY',
        flex: 1,
        editable: true
    },
    {
        headerName: 'Test Parameters',
        field: 'TEST_PARAMETERS',
        flex: 1,
        editable: true
    },
    {
        headerName: 'CFGs Per Sheet',
        field: 'CFGS_PER_SHEET',
        flex: 1,
        editable: true
    },
    {
        headerName: 'Trend Plot Layout',
        field: 'TREND_PLOT_LAYOUT',
        flex: 1,
        editable: true
    }
];


defaultColConfig['param_table'] = {
  sortable: true,
  filter: true,
  flex: 1,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['param_table'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true
};


//columns PARAMETER	SUB_SYSTEM	TITLE	X_LABEL	Y_LABEL	AXIS_MAX	AXIS_MIN	MAJOR_GRID_LINES	MINOR_GRID_LINES	SPEC_LOW	SPEC_HIGH	DATA	UNCERTINAITY	PLOT_TYPE	MULTI_VALUED

tableConfig['param_table'] = [
  {
    headerName: 'PARAMETER',
    field: 'PARAMETER',
    flex: 1,
    editable: false,
    hide:true
  },

  {
    headerName: 'PARAMETER',
    field: 'DISPLAY_LABEL',
    flex: 1,
    editable: false
  },

  {
    headerName: 'ENABLE',
    field: 'ENABLE',
    flex: 1,
    editable: false,
    hide:true
  },

  {
    headerName: 'SUB_SYSTEM',
    field: 'SUB_SYSTEM',
    flex: 1,
    editable: false
  },
  {
    headerName: 'TITLE',
    field: 'TITLE',
    flex: 1,
    editable: true
  },
  {
    headerName: 'X_LABEL',
    field: 'X_LABEL',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Y_LABEL',
    field: 'Y_LABEL',
    flex: 1,
    editable: true
  },
  {
    headerName: 'AXIS_MAX',
    field: 'AXIS_MAX',
    flex: 1,
    editable: true
  },
  {
    headerName: 'AXIS_MIN',
    field: 'AXIS_MIN',
    flex: 1,
    editable: true
  },
  {
    headerName: 'MAJOR_GRID_LINES',
    field: 'MAJOR_GRID_LINES',
    flex: 1,
    editable: true
  },
  {
    headerName: 'MINOR_GRID_LINES',
    field: 'MINOR_GRID_LINES',
    flex: 1,
    editable: true
  },
  {
    headerName: 'SPEC_LOW',
    field: 'SPEC_LOW',
    flex: 1,
    editable: false,
    hide: true
  },
  {
    headerName: 'SPEC_HIGH',
    field: 'SPEC_HIGH',
    flex: 1,
    editable: false,
    hide: true
  },
  {
    headerName: 'DATA',
    field: 'DATA',
    flex: 1,
    editable: false,
    hide: true
  },
  {
    headerName: 'UNCERTINAITY',
    field: 'UNCERTINAITY',
    flex: 1,
    editable: true
  },
  {
    headerName: 'PLOT_TYPE',
    field: 'PLOT_TYPE',
    flex: 1,
    editable: false,
    hide: true
  },
  {
    headerName: 'MULTI_VALUED',
    field: 'MULTI_VALUED',
    flex: 1,
    editable: false,
    hide: true
  }

];


export function papert_table_config_func(table_name: string) {
    const ColDefs = tableConfig[table_name];
    const DefaultDef = defaultColConfig[table_name];
    const ToolBarConf = ToolBarDisplays[table_name];
    return { DefaultDef, ColDefs, ToolBarConf };
}
//export { defaultColConfig, allow_add_remove_rows, getTableConfig }
