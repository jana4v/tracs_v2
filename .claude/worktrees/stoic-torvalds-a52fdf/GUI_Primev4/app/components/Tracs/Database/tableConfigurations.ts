import { ref } from 'vue';
import CellStyles from '@/components/AgGrid/CellStyles/Validators';
import DropDownCellEditor from '@/components/AgGrid/CellEditors/DropDownCellEditor.vue';
import MultiSelectDropDownCellEditor from '@/components/AgGrid/CellEditors/MultiSelectDropDownCellEditor.vue';
import TransmitterDetails from '@/components/AgGrid/CellEditors/TransmitterDetails.vue';
import multiLineTextRenderer from '@/components/AgGrid/CellRenderers/multiLineTextRenderer.vue';
import basic from '@/components/AgGrid/CellToolTips/basic.vue';
import HandsonTableCellEditor from '@/components/AgGrid/CellEditors/HandsonTableCellEditor.vue';
import HandsonTableCellRenderer from '@/components/AgGrid/CellRenderers/HandsonTableCellRenderer.vue';
import TriStateCheckboxRenderer from '@/components/AgGrid/CellRenderers/TriStateCheckboxRenderer.vue';

interface TableColumnDef {
  [key: string]: any;
}

interface ToolBarType {
  [key: string]: boolean;
}


const defaultColConfig: Record<string, TableColumnDef> = {};
const tableConfig: Record<string, TableColumnDef[]> = {};
const ToolBarDisplays: Record<string, ToolBarType> = {};


export const table_details = [
  { "Systems": ["Transmitter", "Receiver", "Transponder"] },
  { "Configurations": ["Transmitter", "Receiver", "Transponder"] },
  { "Specifications": { "Transmitter": ["Power", "Frequency", "Modulation Index", "Spurious"], "Receiver": ["Command Threshold"], "Transponder": ["Ranging Threshold"] } },
  { "Test Systems": ["Instrument Address", "Power Meter Channels", "TSM Paths"] },
  { "Test Profiles": ["Spurious Bands", "Command Threshold", "Ranging Threshold"] },
  { "On Board Losses": ["Up Link", "Down Link"]},
  { "Calibration": ["Up Link", "Down Link","TVAC Ref. Cable Details"] },
  { "Test Plan": ["Plan Names", "Transmitter", "Receiver", "Transponder"] },
  { "TM TC": ["Telemetry", "Tele Commands", "CFG Based TC"] },
  { "ENV Data": ["ENV Data"] },
]



defaultColConfig['systems#transponder'] = {
  sortable: true,
  filter: true,
  flex: 1,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['systems#transponder'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};


tableConfig['systems#transponder'] = [
  {
    headerName: 'Name',
    field: 'name',
    flex: 1,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(3) && CellStyles.unique_value_in_column('name'),
    },
    tooltipField: 'name',
    tooltipComponent: basic,

  },
  {
    headerName: 'Code',
    field: 'code',
    flex: 1,
    editable: true,
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(3) && CellStyles.unique_value_in_column('code'),
    },
    tooltipField: 'code',
    tooltipComponent: basic,
  },
  {
    headerName: 'Receiver Code',
    field: 'receiver_code',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { url: "tracs/getReceiverCodes", placeholder: 'Select Code' },
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(3) && CellStyles.unique_value_in_column('receiver_code'),
    },
    tooltipField: 'receiver_code',
    tooltipComponent: basic,
  },
  {
    headerName: 'Transmitter Code',
    field: 'transmitter_code',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { url: "tracs/getTransmitterCodes", placeholder: 'Select Code' },
    cellClassRules: {
      table_cell_error: CellStyles.atlest_n_characters(3) && CellStyles.unique_value_in_column('transmitter_code'),
    },
    tooltipField: 'transmitter_code',
    tooltipComponent: basic,
  },


];

defaultColConfig['configurations#transmitter'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }

};

ToolBarDisplays['configurations#transmitter'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};


tableConfig['configurations#transmitter'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   width: 100,
  //   valueGetter: "node.rowIndex + 1",
  //   enableRowGroup:false,
  // },

  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: true
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },


];

defaultColConfig['configurations#receiver'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['configurations#receiver'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};


tableConfig['configurations#receiver'] = [
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: true
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },


];

