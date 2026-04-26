interface editOptionsType {
  is_addRowsAllowed: boolean;
  is_removeRowsAllowed: boolean;
  is_saveAllowed: boolean;
}

const DefaultEditOptions: editOptionsType = {
  is_addRowsAllowed: false,
  is_removeRowsAllowed: false,
  is_saveAllowed: true,
};

interface save_data_callback_return_type {
  error: boolean;
  data?: Record<string, any>;
  message: string;
}

interface before_table_load_callback_return_type {
  url: string;
  data?: Record<string, any>;
}
interface before_table_save_callback_retrun_type {
  error: boolean;
  data?: Record<string, any>;
  message: string;
}

interface BeforeTableLoadCallback {
  (url: string): before_table_load_callback_return_type;
}
interface BeforeTableSaveCallback {
  (tbl_data: Record<string, any>[]): before_table_save_callback_retrun_type;
}

export {
  editOptionsType,
  DefaultEditOptions,
  save_data_callback_return_type,
  before_table_load_callback_return_type,
  before_table_save_callback_retrun_type,
  BeforeTableLoadCallback,
  BeforeTableSaveCallback,
};
