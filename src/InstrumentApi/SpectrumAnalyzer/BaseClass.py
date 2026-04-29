from abc import ABC,abstractmethod
import math
import time
from typing import Literal, Optional
from ..InstrumentBaseClass import InstrumentBaseClass
from ..Models import Limits,InterfaceProtocols
from pydantic import BaseModel


class Specification(BaseModel):
    model_number: Optional[str] = None
    frequency: Optional[Limits] = None
    span: Optional[Limits] = None
    markers: Optional[Limits] = None
    delta_marker_x_offset: Optional[Limits] = None
    rbw: Optional[Limits] = None
    vbw: Optional[Limits] = None
    peak_excursion_value: Optional[Limits] = None
    trace: Optional[Limits] = None
    reference_level: Optional[Limits] = None
    sweep_points: Optional[Limits] = None
    sweep_time: Optional[Limits] = None
    trace_data_smooth_weightage: Optional[Limits] = None
    vbw_to_rbw_ratio: Optional[Limits] = None
    y_axis_log_scaling: Optional[Limits] = None
    trigger_delay: Optional[Limits] = None
    trigger_offset_to_trace_data: Optional[Limits] = None
    frequency_counter_resolution: Optional[Limits] = None
    display_line_position: Optional[Limits] = None
    peak_detection_threshold: Optional[Limits] = None
    power_attenuation_value: Optional[Limits] = None
    output_power_offset: Optional[Limits] = None
    register_number: Optional[Limits] = None


