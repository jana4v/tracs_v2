// ── Types ─────────────────────────────────────────────────────────────────────

export interface PowerRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  specification: string | number | null;
  tolerance: string | number | null;
  fbt: string | number | null;
  fbt_hot: string | number | null;
  fbt_cold: string | number | null;
}

export interface FrequencyRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  tolerance: string | number | null;
  fbt: string | number | null;
  fbt_hot: string | number | null;
  fbt_cold: string | number | null;
}

export interface ModulationIndexRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  specification: string | number | null;
  tolerance: string | number | null;
  [key: string]: string | number | null;
}

export interface SpuriousRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  specification: string | number | null;
  tolerance: string | number | null;
  fbt: (string | number)[][];
  fbt_hot: (string | number)[][];
  fbt_cold: (string | number)[][];
}

export interface CalibrationSpecLegacyRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  onboard_loss: string | number | null;
  system_loss: string | number | null;
  fixed_pad: string | number | null;
  antenna_gain: string | number | null;
  total_loss: string | number | null;
}

export interface TestProfileSpuriousRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  inband: boolean;
  spurband: boolean;
}

export interface TestProfileSpuriousRowView {
  transmitter_code: string;
  transmitter_name?: string | null;
  modulation_type: string;
  row: TestProfileSpuriousRow;
}

export interface TestProfileSpuriousRowsResponse {
  rows: TestProfileSpuriousRowView[];
}

export interface TestProfileSpuriousRowsUpdateRequest {
  rows: Array<{
    transmitter_code: string;
    row: TestProfileSpuriousRow;
  }>;
}

export interface TestProfileSpuriousRowsUpdateResponse {
  updated_transmitters: number;
  updated_rows: number;
}

export interface SpuriousBandConfigRow {
  profile_name: string;
  enable: boolean;
  start_frequency: number | null;
  stop_frequency: number | null;
}

export interface PskPmDetails {
  ports: string[][];
  sub_carriers: (number | string)[][];
  frequencies: string[][];
  power_specs?: PowerRow[];
  frequency_specs?: FrequencyRow[];
  modulation_index_specs?: ModulationIndexRow[];
  spurious_specs?: SpuriousRow[];
  calibration_specs?: CalibrationSpecLegacyRow[];
  on_board_loss_specs?: Array<{
    code: string;
    port: string;
    frequency_label: string;
    frequency: string;
    loss_db: string | number | null;
  }>;
  test_profile_spurious_specs?: TestProfileSpuriousRow[];
}

export interface Transmitter {
  name: string;
  code: string;
  system_type: string;
  modulation_type: string;
  modulation_details: PskPmDetails;
}

export type Receiver = Transmitter;

export type ParameterName = 'power' | 'frequency' | 'modulation_index' | 'spurious' | 'command_threshold';

export interface ParameterRowView {
  transmitter_code: string;
  transmitter_name?: string | null;
  modulation_type: string;
  row: Record<string, any>;
}

export interface ParameterRowsResponse {
  parameter: ParameterName;
  unit: string;
  rows: ParameterRowView[];
}

export interface ParameterRowsUpdateRequest {
  rows: Array<{
    transmitter_code: string;
    row: Record<string, any>;
  }>;
}

export interface ParameterRowsUpdateResponse {
  parameter: ParameterName;
  unit: string;
  updated_transmitters: number;
  updated_rows: number;
}

export interface CatalogPort {
  port_id: number;
  system_kind: string;
  system_code: string;
  port_name: string;
  sort_order: number;
}

export interface CatalogFrequency {
  frequency_id: number;
  system_kind: string;
  system_code: string;
  frequency_label: string;
  frequency_hz: string;
  sort_order: number;
}

export interface CatalogSpecRow {
  id: number;
  system_kind: string;
  system_code: string;
  parameter_type: ParameterName;
  port_id: number;
  port_name: string;
  frequency_id: number;
  frequency_label: string;
  frequency_hz: string;
  sort_order: number;
  payload: Record<string, any>;
}

export interface CatalogLossRow {
  id: number;
  system_kind: string;
  system_code: string;
  port_id: number;
  port_name: string;
  frequency_id: number;
  frequency_label: string;
  frequency_hz: string;
  loss_db: number | null;
  sort_order: number;
  payload: Record<string, any>;
}

export interface OnBoardLossRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  loss_db: string | number | null;
}

export interface OnBoardLossRowView {
  transmitter_code: string;
  transmitter_name?: string | null;
  modulation_type: string;
  row: OnBoardLossRow;
}

