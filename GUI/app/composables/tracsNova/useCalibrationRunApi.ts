export interface CalibrationChannel {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
}

export interface CalibrationSample {
  code: string;
  port: string;
  frequency_label: string;
  frequency: string;
  value?: number;
  sa_loss?: number;
  dl_pm_loss?: number;
  timestamp: string;
}

export interface CalibrationRunSnapshot {
  run_id: string;
  cal_id: string;
  cal_type: string;
  state: string;
  progress: number;
  status_lines: string[];
  prompt_message: string | null;
  prompt_channel: CalibrationChannel | null;
  samples: CalibrationSample[];
  channels: CalibrationChannel[];
  created_at: string;
  updated_at: string;
  operator_connected: boolean;
}

export interface CalibrationRunStartRequest {
  cal_id: string;
  cal_type: string;
  include_spurious_bands?: boolean | null;
  cal_sg_level?: number | null;
  channels: CalibrationChannel[];
}

export interface CalibrationRunPromptResponseRequest {
  action: 'connected' | 'abort';
}

const BASE_URL = 'http://localhost:8001';

const apiFetch = async (path: string, options: Record<string, any> = {}) => {
  try {
    const data = await $fetch(`${BASE_URL}/${path}`, options);
    return { data: ref(data), error: ref(null) };
  } catch (err) {
    return { data: ref(null), error: ref(err) };
  }
};

export const useCalibrationRunApi = () => {
  const startRun = (payload: CalibrationRunStartRequest) =>
    apiFetch('api/v2/calibration/runs/start', { method: 'POST', body: payload });

  const getActiveRun = () => apiFetch('api/v2/calibration/runs/active');

  const getLatestRun = (calType?: string) => {
    const query = calType ? `?cal_type=${encodeURIComponent(calType)}` : '';
    return apiFetch(`api/v2/calibration/runs/latest${query}`);
  };

  const getRun = (runId: string) => apiFetch(`api/v2/calibration/runs/${encodeURIComponent(runId)}`);

  const respondPrompt = (runId: string, payload: CalibrationRunPromptResponseRequest) =>
    apiFetch(`api/v2/calibration/runs/${encodeURIComponent(runId)}/prompt-response`, {
      method: 'POST',
      body: payload,
    });

  const abortRun = (runId: string) =>
    apiFetch(`api/v2/calibration/runs/${encodeURIComponent(runId)}/abort`, { method: 'POST' });

  const streamUrl = (runId: string) => `${BASE_URL}/api/v2/calibration/runs/${encodeURIComponent(runId)}/events`;

  return {
    startRun,
    getActiveRun,
    getLatestRun,
    getRun,
    respondPrompt,
    abortRun,
    streamUrl,
  };
};
