import time
from ..factory import factory
from .BaseClass import SpectrumAnalyzer, Specification
from ..Models import InstrumentTypes, Limits, Units
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods, catch_exceptions_and_traceback


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
class RS_FSW(SpectrumAnalyzer):
    """Rohde & Schwarz FSW Signal and Spectrum Analyzer driver."""

    def __init__(self) -> None:
        super().__init__()
        self.specification: Specification = self.initialize_specifications()

    def initialize_specifications(self) -> Specification:
        specification = Specification()
        specification.model_number = 'FSW'
        specification.frequency = Limits(lower_limit=0, upper_limit=85000, units=Units.MHz)
        specification.span = Limits(lower_limit=0, upper_limit=85000, units=Units.MHz)
        specification.markers = Limits(lower_limit=1, upper_limit=16, units=Units.count)
        specification.delta_marker_x_offset = Limits(lower_limit=-85e9, upper_limit=85e9, units=Units.count)
        specification.rbw = Limits(lower_limit=0.001, upper_limit=60000, units=Units.kHz)
        specification.vbw = Limits(lower_limit=0.001, upper_limit=60000, units=Units.kHz)
        specification.peak_excursion_value = Limits(lower_limit=0, upper_limit=200, units=Units.dB)
        specification.trace = Limits(lower_limit=1, upper_limit=6, units=Units.count)
        specification.reference_level = Limits(lower_limit=-130, upper_limit=30, units=Units.dBm)
        specification.sweep_points = Limits(lower_limit=101, upper_limit=100001, units=Units.count)
        specification.sweep_time = Limits(lower_limit=0, upper_limit=16000, units=Units.seconds)
        specification.trace_data_smooth_weightage = Limits(lower_limit=1, upper_limit=50, units=Units.count)
        specification.vbw_to_rbw_ratio = Limits(lower_limit=0.001, upper_limit=1000, units=Units.count)
        specification.y_axis_log_scaling = Limits(lower_limit=1, upper_limit=200, units=Units.dB)
        specification.trigger_delay = Limits(lower_limit=0, upper_limit=30000, units=Units.milli_seconds)
        specification.trigger_offset_to_trace_data = Limits(lower_limit=-30000, upper_limit=30000, units=Units.milli_seconds)
        specification.frequency_counter_resolution = Limits(lower_limit=0.000001, upper_limit=10000, units=Units.MHz)
        specification.display_line_position = Limits(lower_limit=-130, upper_limit=30, units=Units.dBm)
        specification.peak_detection_threshold = Limits(lower_limit=-200, upper_limit=30, units=Units.dBm)
        specification.power_attenuation_value = Limits(lower_limit=0, upper_limit=75, units=Units.dB)
        specification.output_power_offset = Limits(lower_limit=-200, upper_limit=200, units=Units.dB)
        specification.register_number = Limits(lower_limit=1, upper_limit=99, units=Units.count)
        return specification

    def get_instrument_name(self):
        return self.command('*IDN?', read_operation=True)

    def preset_instrument(self):
        self.command('*RST')
        self.command(':INIT:CONT ON')
        self.command(':SENS:ROSC:SOUR EXT')
        return self

    def set_center_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        self.command(f':FREQ:CENT {frequency} MHz')
        return self

    def get_center_frequency(self):
        return float(self.command(':FREQ:CENT?', read_operation=True))

    def set_span(self, frequency):
        self.check_limit(self.specification.span, frequency)
        self.command(f':FREQ:SPAN {frequency} MHz')
        return self

    def get_span(self):
        return float(self.command(':FREQ:SPAN?', read_operation=True))

    def set_span_as_full_range(self):
        self.command(':FREQ:SPAN:FULL')
        return self

    def set_normal_marker(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:STAT ON')
        return self

    def set_peak_search(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MAX:PEAK')
        return self

    def set_min_search(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MIN:PEAK')
        return self

    def set_delta_marker_on(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:DELT{marker_number}:STAT ON')
        return self

    def set_delta_marker_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:DELT{marker_number}:MAX:PEAK')
        return self

    def set_delta_marker_maximum_next(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:DELT{marker_number}:MAX:NEXT')
        return self

    def set_delta_marker_maximum_right(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:DELT{marker_number}:MAX:RIGH')
        return self

    def set_delta_marker_maximum_left(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:DELT{marker_number}:MAX:LEFT')
        return self

    def set_marker_to_center_frequency(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:FUNC:CENT')
        return self

    def set_marker_to_reference_level(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:FUNC:REF')
        return self

    def set_move_delta_marker_x_position(self, x_position_offset):
        self.check_limit(self.specification.delta_marker_x_offset, x_position_offset)
        self.command(f':CALC:DELT1:X {x_position_offset}')
        return self

    def set_resolution_bandwidth(self, frequency):
        self.check_limit(self.specification.rbw, frequency)
        self.command(f':BAND:RES {frequency} KHz')
        return self

    def get_resolution_bandwidth(self):
        return float(self.command(':BAND:RES?', read_operation=True))

    def set_resolution_bandwidth_auto_on_off(self, on_or_off):
        self.command(f':BAND:RES:AUTO {on_or_off}')
        return self

    def set_video_bandwidth(self, frequency):
        self.check_limit(self.specification.vbw, frequency)
        self.command(f':BAND:VID {frequency} KHz')
        return self

    def get_video_bandwidth(self):
        return float(self.command(':BAND:VID?', read_operation=True))

    def set_video_bandwidth_auto_on_off(self, on_or_off):
        self.command(f':BAND:VID:AUTO {on_or_off}')
        return self

    def set_marker_continuous_peaking_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MAX:AUTO {on_or_off}')
        return self

    def set_marker_peak_excursion_value(self, value):
        self.check_limit(self.specification.peak_excursion_value, value)
        self.command(f':CALC:MARK:PEXC {value} dB')
        return self

    def set_marker_peak_search_mode_to_maximum(self):
        self.command(':CALC:MARK:MAX:PEAK')
        return self

    def set_marker_peak_search_mode_to_minimum(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MIN:PEAK')
        return self

    def set_marker_peak_search_mode_to_parameter(self):
        # FSW uses excursion and threshold as the peak-search parameters;
        # enable the threshold gate to activate parameter-based search.
        self.command(':CALC:THR:STAT ON')
        return self

    def set_marker_find_next_left_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MAX:LEFT')
        return self

    def set_marker_find_next_right_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MAX:RIGH')
        return self

    def set_marker_find_next_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:MAX:NEXT')
        return self

    def set_marker_assign_to_trace(self, marker_number, trace_number):
        self.check_limit(self.specification.markers, marker_number)
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':CALC:MARK{marker_number}:TRAC {trace_number}')
        return self

    def set_markers_off(self):
        self.command(':CALC:MARK:AOFF')
        return self

    def set_clear_status_byte_register(self):
        self.command('*CLS')
        return self

    def set_reference_level(self, value):
        self.check_limit(self.specification.reference_level, value)
        self.command(f':DISP:TRAC:Y:RLEV {value} dBm')
        return self

    def get_reference_level(self):
        return float(self.command(':DISP:TRAC:Y:RLEV?', read_operation=True))

    def set_frequency_counter_state_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:COUN {on_or_off}')
        return self

    def set_start_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        self.command(f':FREQ:STAR {frequency} MHz')
        return self

    def get_start_frequency(self):
        return float(self.command(':FREQ:STAR?', read_operation=True))

    def set_stop_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        self.command(f':FREQ:STOP {frequency} MHz')
        return self

    def get_stop_frequency(self):
        return float(self.command(':FREQ:STOP?', read_operation=True))

    def set_sweep_points(self, number_of_points):
        self.check_limit(self.specification.sweep_points, number_of_points)
        self.command(f':SWE:POIN {number_of_points}')
        return self

    def set_sweep_time(self, sweep_time):
        self.check_limit(self.specification.sweep_time, sweep_time)
        self.command(f':SWE:TIME {sweep_time} S')
        return self

    def get_delta_marker_delta_y_value(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        return float(self.command(f':CALC:DELT{marker_number}:Y?', read_operation=True))

    def get_delta_marker_delta_x_value(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        reference_x = float(self.command(f':CALC:MARK{marker_number}:X?', read_operation=True))
        delta_marker_x = float(self.command(f':CALC:DELT{marker_number}:X?', read_operation=True))
        return delta_marker_x - reference_x

    def get_sweep_time(self):
        return float(self.command(':SWE:TIME?', read_operation=True))

    def set_sweep_time_automatic_on_off(self, on_or_off):
        self.command(f':SWE:TIME:AUTO {on_or_off}')
        return self

    def set_trace_data_smooth(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:SMO ON')
        return self

    def set_trace_data_smooth_weightage(self, averaging_window_length):
        self.check_limit(self.specification.trace_data_smooth_weightage, averaging_window_length)
        self.command(f':DISP:TRAC:SMO:APER {averaging_window_length} PCT')
        return self

    def set_trace_copy(self, destination_trace_number, source_trace_number):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number)
        self.command(f':TRAC:COPY TRACE{destination_trace_number},TRACE{source_trace_number}')
        return self

    def set_trace_exchange(self, trace_number1, trace_number2):
        # FSW does not have a native trace exchange command;
        # emulate by copying via a temporary hold on trace memory.
        self.check_limit(self.specification.trace, trace_number1)
        self.check_limit(self.specification.trace, trace_number2)
        raise NotImplementedError("FSW does not support native trace exchange; use set_trace_copy instead.")

    def set_trace_math_add(self, destination_trace_number, source_trace_number1, source_trace_number2):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number1)
        self.check_limit(self.specification.trace, source_trace_number2)
        self.command(f':CALC:MATH{destination_trace_number}:STAT ON')
        self.command(f':CALC:MATH{destination_trace_number}:EXPR:DEF (TRACE{source_trace_number1}+TRACE{source_trace_number2})')
        return self

    def set_trace_math_subtract(self, destination_trace_number, source_trace_number1, source_trace_number2):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number1)
        self.check_limit(self.specification.trace, source_trace_number2)
        self.command(f':CALC:MATH{destination_trace_number}:STAT ON')
        self.command(f':CALC:MATH{destination_trace_number}:EXPR:DEF (TRACE{source_trace_number1}-TRACE{source_trace_number2})')
        return self

    def set_trace_mode_to_blank(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:MODE BLAN')
        return self

    def set_trace_mode_to_maxhold(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:MODE MAXH')
        return self

    def set_trace_mode_to_minhold(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:MODE MINH')
        return self

    def set_trace_mode_to_normal(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:MODE WRIT')
        return self

    def set_trace_mode_to_view(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        self.command(f':DISP:TRAC{trace_number}:MODE VIEW')
        return self

    def set_trace_detector_mode(self, mode: str, trace_number=1):
        self.check_limit(self.specification.trace, trace_number)
        detector = str(mode or "").strip().lower()
        mode_map = {
            # FSW user manual abbreviations: AP (AutoPeak), Av (Average), Rm (RMS)
            "peak": "APE",
            "average": "AVER",
            "rms": "RMS",
        }
        if detector not in mode_map:
            raise ValueError("Unsupported detector mode. Use peak, average, or rms")

        target = mode_map[detector]
        # FSW firmware variants differ in detector command tree support.
        # Try command forms with detector keywords used in FSW documentation.
        command_candidates = [
            f':SENS:DET:FUNC {target}',
            f':SENS:DET {target}',
            f':DET:FUNC {target}',
            f':DET {target}',
            f':DET:TRAC{trace_number} {target}',
        ]

        query_candidates = [
            ':SENS:DET:FUNC?',
            ':SENS:DET?',
            ':DET:FUNC?',
            ':DET?',
        ]

        expected_prefixes = {
            "APE": ("APE", "AP", "AUTO", "POS", "PK", "PEAK"),
            "AVER": ("AVER", "AV", "AVG", "AVERAGE"),
            "RMS": ("RMS", "RM"),
        }[target]

        last_error = None
        for cmd in command_candidates:
            try:
                self.command(cmd)

                # Ensure the changed detector is applied to the current acquisition.
                try:
                    self.command(':INIT:IMM')
                except Exception:
                    pass

                try:
                    for query_cmd in query_candidates:
                        try:
                            detector_value = str(self.command(query_cmd, read_operation=True)).strip().upper()
                            if detector_value.startswith(expected_prefixes):
                                return self
                        except Exception:
                            continue
                    continue
                except Exception:
                    # Query may be unavailable on some firmware; a successful set is enough.
                    return self
            except Exception as exc:
                last_error = exc
                continue

        raise RuntimeError(f"Unable to set detector mode on FSW to {detector}") from last_error

    def set_vbw_to_rbw_ratio(self, ratio):
        self.check_limit(self.specification.vbw_to_rbw_ratio, ratio)
        self.command(f':BAND:VID:RAT {ratio}')
        return self

    def set_vbw_to_rbw_ratio_mode_on_off(self, on_or_off):
        self.command(f':BAND:VID:AUTO {on_or_off}')
        return self

    def set_y_axis_scaling(self, amplitude):
        self.check_limit(self.specification.y_axis_log_scaling, amplitude)
        self.command(f':DISP:TRAC:Y:SPAC LOG')
        self.command(f':DISP:TRAC:Y {amplitude} dB')
        return self

    def get_y_axis_scaling(self):
        return float(self.command(':DISP:TRAC:Y?', read_operation=True))

    def set_trace_peak_sort_by_amplitude_of_trace1_data(self):
        # FSW peak list is accessed via marker peak list (FPEaks); sorting is implicit.
        # Activate the marker peak list sorted by amplitude (default).
        self.command(':CALC:MARK:FUNC:FPE:SORT Y')
        return self

    def set_trace_peak_sort_by_frequency_of_trace1_data(self):
        self.command(':CALC:MARK:FUNC:FPE:SORT X')
        return self

    def set_trigger_polarity_to_negative(self):
        self.command(':TRIG:SLOP NEG')
        return self

    def set_trigger_polarity_to_positive(self):
        self.command(':TRIG:SLOP POS')
        return self

    def set_trigger_source_as_external(self):
        self.command(':TRIG:SOUR EXT')
        return self

    def set_trigger_source_as_free_run(self):
        self.command(':TRIG:SOUR IMM')
        return self

    def set_trigger_delay(self, delay):
        self.check_limit(self.specification.trigger_delay, delay)
        self.command(f':TRIG:HOLD {delay} ms')
        return self

    def set_trigger_delay_state_on_off(self, on_or_off):
        # FSW controls trigger delay via the holdoff time value; 0 = off.
        if str(on_or_off).upper() in ('OFF', '0'):
            self.command(':TRIG:HOLD 0')
        return self

    def set_trigger_offset_state_on_off(self, on_or_off):
        # FSW manages trigger offset via holdoff; this delegates to the delay state.
        self.set_trigger_delay_state_on_off(on_or_off)
        return self

    def set_trigger_offset_to_trace_data(self, offset_delay):
        self.check_limit(self.specification.trigger_offset_to_trace_data, offset_delay)
        self.command(f':TRIG:HOLD {offset_delay} ms')
        return self

    def set_frequency_counter_resolution(self, frequency_counter_resolution, marker_number=1):
        self.check_limit(self.specification.frequency_counter_resolution, frequency_counter_resolution)
        self.check_limit(self.specification.markers, marker_number)
        # FSW resolution is in Hz; convert MHz to Hz
        resolution_hz = frequency_counter_resolution * 1e6
        self.command(f':CALC:MARK{marker_number}:COUN:RES {resolution_hz} Hz')
        return self

    def set_frequency_counter_resolution_to_auto_on_off(self, on_or_off):
        # FSW does not have auto-resolution for the frequency counter;
        # reset to default 0.1 Hz when "auto on" is requested.
        if str(on_or_off).upper() in ('ON', '1'):
            self.command(':CALC:MARK:COUN:RES 0.1 Hz')
        return self

    def get_error_code(self):
        return self.command(':SYST:ERR?', read_operation=True)

    def get_frequency_counter_value(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        return float(self.command(f':CALC:MARK{marker_number}:COUN:FREQ?', read_operation=True))

    def get_marker_value_x_data(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        return float(self.command(f':CALC:MARK{marker_number}:X?', read_operation=True))

    def get_marker_value_y_data(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        return float(self.command(f':CALC:MARK{marker_number}:Y?', read_operation=True))

    def get_trace_mean(self, trace_number=1):
        self.check_limit(self.specification.trace, trace_number)
        # FSW does not expose a direct trace mean query; use the mean power marker function.
        code = f':TRAC? TRACE{trace_number}'
        raw = self.command(code, read_operation=True)
        values = [float(v) for v in raw.split(',')]
        return sum(values) / len(values)

    def get_number_of_signal_peaks_found_on_trace1(self):
        return int(float(self.command(':CALC:MARK:FUNC:FPE:COUN?', read_operation=True)))

    def get_signal_peaks_data_of_trace1(self, sort_by_amplitudes=True):
        if sort_by_amplitudes:
            self.set_trace_peak_sort_by_amplitude_of_trace1_data()
        else:
            self.set_trace_peak_sort_by_frequency_of_trace1_data()
        count = self.get_number_of_signal_peaks_found_on_trace1()
        if count == 0:
            return []
        x_data = self.command(':CALC:MARK:FUNC:FPE:X?', read_operation=True)
        y_data = self.command(':CALC:MARK:FUNC:FPE:Y?', read_operation=True)
        x_values = [float(v) for v in x_data.split(',')]
        y_values = [float(v) for v in y_data.split(',')]
        return list(zip(x_values, y_values))

    def get_trace_data(self, trace_number=1):
        self.check_limit(self.specification.trace, trace_number)
        raw = self.command(f':TRAC? TRACE{trace_number}', read_operation=True, buffer_length=20000)
        return [float(v) for v in raw.split(',')]

    def set_displayline_state_on_off(self, on_or_off):
        self.command(f':CALC:DLIN:STAT {on_or_off}')
        return self

    def get_displayline_state(self):
        return self.command(':CALC:DLIN:STAT?', read_operation=True)

    def set_displayline_position(self, amplitude):
        self.check_limit(self.specification.display_line_position, amplitude)
        self.command(f':CALC:DLIN {amplitude}')
        return self

    def get_displayline_position(self):
        return float(self.command(':CALC:DLIN?', read_operation=True))

    def set_peak_detection_threshold(self, amplitude):
        self.check_limit(self.specification.peak_detection_threshold, amplitude)
        self.command(f':CALC:THR {amplitude} dBm')
        self.command(':CALC:THR:STAT ON')
        return self

    def get_peak_detection_threshold(self):
        return float(self.command(':CALC:THR?', read_operation=True))

    def set_initiate_sweep(self):
        self.command(':INIT:IMM')
        return self

    def set_continuous_sweep_on_off(self, on_or_off):
        self.command(f':INIT:CONT {on_or_off}')
        return self

    def set_power_attenuation_value(self, attenuation):
        self.check_limit(self.specification.power_attenuation_value, attenuation)
        self.command(f':INP:ATT {attenuation} dB')
        return self

    def get_power_attenuation_value(self):
        return float(self.command(':INP:ATT?', read_operation=True))

    def set_output_power_offset(self, amplitude_offset):
        self.check_limit(self.specification.output_power_offset, amplitude_offset)
        self.command(f':DISP:TRAC:Y:RLEV:OFFS {amplitude_offset} dB')
        return self

    def set_free_run(self):
        self.command(':TRIG:SOUR IMM')
        return self

    def set_save_instrument_state(self, register_number=1):
        self.check_limit(self.specification.register_number, register_number)
        self.command(f":MMEM:STOR:STAT 1,'REG{register_number:03d}'")
        return self

    def get_recall_instrument_state(self, register_number=1):
        self.check_limit(self.specification.register_number, register_number)
        self.command(f":MMEM:LOAD:STAT 1,'REG{register_number:03d}'")
        return self

    def set_marker_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.command(f':CALC:MARK{marker_number}:STAT {on_or_off}')
        return self

    def set_move_marker_to_x_position(self, frequency, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.check_limit(self.specification.frequency, frequency)
        self.command(f':CALC:MARK{marker_number}:X {frequency} MHz')
        return self

    def set_wait_until_to_complete_current_action(self):
        self.command('*WAI')
        return self

    def get_trace_dump(self, file_name):
        file_name = file_name + '.png'
        remote_path = f'C:\\TEMP\\{file_name}'
        self.command(f":MMEM:STOR:SCR '{remote_path}'")
        self.set_wait_until_to_complete_current_action()
        raw = self.command(f":MMEM:DATA? '{remote_path}'", read_operation=True, get_raw_data=True)
        with open(file_name, 'wb') as f:
            f.write(raw)
        return file_name

    def is_external_lo_reference_connected(self):
        # Bit 1 of STATus:QUEStionable:FREQuency:CONDition = reference unlock.
        # Value 0 means no frequency error = reference is locked.
        status = self.command(':STAT:QUES:FREQ:COND?', read_operation=True)
        return int(status) == 0

    def is_carrier_presence_at_frequency(self, frequency):
        self.set_center_frequency(frequency)
        self.set_peak_search()
        time.sleep(0.1)
        peak_freq = self.get_marker_value_x_data()
        peak_level = self.get_marker_value_y_data()
        threshold = self.get_peak_detection_threshold()
        freq_tolerance_hz = 1e6
        return abs(peak_freq - frequency * 1e6) < freq_tolerance_hz and peak_level > threshold


factory.register_component(InstrumentTypes.SpectrumAnalyzer, "FSW", RS_FSW)