export interface OnBoardLossRowsResponse {
  unit: string;
  rows: OnBoardLossRowView[];
}

export interface OnBoardLossRowsUpdateRequest {
  rows: Array<{
    transmitter_code: string;
    row: OnBoardLossRow;
  }>;
}

export interface OnboardLossItem {
  id?: number;
  source_type: string;
  code: string;
  port: string;
  frequency: string;
  freq_label: string;
  loss_db: number | null;
}

export interface OnboardLossBulkSave {
  rows: OnboardLossItem[];
}

export interface OnBoardLossRowsUpdateResponse {
  unit: string;
  updated_transmitters: number;
  updated_rows: number;
}

export interface CalibrationLossRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  system_loss: string | number | null;
  fixed_pad_loss: string | number | null;
  antenna_gain: string | number | null;
  ground_antenna_gain?: string | number | null;
  distance?: string | number | null;
  total_loss: string | number | null;
}

export interface CalibrationRowView {
  transmitter_code: string;
  transmitter_name?: string | null;
  modulation_type: string;
  row: CalibrationLossRow;
}

export interface CalibrationRowsResponse {
  unit: string;
  rows: CalibrationRowView[];
}

export interface CalibrationRowsUpdateRequest {
  rows: Array<{
    transmitter_code: string;
    row: CalibrationLossRow;
  }>;
}

export interface CalibrationRowsUpdateResponse {
  unit: string;
  updated_transmitters: number;
  updated_rows: number;
}

export interface InstrumentCatalogResponse {
  instruments: Record<string, string[]>;
}

export interface ProjectInstrumentRow {
  instrument_name: string;
  model: string;
  address_main: string;
  address_redt: string;
  use_redt: boolean;
}

export interface ProjectInstrumentsResponse {
  rows: ProjectInstrumentRow[];
}

export interface ProjectInstrumentsSaveRequest {
  rows: ProjectInstrumentRow[];
}

export interface ProjectInstrumentsSaveResponse {
  saved_rows: number;
}

export interface ConfigurationValueResponse {
  parameter: string;
  value: string;
}

export interface ConfigurationValueSaveRequest {
  value: string;
}

export interface ProjectPowerMeterRow {
  powerMeter: string;
  channel: 'A' | 'B';
}

export interface ProjectPowerMetersResponse {
  rows: ProjectPowerMeterRow[];
}

export interface ProjectPowerMetersSaveRequest {
  rows: ProjectPowerMeterRow[];
}

export interface ProjectPowerMetersSaveResponse {
  saved_rows: number;
}

export interface TsmPathRow {
  code: string;
  port: string;
  path1: string | null;
  path2: string | null;
  path3: string | null;
  path4: string | null;
  path5: string | null;
  path6: string | null;
}

export interface TsmPathsResponse {
  rows: TsmPathRow[];
}

export interface TsmPathsSaveRequest {
  rows: TsmPathRow[];
}

export interface TsmPathsSaveResponse {
  saved_rows: number;
}

export interface ProjectTransponderRow {
  name: string;
  code: string;
  rx_code: string;
  rx_port: string;
  rx_freq: string;
  tx_code: string;
  tx_port: string;
  tx_freq: string;
}

export interface ProjectTranspondersResponse {
  rows: ProjectTransponderRow[];
}

export interface ProjectTranspondersSaveRequest {
  rows: ProjectTransponderRow[];
}

export interface ProjectTranspondersSaveResponse {
  saved_rows: number;
}

export interface RangingTone {
  id: number;
  tone_khz: string;
  sort_order: number;
}

export interface RangingThresholdRow {
  id?: number;
  transponder_code: string;
  tone_id: number;
  tone_khz?: string;
  uplink: string;
  downlink: string;
  max_input_power: number | null;
  specification: number | null;
  tolerance: number | null;
  fbt: (string | number)[][] | null;
  fbt_hot: (string | number)[][] | null;
  fbt_cold: (string | number)[][] | null;
  sort_order?: number;
}

export interface RangingThresholdRowsBulkSave {
  rows: RangingThresholdRow[];
}

export interface EnvDataRow {
  parameter: string;
  value: string;
}

export interface EnvDataRowsResponse {
  rows: EnvDataRow[];
}

export interface EnvDataRowsSaveRequest {
  rows: EnvDataRow[];
}

export interface EnvDataRowsSaveResponse {
  saved_rows: number;
}

export interface EnvDataDirectorySelectResponse {
  path: string | null;
}