defaultColConfig['configurations#transponder'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['configurations#transponder'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};


tableConfig['configurations#transponder'] = [
  {
    headerName: 'Code',
    field: 'code',
    flex: 1,
  },
  {
    headerName: 'Up Link',
    field: 'mapping_details.up_link_config',
    flex: 1,
  },
  {
    headerName: 'Down Link',
    field: 'mapping_details.down_link_config',
    flex: 1,
  },
];



defaultColConfig['specifications#transmitter#power'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#transmitter#power'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['specifications#transmitter#power'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Specification(dBm)',
    field: 'specification',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Tolerance(dB)',
    field: 'tolerance',
    flex: 1,
    editable: true
  },

  {
    headerName: 'FBT(dBm)',
    field: 'FBT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_COLD(dBm)',
    field: 'FBT_COLD',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_HOT(dBm)',
    field: 'FBT_HOT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT(dBm)',
    field: 'FBT_VACCUM_AMBIENT',
    flex: 1,
    editable: true
  },

];



defaultColConfig['specifications#transmitter#frequency'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#transmitter#frequency'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['specifications#transmitter#frequency'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: true,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Tolerance(ppm)',
    field: 'tolerance',
    flex: 1,
    editable: true
  },


  {
    headerName: 'Specification',
    field: 'specification',
    flex: 1,
    hide: true,
    editable: true
  },
  {
    headerName: 'FBT(MHz)',
    field: 'FBT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_COLD(MHz)',
    field: 'FBT_COLD',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_HOT(MHz)',
    field: 'FBT_HOT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT(MHz)',
    field: 'FBT_VACCUM_AMBIENT',
    flex: 1,
    editable: true
  },

];