class SpectrumAnalyzer(ABC, InstrumentBaseClass):
    def __init__(self):
        super().__init__()
        self.supported_protocol = InterfaceProtocols.ANY.value

    @abstractmethod
    def preset_instrument(self):
        pass

    @abstractmethod
    def set_center_frequency(self, frequency):
        pass

    @abstractmethod
    def get_center_frequency(self):
        pass

    @abstractmethod
    def set_span(self, frequency):
        pass

    @abstractmethod
    def get_span(self):
        pass

    @abstractmethod
    def set_span_as_full_range(self):
        pass

    @abstractmethod
    def set_normal_marker(self, marker_number=1):
        pass

    @abstractmethod
    def set_peak_search(self, marker_number=1):
        pass

    @abstractmethod
    def set_min_search(self, marker_number=1):
        pass

    @abstractmethod
    def set_delta_marker_on(self, marker_number=1):
        pass

    @abstractmethod
    def set_delta_marker_peak(self, marker_number=1):
        pass

    @abstractmethod
    def set_delta_marker_maximum_next(self, marker_number=1):
        pass

    @abstractmethod
    def set_delta_marker_maximum_right(self, marker_number=1):
        pass

    @abstractmethod
    def set_delta_marker_maximum_left(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_to_center_frequency(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_to_reference_level(self, marker_number=1):
        pass

    @abstractmethod
    def set_move_delta_marker_x_position(self, x_position_offset):
        pass

    @abstractmethod
    def set_resolution_bandwidth(self, frequency):
        pass

    @abstractmethod
    def get_resolution_bandwidth(self):
        pass

    @abstractmethod
    def set_resolution_bandwidth_auto_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_video_bandwidth(self, frequency):
        pass

    @abstractmethod
    def get_video_bandwidth(self):
        pass

    @abstractmethod
    def set_video_bandwidth_auto_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_marker_continuous_peaking_on_off(self, on_or_off, marker_number=1):
        pass

    @abstractmethod
    def set_marker_peak_excursion_value(self, value):
        pass

    @abstractmethod
    def set_marker_peak_search_mode_to_maximum(self):
        pass

    @abstractmethod
    def set_marker_peak_search_mode_to_minimum(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_peak_search_mode_to_parameter(self):
        pass

    @abstractmethod
    def set_marker_find_next_left_peak(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_find_next_right_peak(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_find_next_peak(self, marker_number=1):
        pass

    @abstractmethod
    def set_marker_assign_to_trace(self, marker_number, trace_number):
        pass

    @abstractmethod
    def set_markers_off(self):
        pass

    @abstractmethod
    def set_clear_status_byte_register(self):
        pass

    @abstractmethod
    def set_reference_level(self, value):
        pass

    @abstractmethod
    def get_reference_level(self):
        pass

    @abstractmethod
    def set_frequency_counter_state_on_off(self, on_or_off, marker_number=1):
        pass

    @abstractmethod
    def set_start_frequency(self, frequency):
        pass

    @abstractmethod
    def get_start_frequency(self):
        pass

    @abstractmethod
    def set_stop_frequency(self, frequency):
        pass

    @abstractmethod
    def get_stop_frequency(self):
        pass

    @abstractmethod
    def set_sweep_points(self, number_of_points):
        pass

    @abstractmethod
    def set_sweep_time(self, sweep_time):
        pass

    @abstractmethod
    def get_delta_marker_delta_y_value(self, marker_number=1):
        pass

    @abstractmethod
    def get_delta_marker_delta_x_value(self, marker_number=1):
        pass

    @abstractmethod
    def get_sweep_time(self):
        pass

    @abstractmethod
    def set_sweep_time_automatic_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_trace_data_smooth(self, trace_number):
        pass

    @abstractmethod
    def set_trace_data_smooth_weightage(self, averaging_window_length):
        pass

    @abstractmethod
    def set_trace_copy(self, destination_trace_number, source_trace_number):
        pass

    @abstractmethod
    def set_trace_exchange(self, trace_number1, trace_number2):
        pass

    @abstractmethod
    def set_trace_math_add(self, destination_trace_number, source_trace_number1, source_trace_number2):
        pass

    @abstractmethod
    def set_trace_math_subtract(self, destination_trace_number, source_trace_number1, source_trace_number2):
        pass

    @abstractmethod
    def set_trace_mode_to_blank(self, trace_number):
        pass

    @abstractmethod
    def set_trace_mode_to_maxhold(self, trace_number):
        pass

    @abstractmethod
    def set_trace_mode_to_minhold(self, trace_number):
        pass

    @abstractmethod
    def set_trace_mode_to_normal(self, trace_number):
        pass

    @abstractmethod
    def set_trace_mode_to_view(self, trace_number):
        pass

    @abstractmethod
    def set_trace_detector_mode(self, mode: str, trace_number=1):
        pass

    @abstractmethod
    def set_vbw_to_rbw_ratio(self, ratio):
        pass

    @abstractmethod
    def set_vbw_to_rbw_ratio_mode_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_y_axis_scaling(self, amplitude):
        pass

    @abstractmethod
    def get_y_axis_scaling(self):
        pass

    @abstractmethod
    def set_trace_peak_sort_by_amplitude_of_trace1_data(self):
        pass

    @abstractmethod
    def set_trace_peak_sort_by_frequency_of_trace1_data(self):
        pass

    @abstractmethod
    def set_trigger_polarity_to_negative(self):
        pass

    @abstractmethod
    def set_trigger_polarity_to_positive(self):
        pass

    @abstractmethod
    def set_trigger_source_as_external(self):
        pass

    @abstractmethod
    def set_trigger_source_as_free_run(self):
        pass

    @abstractmethod
    def set_trigger_delay(self, delay):
        pass

    @abstractmethod
    def set_trigger_delay_state_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_trigger_offset_state_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_trigger_offset_to_trace_data(self, offset_delay):
        pass

    @abstractmethod
    def set_frequency_counter_resolution(self, frequency_counter_resolution, marker_number=1):
        pass

    @abstractmethod
    def set_frequency_counter_resolution_to_auto_on_off(self, on_or_off):
        pass

    @abstractmethod
    def get_error_code(self):
        pass

    @abstractmethod
    def get_frequency_counter_value(self, marker_number=1):
        pass

    @abstractmethod
    def get_marker_value_x_data(self, marker_number=1):
        pass

    @abstractmethod
    def get_marker_value_y_data(self, marker_number=1):
        pass

    @abstractmethod
    def get_trace_mean(self, trace_number=1):
        pass

    @abstractmethod
    def get_number_of_signal_peaks_found_on_trace1(self):
        pass

    @abstractmethod
    def get_signal_peaks_data_of_trace1(self, sort_by_amplitudes=True):
        pass

    @abstractmethod
    def get_trace_data(self, trace_number=1):
        pass

    @abstractmethod
    def set_displayline_state_on_off(self, on_or_off):
        pass

    @abstractmethod
    def get_displayline_state(self):
        pass

    @abstractmethod
    def set_displayline_position(self, amplitude):
        pass

    @abstractmethod
    def get_displayline_position(self):
        pass

    @abstractmethod
    def set_peak_detection_threshold(self, amplitude):
        pass

    @abstractmethod
    def get_peak_detection_threshold(self):
        pass

    @abstractmethod
    def set_initiate_sweep(self):
        pass

    @abstractmethod
    def set_continuous_sweep_on_off(self, on_or_off):
        pass

    @abstractmethod
    def set_power_attenuation_value(self, attenuation):
        pass

    @abstractmethod
    def get_power_attenuation_value(self):
        pass

    @abstractmethod
    def set_output_power_offset(self, amplitude_offset):
        pass

    @abstractmethod
    def set_free_run(self):
        pass

    @abstractmethod
    def set_save_instrument_state(self, register_number=1):
        pass

    @abstractmethod
    def get_recall_instrument_state(self, register_number=1):
        pass

    @abstractmethod
    def set_marker_on_off(self, on_or_off, marker_number=1):
        pass

    @abstractmethod
    def set_move_marker_to_x_position(self, frequency, marker_number=1):
        pass

    @abstractmethod
    def set_wait_until_to_complete_current_action(self):
        pass

    @abstractmethod
    def get_trace_dump(self, file_name):
        pass

    @abstractmethod
    def is_external_lo_reference_connected(self):
        pass

    @abstractmethod
    def is_carrier_presence_at_frequency(self, frequency):
        pass

    def measure_channel_power(
        self,
        center_frequency_mhz: float,
        channel_bandwidth_mhz: float = 1.0,
        trace_number: int = 1,
        detector_mode: Literal["peak", "average", "rms"] = "average",
    ) -> float:
        if channel_bandwidth_mhz <= 0.0:
            raise ValueError("channel_bandwidth_mhz must be > 0")

        # Average detector is used by default to stabilize trace-power integration.
        self.set_trace_detector_mode(detector_mode, trace_number)
        sweep_time = float(self.get_sweep_time())
        time.sleep(0.5+5*sweep_time)  # Allow trace to update with new detector mode
        trace_data = self.get_trace_data(trace_number)
        if not isinstance(trace_data, list) or len(trace_data) < 3:
            raise ValueError("Unable to read valid trace data for channel power measurement")

        span_raw = float(self.get_span())
        rbw_raw = float(self.get_resolution_bandwidth())

        span_hz = span_raw if span_raw > 100_000 else (span_raw * 1_000_000.0)
        rbw_hz = rbw_raw if rbw_raw > 1_000 else (rbw_raw * 1_000.0)
        if rbw_hz <= 0.0:
            raise ValueError("Invalid RBW while computing channel power")

        points = len(trace_data)
        bin_bw_hz = span_hz / float(points - 1)
        center_hz = center_frequency_mhz * 1_000_000.0
        start_freq_hz = center_hz - (span_hz / 2.0)

        half_bw_hz = (channel_bandwidth_mhz * 1_000_000.0) / 2.0
        lower_hz = center_hz - half_bw_hz
        upper_hz = center_hz + half_bw_hz

        first_idx = max(0, int(math.ceil((lower_hz - start_freq_hz) / bin_bw_hz)))
        last_idx = min(points - 1, int(math.floor((upper_hz - start_freq_hz) / bin_bw_hz)))
        if first_idx > last_idx:
            raise ValueError("Computed channel power is invalid (no bins found in channel window)")

        linear_sum_mw = 0.0
        for idx in range(first_idx, last_idx + 1):
            power_in_rbw_mw = 10 ** (float(trace_data[idx]) / 10.0)
            linear_sum_mw += power_in_rbw_mw * (bin_bw_hz / rbw_hz)

        if linear_sum_mw <= 0.0:
            raise ValueError("Computed channel power is invalid (no bins found in channel window)")

        return 10.0 * math.log10(linear_sum_mw)
