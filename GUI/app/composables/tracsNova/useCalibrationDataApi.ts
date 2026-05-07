// ── Types ─────────────────────────────────────────────────────────────────────

export interface CalIdsResponse {
  cal_ids: string[];
}

export interface CalSgCompletedFrequenciesResponse {
  cal_id: string;
  frequencies: number[];
}

export interface CalSgDataRow {
  frequency: number;
  value?: number;
  sa_loss?: number;
  dl_pm_loss?: number;
  datetime: string;
}

export interface CalSgDataRowsResponse {
  cal_id: string;
  rows: CalSgDataRow[];
}

export interface DownlinkCalDataRow {
  code: string;
  port: string;
  frequency: number;
  frequency_label: string;
  value: number;
  datetime: string;
}

export interface DownlinkCalDataRowsResponse {
  cal_id: string;
  rows: DownlinkCalDataRow[];
}

export interface CalibrationReportGenerateRequest {
  cal_id: string;
  cal_type: string;
}

export interface CalibrationReportGenerateResponse {
  cal_id: string;
  cal_type: string;
  satellite_name: string;
  results_directory: string;
  calibration_data_directory: string;
  pdf_path: string | null;
  excel_path: string | null;
  pdf_generated: boolean;
  excel_rows_appended: number;
  message: string;
}

export interface MeasureOptionsResponse {
  test_phases: string[];
  cal_ids: string[];
  test_plan_types: string[];
  default_cal_id: string | null;
  default_test_plan_type: string | null;
}

export interface MeasureTableRow {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  power_selected: boolean;
  frequency_selected: boolean;
  modulation_index_selected: boolean;
  spurious_selected: boolean;
  command_threshold_selected?: boolean;
  ranging_threshold_selected?: boolean;
}

export interface MeasureRunStartRequest {
  test_phase: string;
  sub_test_phase: string;
  cal_id: string;
  test_plan_type: string;
  execution_mode: string;
  remarks?: string;
  continue_on_missing_downlink_cal?: boolean;
  transmitter_rows: MeasureTableRow[];
  receiver_rows: MeasureTableRow[];
  transponder_rows: MeasureTableRow[];
}

export interface MeasureMissingChannel {
  system_kind: 'transmitter' | 'receiver' | 'transponder';
  code: string;
  port: string;
  frequency_label: string;
  frequency: number;
  parameter: string;
}

export interface MeasureRunResultRow {
  system_kind: 'transmitter' | 'receiver' | 'transponder';
  code: string;
  port: string;
  frequency_label: string;
  frequency: number;
  parameter: string;
  measured_value: number;
  applied_loss: number;
  final_value: number;
  raw_value?: number;
  system_loss?: number;
  fixed_pad_loss?: number;
  antenna_gain?: number;
  ground_antenna_gain?: number;
  distance?: number;
  fspl?: number;
  total_loss_calibration?: number;
  on_board_loss?: number;
  status: string;
  message: string;
  timestamp: string;
}

export interface MeasureRunStartResponse {
  message: string;
  processed_rows: number;
  requires_confirmation: boolean;
  missing_downlink_channels: MeasureMissingChannel[];
  requested_parameters: string[];
  results: MeasureRunResultRow[];
}

// ── Internal fetch helper (same base URL as transmitter API) ──────────────────
const BASE_URL = 'http://localhost:8001';

const apiFetch = async (path: string, options: Record<string, any> = {}) => {
  try {
    const data = await $fetch(`${BASE_URL}/${path}`, options);
    return { data: ref(data), error: ref(null) };
  } catch (err) {
    return { data: ref(null), error: ref(err) };
  }
};

// ── Composable ────────────────────────────────────────────────────────────────

export const useCalibrationDataApi = () => {
  /**
   * Fetch distinct cal_ids from the CalibrationData collection.
   * @param calType  optional filter – one of: uplink | downlink | fixedpad | calSG | injectcal
   */
  const getCalIds = (calType?: string) => {
    const query = calType ? `?cal_type=${encodeURIComponent(calType)}` : '';
    return apiFetch(`api/v2/calibration/cal-ids${query}`);
  };

  const getCalSgCompletedFrequencies = (calId: string, calType: string = 'cal_sg') => {
    const normalizedType = String(calType ?? '').trim().toLowerCase();
    const path = normalizedType === 'inject_cal'
      ? 'api/v2/calibration/inject-cal/completed-frequencies'
      : 'api/v2/calibration/cal-sg/completed-frequencies';
    return apiFetch(`${path}?cal_id=${encodeURIComponent(calId)}`);
  };

  const getCalSgData = (calId: string, calType: string = 'cal_sg') => {
    const normalizedType = String(calType ?? '').trim().toLowerCase();
    const path = normalizedType === 'inject_cal'
      ? 'api/v2/calibration/inject-cal/data'
      : 'api/v2/calibration/cal-sg/data';
    return apiFetch(`${path}?cal_id=${encodeURIComponent(calId)}`);
  };

  const generateReport = (payload: CalibrationReportGenerateRequest) =>
    apiFetch('api/v2/calibration/reports/generate', { method: 'POST', body: payload });

  const getDownlinkCalData = (calId: string) =>
    apiFetch(`api/v2/calibration/downlink-cal/data?cal_id=${encodeURIComponent(calId)}`);

  const getMeasureOptions = () =>
    apiFetch('api/v2/measure/options');

  const startMeasureRun = (payload: MeasureRunStartRequest) =>
    apiFetch('api/v2/measure/runs/start', { method: 'POST', body: payload });

  return {
    getCalIds,
    getCalSgCompletedFrequencies,
    getCalSgData,
    generateReport,
    getDownlinkCalData,
    getMeasureOptions,
    startMeasureRun,
  };
};