defaultColConfig['specifications#transmitter#modulation_index'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#transmitter#modulation_index'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['specifications#transmitter#modulation_index'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: true,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Tolerance(%)',
    field: 'tolerance',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Specification(rad)',
    field: 'specification',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT(rad)',
    flex: 1,
    children: [
      {
        headerName: "SC1", field: 'FBT_SC1', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
      {
        headerName: "SC2", field: 'FBT_SC2', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
    ]
  },
  {
    headerName: 'FBT_COLD(rad)',
    flex: 1,
    children: [
      {
        headerName: "SC1", field: 'FBT_COLD_SC1', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
      {
        headerName: "SC2", field: 'FBT_COLD_SC2', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
    ]
  },
  {
    headerName: 'FBT_HOT(rad)',
    flex: 1,
    children: [
      {
        headerName: "SC1", field: 'FBT_HOT_SC1', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
      {
        headerName: "SC2", field: 'FBT_HOT_SC2', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
    ]
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT(rad)',
    flex: 1,
    children: [
      {
        headerName: "SC1", field: 'FBT_VACCUM_AMBIENT_SC1', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
      {
        headerName: "SC2", field: 'FBT_VACCUM_AMBIENT_SC2', editable: true, cellStyle: {
          textAlign: 'center',
        },
      },
    ]
  }

];




defaultColConfig['specifications#transmitter#spurious'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#transmitter#spurious'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};

const spurious_spec_cellEditor_renderer_Params = {
  tbl_settings: {
    colHeaders: ["Offset(MHz)", "Value(dBc)"],
    readOnly: false,
    contextMenu: false,
    className: 'handson-cell-text-center',
    height: 250,
    width: '300px',
    columns: [
      {
        type: 'numeric',
        data: 'offset',
        width: 200

      },
      {
        type: 'numeric',
        data: 'value',
        width: 100


      },
    ]
  },
  style: { width: "300px" }
}

tableConfig['specifications#transmitter#spurious'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: true,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },

  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },

  {
    headerName: 'Specification(dBc)',
    field: 'specification',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Tolerance(dB)',
    field: 'tolerance',
    minWidth: 100,
    editable: true
  },
  {
    headerName: 'FBT',
    field: 'FBT',

    minWidth: 300,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { ...spurious_spec_cellEditor_renderer_Params, style: { width: "400px", height: "400px" }, editor_header: "FBT Data" },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: spurious_spec_cellEditor_renderer_Params,
  },
  {
    headerName: 'FBT_COLD',
    field: 'FBT_COLD',
    flex: 1,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { ...spurious_spec_cellEditor_renderer_Params, style: { width: "400px", height: "400px" }, editor_header: "FBT COLD Data" },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: spurious_spec_cellEditor_renderer_Params,
  },
  {
    headerName: 'FBT_HOT',
    field: 'FBT_HOT',
    flex: 1,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { ...spurious_spec_cellEditor_renderer_Params, style: { width: "400px", height: "400px" }, editor_header: "FBT HOT Data" },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: spurious_spec_cellEditor_renderer_Params,
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT',
    field: 'FBT_VACCUM_AMBIENT',
    flex: 1,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: { ...spurious_spec_cellEditor_renderer_Params, style: { width: "400px", height: "400px" }, editor_header: "FBT VACCUM AMBIENT Data" },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: spurious_spec_cellEditor_renderer_Params,
  },

];




defaultColConfig['specifications#receiver#command_threshold'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#receiver#command_threshold'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['specifications#receiver#command_threshold'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Specification(dBm)',
    field: 'specification',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Tolerance(dB)',
    field: 'tolerance',
    flex: 1,
    editable: true
  },

  {
    headerName: 'FBT(dBm)',
    field: 'FBT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_COLD(dBm)',
    field: 'FBT_COLD',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_HOT(dBm)',
    field: 'FBT_HOT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT(dBm)',
    field: 'FBT_VACCUM_AMBIENT',
    flex: 1,
    editable: true
  },

];


defaultColConfig['specifications#transponder#ranging_threshold'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['specifications#transponder#ranging_threshold'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['specifications#transponder#ranging_threshold'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Up Link',
    field: 'mapping_details.up_link_config',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Down Link',
    field: 'mapping_details.down_link_config',
    flex: 1,
    pinned: 'left' 
  },

  {
    headerName: 'Specification(dBm)',
    field: 'specification',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Tolerance(dB)',
    field: 'tolerance',
    flex: 1,
    editable: true
  },

  {
    headerName: 'FBT(dBm)',
    field: 'FBT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_COLD(dBm)',
    field: 'FBT_COLD',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_HOT(dBm)',
    field: 'FBT_HOT',
    flex: 1,
    editable: true
  },
  {
    headerName: 'FBT_VACCUM_AMBIENT(dBm)',
    field: 'FBT_VACCUM_AMBIENT',
    flex: 1,
    editable: true
  },

];


defaultColConfig['test_systems#instrument_address'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_systems#instrument_address'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['test_systems#instrument_address'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Name',
    field: 'name',
    flex: 1,
    cellStyle: { textAlign: 'right' }
  },
  {
    headerName: 'Model',
    field: 'model_number',
    minWidth: 200,
    editable: true,
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: { values: ["CHANNEL_1", "CHANNEL_2"] },

  },
  {
    headerName: 'GPIB Board',
    field: 'gpib_board_index',
    flex: 1,
    editable: true,
  },
  {
    headerName: 'GPIB Address',
    field: 'gpib_address',
    flex: 1,
    editable: true,
  },
  {
    headerName: 'IP Address',
    field: 'ip_address',
    flex: 1,
    editable: true,
  },
  {
    headerName: 'Port Number',
    field: 'port_number',
    flex: 1,
    editable: true,
  },

];




defaultColConfig['test_systems#power_meter_channels'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'left' }
};

ToolBarDisplays['test_systems#power_meter_channels'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['test_systems#power_meter_channels'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Power Meter',
    field: 'name',
    flex: 1,
  },
  {
    headerName: 'Channel',
    field: 'channel',
    flex: 1,
    editable: true,
    cellEditor: 'agSelectCellEditor',
    cellEditorParams: { values: ["CHANNEL_1", "CHANNEL_2"] },

  },

];

defaultColConfig['test_systems#tsm_paths'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'left' }
};

ToolBarDisplays['test_systems#tsm_paths'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['test_systems#tsm_paths'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Path Direction',
    field: 'path_direction',
    flex: 1,
  },
  {
    headerName: 'Path Label',
    field: 'path_label',
    flex: 1,
  },
  {
    headerName: 'UP Link TSM',
    field: 'SDU1',
    flex: 1,
    rowGroup: false,
    editable: true
  },
  {
    headerName: 'DOWN Link TSM',
    field: 'SDU2',
    flex: 1,
    rowGroup: false,
    editable: true
  },
  {
    headerName: 'Interface TSM1',
    field: 'SDU3',
    flex: 1,
    rowGroup: false,
    editable: true
  },
  {
    headerName: 'Interface TSM2',
    field: 'SDU4',
    flex: 1,
    rowGroup: false,
    editable: true
  }
];




defaultColConfig['test_profiles#spurious_bands'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_profiles#spurious_bands'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};


tableConfig['test_profiles#spurious_bands'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Profile Name',
    field: 'profile_name',
    minWidth: 300,
    editable: true
  },
  {
    headerName: 'Bands',
    field: 'band',

    flex: 2,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      tbl_settings: {
        height: 250,
        width: '400px',
        stretchH: 'all',
        colHeaders: ["Start Frequency(MHz)", "Stop Frequency(MHz)"],
        readOnly: false,
        columns: [
          {
            type: 'numeric',
            data: 'start_frequency',
            width: 200
          },
          {
            type: 'numeric',
            data: 'stop_frequency',
            width: 200

          },
        ]

      },
      style: {},
      editor_header: `Add Bands`
    },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: {
      tbl_settings: {

        colHeaders: ["Start Frequency(MHz)", "Stop Frequency(MHz)"],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'start_frequency',
            width: 200

          },
          {
            type: 'numeric',
            data: 'stop_frequency',
            width: 200
          },
        ]
      },
      style: { width: "400px" },
    },
  },
];