export type TransmitterSavePayload = Omit<Transmitter, 'system_type'> & {
  system_type?: string;
};

// tracs_nova backend runs on port 8001
const BASE_URL = 'http://localhost:8001';

// Use $fetch (works in any context including Pinia stores)
// Returns { data, error } shape consistent with useFetch for components
const apiFetch = async (path: string, options: Record<string, any> = {}) => {
  try {
    const data = await $fetch(`${BASE_URL}/${path}`, options);
    return { data: ref(data), error: ref(null) };
  } catch (err) {
    return { data: ref(null), error: ref(err) };
  }
};

// ── Composable ────────────────────────────────────────────────────────────────

export const useTransmitterApi = () => {
  const getTransmitters = () => apiFetch('api/v2/transmitters');

  const getTestPlanTypes = () => apiFetch('api/v2/test-plan-types');

  const getReceivers = () => apiFetch('api/v2/receivers');

  const saveTransmitter = (payload: TransmitterSavePayload) =>
    apiFetch('api/v2/transmitters', { method: 'POST', body: payload });

  const saveReceiver = (payload: TransmitterSavePayload) =>
    apiFetch('api/v2/receivers', { method: 'POST', body: payload });

  const deleteTransmitter = (code: string) =>
    apiFetch(`api/v2/transmitters/${code}`, { method: 'DELETE' });

  const deleteReceiver = (code: string) =>
    apiFetch(`api/v2/receivers/${encodeURIComponent(code)}`, { method: 'DELETE' });

  const getModulationTypes = () => apiFetch('api/v2/modulation-types');

  const getParameterRows = (parameter: ParameterName) =>
    apiFetch(`api/v2/transmitters/parameters/${parameter}`);

  const saveParameterRows = (parameter: ParameterName, payload: ParameterRowsUpdateRequest) =>
    apiFetch(`api/v2/transmitters/parameters/${parameter}`, { method: 'PUT', body: payload });

  const getOnBoardLossRows = () =>
    apiFetch('api/v2/transmitters/on-board-losses');

  const saveOnBoardLossRows = (payload: OnBoardLossRowsUpdateRequest) =>
    apiFetch('api/v2/transmitters/on-board-losses', { method: 'PUT', body: payload });

  const getCalibrationRows = () =>
    apiFetch('api/v2/transmitters/calibration');

  const saveCalibrationRows = (payload: CalibrationRowsUpdateRequest) =>
    apiFetch('api/v2/transmitters/calibration', { method: 'PUT', body: payload });

  // Normalized system-catalog APIs (Phase 2+)
  const getSystemCatalogTransmitterPorts = (code: string) =>
    apiFetch(`api/v2/system-catalog/transmitters/${encodeURIComponent(code)}/ports`);

  const getSystemCatalogTransmitterFrequencies = (code: string) =>
    apiFetch(`api/v2/system-catalog/transmitters/${encodeURIComponent(code)}/frequencies`);

  const getSystemCatalogTransmitterSpecRows = (parameter: ParameterName) =>
    apiFetch(`api/v2/system-catalog/transmitters/spec-rows?parameter_type=${encodeURIComponent(parameter)}`);

  const getSystemCatalogReceiverSpecRows = (parameter: ParameterName) =>
    apiFetch(`api/v2/system-catalog/receivers/spec-rows?parameter_type=${encodeURIComponent(parameter)}`);

  const upsertSystemCatalogTransmitterSpecRow = (
    code: string,
    payload: {
      parameter_type: ParameterName;
      port_id: number;
      frequency_id: number;
      payload: Record<string, any>;
      sort_order?: number;
    },
  ) => apiFetch(`api/v2/system-catalog/transmitters/${encodeURIComponent(code)}/spec-rows`, { method: 'POST', body: payload });

  const upsertSystemCatalogReceiverSpecRow = (
    code: string,
    payload: {
      parameter_type: ParameterName;
      port_id: number;
      frequency_id: number;
      payload: Record<string, any>;
      sort_order?: number;
    },
  ) => apiFetch(`api/v2/system-catalog/receiver/${encodeURIComponent(code)}/spec-rows`, { method: 'POST', body: payload });

  const getSystemCatalogSystemPorts = (systemKind: 'transmitter' | 'receiver' | 'transponder', code: string) =>
    apiFetch(`api/v2/system-catalog/${systemKind}/${encodeURIComponent(code)}/ports`);

  const getSystemCatalogSystemFrequencies = (systemKind: 'transmitter' | 'receiver' | 'transponder', code: string) =>
    apiFetch(`api/v2/system-catalog/${systemKind}/${encodeURIComponent(code)}/frequencies`);

  const deleteSystemCatalogSpecRow = (rowId: number) =>
    apiFetch(`api/v2/system-catalog/spec-rows/${rowId}`, { method: 'DELETE' });

  const getSystemCatalogTransmitterLossRows = () =>
    apiFetch('api/v2/system-catalog/transmitters/loss-rows');

  const upsertSystemCatalogTransmitterLossRow = (
    code: string,
    payload: {
      port_id: number;
      frequency_id: number;
      loss_db: number | null;
      payload?: Record<string, any>;
      sort_order?: number;
    },
  ) => apiFetch(`api/v2/system-catalog/transmitters/${encodeURIComponent(code)}/loss-rows`, { method: 'POST', body: payload });

  const getSystemCatalogRows = (systemKind: 'transmitter' | 'receiver' | 'transponder', code: string, rowType: 'test-plan-rows' | 'profile-rows') =>
    apiFetch(`api/v2/system-catalog/${systemKind}/${encodeURIComponent(code)}/${rowType}`);

  const upsertSystemCatalogRow = (
    systemKind: 'transmitter' | 'receiver' | 'transponder',
    code: string,
    rowType: 'test-plan-rows' | 'profile-rows',
    payload: Record<string, any>,
  ) => apiFetch(`api/v2/system-catalog/${systemKind}/${encodeURIComponent(code)}/${rowType}`, { method: 'POST', body: payload });

  const getInstrumentCatalog = () =>
    apiFetch('api/v2/test-systems/instruments/catalog');

  const getProjectInstruments = () =>
    apiFetch('api/v2/test-systems/project-instruments');

  const saveProjectInstruments = (payload: ProjectInstrumentsSaveRequest) =>
    apiFetch('api/v2/test-systems/project-instruments', { method: 'PUT', body: payload });

  const getConfigurationValue = (parameter: string) =>
    apiFetch(`api/v2/test-systems/configuration/${encodeURIComponent(parameter)}`);

  const saveConfigurationValue = (parameter: string, payload: ConfigurationValueSaveRequest) =>
    apiFetch(`api/v2/test-systems/configuration/${encodeURIComponent(parameter)}`, { method: 'PUT', body: payload });

  const getProjectPowerMeters = () =>
    apiFetch('api/v2/test-systems/project-power-meters');

  const saveProjectPowerMeters = (payload: ProjectPowerMetersSaveRequest) =>
    apiFetch('api/v2/test-systems/project-power-meters', { method: 'PUT', body: payload });

  const getProjectTsmPaths = () =>
    apiFetch('api/v2/test-systems/project-tsm-paths');

  const saveProjectTsmPaths = (payload: TsmPathsSaveRequest) =>
    apiFetch('api/v2/test-systems/project-tsm-paths', { method: 'PUT', body: payload });

  const getProjectTransponders = () =>
    apiFetch('api/v2/test-systems/project-transponders');

  const saveProjectTransponders = (payload: ProjectTranspondersSaveRequest) =>
    apiFetch('api/v2/test-systems/project-transponders', { method: 'PUT', body: payload });

  const getRangingTones = () =>
    apiFetch('api/v2/system-catalog/ranging-tones');

  const getRangingThresholdRows = () =>
    apiFetch('api/v2/system-catalog/ranging-threshold-rows');

  const saveRangingThresholdRows = (payload: RangingThresholdRowsBulkSave) =>
    apiFetch('api/v2/system-catalog/ranging-threshold-rows', { method: 'PUT', body: payload });

  const getOnboardLosses = (sourceType: 'transmitter' | 'receiver') =>
    apiFetch(`api/v2/system-catalog/onboard-losses?source_type=${sourceType}`);

  const saveOnboardLosses = (payload: OnboardLossBulkSave) =>
    apiFetch('api/v2/system-catalog/onboard-losses', { method: 'PUT', body: payload });

  const getSpuriousBandConfigs = () =>
    apiFetch('api/v2/spurious-bands');

  const saveSpuriousBandConfigs = (bands: SpuriousBandConfigRow[]) =>
    apiFetch('api/v2/spurious-bands', { method: 'PUT', body: { bands } });

  const getTestProfileSpuriousRows = () =>
    apiFetch('api/v2/transmitters/test-profile-spurious');

  const saveTestProfileSpuriousRows = (payload: TestProfileSpuriousRowsUpdateRequest) =>
    apiFetch('api/v2/transmitters/test-profile-spurious', { method: 'PUT', body: payload });

  const getEnvDataRows = () =>
    apiFetch('api/v2/env-data');

  const saveEnvDataRows = (payload: EnvDataRowsSaveRequest) =>
    apiFetch('api/v2/env-data', { method: 'PUT', body: payload });

  const createEnvDataRow = (payload: EnvDataRow) =>
    apiFetch('api/v2/env-data', { method: 'POST', body: payload });

  const updateEnvDataRow = (parameter: string, payload: EnvDataRow) =>
    apiFetch(`api/v2/env-data/${encodeURIComponent(parameter)}`, { method: 'PUT', body: payload });

  const deleteEnvDataRow = (parameter: string) =>
    apiFetch(`api/v2/env-data/${encodeURIComponent(parameter)}`, { method: 'DELETE' });

  const selectEnvDataDirectory = () =>
    apiFetch('api/v2/env-data/select-directory');

  // ── Receiver Test Profiles ────────────────────────────────────────────────

  const getReceiverTestProfile = (profileType: string) =>
    apiFetch(`api/v2/receiver-test-profiles/${encodeURIComponent(profileType)}`);

  const saveReceiverTestProfile = (payload: { profile_type: string; rows: any[] }) =>
    apiFetch('api/v2/receiver-test-profiles', { method: 'PUT', body: payload });

  const getTransponderTestProfile = (profileType: string) =>
    apiFetch(`api/v2/transponder-test-profiles/${encodeURIComponent(profileType)}`);

  const saveTransponderTestProfile = (payload: { profile_type: string; rows: any[] }) =>
    apiFetch('api/v2/transponder-test-profiles', { method: 'PUT', body: payload });

  // ── Test Plan Selections (per-system-kind, per-test-plan-type) ─────────────

  const getTestPlanSelections = (
    systemKind: 'transmitter' | 'receiver' | 'transponder',
    testPlanName: string,
  ) =>
    apiFetch(
      `api/v2/test-plan/selections/${encodeURIComponent(systemKind)}/${encodeURIComponent(testPlanName)}`,
    );

  const saveTestPlanSelections = (
    systemKind: 'transmitter' | 'receiver' | 'transponder',
    payload: { test_plan_name: string; rows: any[] },
  ) =>
    apiFetch(`api/v2/test-plan/selections/${encodeURIComponent(systemKind)}`, {
      method: 'PUT',
      body: payload,
    });

  return {
    getTransmitters,
    getTestPlanTypes,
    getReceivers,
    saveTransmitter,
    saveReceiver,
    deleteTransmitter,
    deleteReceiver,
    getModulationTypes,
    getParameterRows,
    saveParameterRows,
    getOnBoardLossRows,
    saveOnBoardLossRows,
    getCalibrationRows,
    saveCalibrationRows,
    getSystemCatalogTransmitterPorts,
    getSystemCatalogTransmitterFrequencies,
    getSystemCatalogTransmitterSpecRows,
    getSystemCatalogReceiverSpecRows,
    upsertSystemCatalogTransmitterSpecRow,
    upsertSystemCatalogReceiverSpecRow,
    getSystemCatalogSystemPorts,
    getSystemCatalogSystemFrequencies,
    deleteSystemCatalogSpecRow,
    getSystemCatalogTransmitterLossRows,
    upsertSystemCatalogTransmitterLossRow,
    getSystemCatalogRows,
    upsertSystemCatalogRow,
    getInstrumentCatalog,
    getProjectInstruments,
    saveProjectInstruments,
    getConfigurationValue,
    saveConfigurationValue,
    getProjectPowerMeters,
    saveProjectPowerMeters,
    getProjectTsmPaths,
    saveProjectTsmPaths,
    getProjectTransponders,
    saveProjectTransponders,
    getRangingTones,
    getRangingThresholdRows,
    saveRangingThresholdRows,
    getSpuriousBandConfigs,
    saveSpuriousBandConfigs,
    getTestProfileSpuriousRows,
    saveTestProfileSpuriousRows,
    getEnvDataRows,
    saveEnvDataRows,
    createEnvDataRow,
    updateEnvDataRow,
    deleteEnvDataRow,
    selectEnvDataDirectory,
    getReceiverTestProfile,
    saveReceiverTestProfile,
    getTransponderTestProfile,
    saveTransponderTestProfile,
    getOnboardLosses,
    saveOnboardLosses,
    getTestPlanSelections,
    saveTestPlanSelections,
  };
};
