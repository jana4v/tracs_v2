import { ref } from 'vue';
import CellStyles from '@/components/AgGrid/CellStyles/Validators';
import DropDownCellEditor from 'components/AgGrid/CellEditors/DropDownCellEditor.vue';
import MultiSelectDropDownCellEditor from '@/components/AgGrid/CellEditors/MultiSelectDropDownCellEditor.vue';
import TransmitterDetails from '@/components/AgGrid/CellEditors/TransmitterDetails.vue';
import multiLineTextRenderer from '@/components/AgGrid/CellRenderers/multiLineTextRenderer.vue';

interface TableColumnDef {
  [key: string]: any;
}

interface ToolBarType {
  [key: string]: boolean;
}


const defaultColConfig: Record<string, TableColumnDef> ={};
const tableConfig: Record<string, TableColumnDef[]> ={};
const ToolBarDisplays: Record<string, ToolBarType> ={};



defaultColConfig['forms#ports'] = {
  sortable: false,
  filter: false,
  flex: 1,
};

ToolBarDisplays['forms#ports'] = {
  is_addRowsAllowed: true,
  is_removeRowsAllowed: true,
  is_saveAllowed: true,
};


tableConfig['forms#ports'] = [
  {
    headerName: 'PORT NAMES',
    field: 'port',
    flex: 1,
    editable: true,
    
    
 },
  
];



export function forms_table_config_func(table_name: string) {
  const ColDefs = tableConfig[table_name];
  const DefaultDef = defaultColConfig[table_name];
  const ToolBarConf = ToolBarDisplays[table_name];
  return { DefaultDef, ColDefs, ToolBarConf };
}
//export { defaultColConfig, allow_add_remove_rows, getTableConfig };