defaultColConfig['test_profiles#command_threshold'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_profiles#command_threshold'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};


tableConfig['test_profiles#command_threshold'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Profile Name',
    field: 'profile_name',
    minWidth: 300,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },
  {
    headerName: 'Profile',
    field: 'profile',

    flex: 2,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      tbl_settings: {
        height: 250,
        width: '400px',
        colHeaders: ["Power Levels(dBm)", "Number of Commands"],
        readOnly: false,
        stretchH: 'all',
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'power_level',
            width: 200

          },
          {
            type: 'numeric',
            data: 'number_of_commands',
            width: 200
          },
        ]
      },
      style: {},
      editor_header: "Profile Data"
    },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: {
      tbl_settings: {
        colHeaders: ["Power Levels(dBm)", "Number of Commands"],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'power_level',
            width: 200

          },
          {
            type: 'numeric',
            data: 'number_of_commands',
            width: 200
          },
        ]
      },
      style: { width: "400px" },
    },
  },
  {
    headerName: 'No. Of CMDs @ Threshold',
    field: 'number_of_commands_at_threshold',
    minWidth: 300,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },
  {
    headerName: 'Establish Threshold',
    field: 'establish_threshold',
    minWidth: 300,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },
];

defaultColConfig['test_profiles#ranging_threshold'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_profiles#ranging_threshold'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};

tableConfig['test_profiles#ranging_threshold'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Profile Name',
    field: 'profile_name',
    minWidth: 300,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },
  {
    headerName: 'Profile',
    field: 'profile',

    flex: 2,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      tbl_settings: {
        height: 250,
        width: '200px',
        colHeaders: ["Power Level(dBm)"],
        readOnly: false,
        stretchH: 'all',
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'power_level',
            width: 200

          }
        ]
      },
      style: {},
      editor_header: "Profile Data"
    },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: {
      tbl_settings: {
        colHeaders: ["Power Levels(dBm)"],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'numeric',
            data: 'power_level',
            width: 200

          },

        ]
      },
      style: { width: "200px" },
    },
  },
  {
    headerName: 'Establish Threshold',
    field: 'establish_threshold',
    minWidth: 300,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },
]

defaultColConfig['test_plan#plan_names'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: false,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#plan_names'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};

tableConfig['test_plan#plan_names'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Test Plan Name',
    field: 'plan_name',
    minWidth: 400,
    editable: true,
    cellStyle: { display: 'flex', justifyContent: 'center', alignItems: 'center' },
  },

]

defaultColConfig['test_plan#transmitter'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#transmitter'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};

tableConfig['test_plan#transmitter'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: false,
    rowGroup: true
  },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },
  {
    headerName: 'Power',
    field: 'power',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Frequency',
    field: 'frequency',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Modulation Index',
    field: 'modulation_index',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Spurious',
    field: 'spurious',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Spurious Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/spurious_bands_profile_names/get",
      show_filter:false
    }
  },
   
]

defaultColConfig['test_plan#transmitter#test_selection'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#transmitter#test_selection'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['test_plan#transmitter#test_selection'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: true,
  },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },
  {
    headerName: 'Power',
    field: 'power',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Frequency',
    field: 'frequency',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Modulation Index',
    field: 'modulation_index',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Spurious',
    field: 'spurious',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Spurious Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/spurious_bands_profile_names/get",
      show_filter:false
    }
  },
 
]

defaultColConfig['test_plan#receiver'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#receiver'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};

tableConfig['test_plan#receiver'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: false,
    rowGroup: true
  },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },
  {
    headerName: 'Command Threshold',
    field: 'command_threshold',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Command Threshold Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/command_threshold_profile_names/get",
      show_filter:false
    }
  },
]

defaultColConfig['test_plan#receiver#test_selection'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#receiver#test_selection'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['test_plan#receiver#test_selection'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: true,
  },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false,
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
  },
  {
    headerName: 'Command Threshold',
    field: 'command_threshold',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Command Threshold Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/command_threshold_profile_names/get",
      show_filter:false
    }
  },
]


defaultColConfig['test_plan#transponder'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#transponder'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};

tableConfig['test_plan#transponder'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: false,
    rowGroup: true
  },
  {
    headerName: 'Uplink Config',
    field: 'mapping_details.up_link_config',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Downlink Config',
    field: 'mapping_details.down_link_config',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Ranging Threshold',
    field: 'ranging_threshold',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Ranging Threshold Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/ranging_threshold_profile_names/get",
      show_filter:false
    }
  },
]


defaultColConfig['test_plan#transponder#test_selection'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_plan#transponder#test_selection'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['test_plan#transponder#test_selection'] = [
  {
    headerName: 'Plan Name',
    field: 'plan_name',
    flex: 1,
    hide: true,
    
  },
  {
    headerName: 'Uplink Config',
    field: 'mapping_details.up_link_config',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Downlink Config',
    field: 'mapping_details.down_link_config',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Ranging Threshold',
    field: 'ranging_threshold',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
  {
    headerName: 'Ranging Threshold Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/ranging_threshold_profile_names/get",
      show_filter:false
    }
  },
]




defaultColConfig['calibration#up_link'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['calibration#up_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['calibration#up_link'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Cal Data Loss(dB)',
    field: 'cal_data',
    flex: 1,
    editable: false
  },
  {
    headerName: 'Fixed Pad(dB)',
    field: 'fixed_pad',
    flex: 1,
    editable: true
  },
  {
    headerName: 'TVAC Ref. Cable Correction(dB)',
    field: 'tvac_ref_cable_correction',
    flex: 1,
    editable: true
  },
  
  {
    headerName: 'Total Loss(dB)',
    field: 'total_loss',
    flex: 1,
    editable: false
  },

]



defaultColConfig['calibration#down_link'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['calibration#down_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['calibration#down_link'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Cal Data Loss(dB)',
    field: 'cal_data',
    flex: 1,
    editable: false
  },
  {
    headerName: 'Fixed Pad(dB)',
    field: 'fixed_pad',
    flex: 1,
    editable: true
  },
  {
    headerName: 'TVAC Ref. Cable Correction(dB)',
    field: 'tvac_ref_cable_correction',
    flex: 1,
    editable: true
  },
  {
    headerName: 'Total Loss(dB)',
    field: 'total_loss',
    flex: 1,
    editable: false
  },

]

defaultColConfig['calibration#tvac_ref._cable_details'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['calibration#tvac_ref._cable_details'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['calibration#tvac_ref._cable_details'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },

  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
   {
    headerName: 'RF Cable Length Inside TVAC',
    field: 'rf_cable_length_inside_tvac_chamber',
    flex: 1,
    editable: true
  },
  {
    headerName: 'TVAC Ref.Cable Length',
    field: 'ref_cable_length',
    flex: 1,
    editable: true
  },
  {
    headerName: 'TVAC Ref.Cable Type',
    field: 'cable_type',
    flex: 1,
    editable: true,
    cellEditor: DropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/get_tvac_ref_cable_types",
      show_filter:false
    }
  },
]


defaultColConfig['on_board_losses#up_link'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['on_board_losses#up_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['on_board_losses#up_link'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'On Board Losses(dB)',
    field: 'loss_data',
    flex: 1,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      tbl_settings: {
        height: 250,
        width: '400px',
        colHeaders: ["Label","Loss(dB)"],
        readOnly: false,
        stretchH: 'all',
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'text',
            data: 'label',
            width: 200

          },
          {
            type: 'numeric',
            data: 'loss',
            width: 200

          }

        ]
      },
      style: {},
      editor_header: "On Board Losses"
    },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: {
      tbl_settings: {
        colHeaders: ["Label","Loss(dB)"],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'text',
            data: 'label',
            width: 200

          },
          {
            type: 'numeric',
            data: 'loss',
            width: 200

          }

        ]
      },
      style: { width: "400px" },
    },
  },
  {
    headerName: 'Total Loss(dB)',
    field: 'total_loss',
    flex: 1,
    editable: false
  },

]


defaultColConfig['on_board_losses#down_link'] = {
  sortable: true,
  filter: true,
  flex: 0,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['on_board_losses#down_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['on_board_losses#down_link'] = [
  // {
  //   headerName: 'SNo.',
  //   field: 'sno',
  //   flex: 1,
  //   valueGetter: "node.rowIndex + 1"
  //   },
  {
    headerName: 'Config Label',
    field: 'cfg_label',
    flex: 1,
    hide: true
  },
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    rowGroup: false,
    pinned: 'left' 
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    pinned: 'left' 
  },
  {
    headerName: 'On Board Losses(dB)',
    field: 'loss_data',
    flex: 1,
    editable: true,
    cellEditor: HandsonTableCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      tbl_settings: {
        height: 250,
        width: '400px',
        colHeaders: ["Label","Loss(dB)"],
        readOnly: false,
        stretchH: 'all',
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'text',
            data: 'label',
            width: 200

          },
          {
            type: 'numeric',
            data: 'loss',
            width: 200

          }

        ]
      },
      style: {},
      editor_header: "On Board Losses"
    },
    cellRenderer: HandsonTableCellRenderer,
    cellRendererParams: {
      tbl_settings: {
        colHeaders: ["Label","Loss(dB)"],
        readOnly: true,
        contextMenu: false,
        className: 'handson-cell-text-center',
        columns: [
          {
            type: 'text',
            data: 'label',
            width: 200

          },
          {
            type: 'numeric',
            data: 'loss',
            width: 200

          }

        ]
      },
      style: { width: "400px" },
    },
  },
  {
    headerName: 'Total Loss(dB)',
    field: 'total_loss',
    flex: 1,
    editable: false
  },

]



defaultColConfig['env_data#env_data'] = {
  sortable: true,
  filter: true,
  enableRowGroup: false,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['env_data#env_data'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};


tableConfig['env_data#env_data'] = [
  {
    headerName: 'Parameter',
    field: 'param',
    flex:1,
  },
  {
    headerName: 'Value',
    field: 'value',
    minWidth:400,
    editable:true
    
  },
]


defaultColConfig['test_execution#test_status'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: false,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['test_execution#test_status'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['test_execution#test_status'] = [
  {
    headerName: 'System',
    field: 'system',
    flex: 1,
    
  },
  {
    headerName: 'Planned',
    field: 'planned',
    flex: 1,
    
  },
  {
    headerName: 'Completed',
    field: 'completed',
    flex: 1,
    
  },
  {
    headerName: 'ETC(mm)',
    field: 'etc',
    flex: 1,
    
  },
]


//Calibration UI
defaultColConfig['calibration#channel_selection#up_link'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['calibration#channel_selection#up_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['calibration#channel_selection#up_link'] = [
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    hide: false,
    
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    editable: false,
    
  },
  {
    headerName: 'Loss(dBm)',
    field: 'loss',
    flex: 1,
    editable: false,
    
  },
  {
    headerName: 'Select',
    field: 'select',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  },
]


defaultColConfig['calibration#channel_selection#down_link'] = {
  sortable: true,
  filter: true,
  flex: 1,
  enableRowGroup: true,
  cellStyle: { textAlign: 'center' }
};

ToolBarDisplays['calibration#channel_selection#down_link'] = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: false,
};

tableConfig['calibration#channel_selection#down_link'] = [
  {
    headerName: 'System',
    field: 'code',
    flex: 1,
    hide: false,
    
  },
  {
    headerName: 'Port',
    field: 'port',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Frequency Label',
    field: 'frequency_label',
    flex: 1,
    hide: false
  },
  {
    headerName: 'Frequency(MHz)',
    field: 'frequency_mhz',
    flex: 1,
    editable: false,
    
  },
  {
    headerName: 'Loss(dBm)',
    field: 'loss',
    flex: 1,
    editable: false,
    
  },
  {
    headerName: 'Spurious Profile Name',
    field: 'profile_name',
    flex: 1,
    editable: true,
    cellEditor: MultiSelectDropDownCellEditor,
    cellEditorPopup: true,
    cellEditorParams: {
      url:"tracs/test_profiles/spurious_bands_profile_names/get",
      show_filter:false
    }
  },
  {
    headerName: 'Select',
    field: 'select',
    flex: 1,
    editable: true,
    cellRenderer: TriStateCheckboxRenderer,
  }
  
]




export function tracs_table_config_func(table_name: string) {
  const ColDefs = tableConfig[table_name];
  const DefaultDef = defaultColConfig[table_name];
  const ToolBarConf = ToolBarDisplays[table_name];
  return { DefaultDef, ColDefs, ToolBarConf };
}
//export { defaultColConfig, allow_add_remove_rows, getTableConfig }