
import time
from ..factory import factory
from .BaseClass import SpectrumAnalyzer,Specification
from ..Models import InstrumentTypes,Limits,Units
from ..InstrumentBaseClass import apply_exceptions_and_traceback_to_all_methods,catch_exceptions_and_traceback


@apply_exceptions_and_traceback_to_all_methods(catch_exceptions_and_traceback)
class AG_N9030A(SpectrumAnalyzer):

    def __init__(self) -> None:
        super().__init__()
        self.specification: Specification = self.initialize_specifications()

    def initialize_specifications(self) -> Specification:
        specification = Specification()
        specification.model_number = ''
        specification.frequency = Limits(lower_limit=0, upper_limit=44000, units=Units.MHz)
        specification.span = Limits(lower_limit=0, upper_limit=44000, units=Units.MHz)
        specification.markers = Limits(lower_limit=1, upper_limit=4, units=Units.count)
        specification.delta_marker_x_offset = Limits(lower_limit=-10e9, upper_limit=10e9, units=Units.count)
        specification.rbw = Limits(lower_limit=0.01, upper_limit=5000, units=Units.kHz)
        specification.vbw = Limits(lower_limit=0.001, upper_limit=3000, units=Units.kHz)
        specification.peak_excursion_value = Limits(lower_limit=0, upper_limit=70, units=Units.dB)
        specification.trace = Limits(lower_limit=1, upper_limit=3, units=Units.count)
        specification.reference_level = Limits(lower_limit=-120, upper_limit=30, units=Units.dB)
        specification.sweep_points = Limits(lower_limit=101, upper_limit=8192, units=Units.count)
        specification.sweep_time = Limits(lower_limit=0, upper_limit=1500, units=Units.seconds)
        specification.trace_data_smooth_weightage = Limits(lower_limit=3, upper_limit=400, units=Units.count)
        specification.vbw_to_rbw_ratio = Limits(lower_limit=0.00001, upper_limit=3000000, units=Units.count)
        specification.y_axis_log_scaling = Limits(lower_limit=0.1, upper_limit=20, units=Units.dB)
        specification.trigger_delay = Limits(lower_limit=0.0003, upper_limit=429000, units=Units.milli_seconds)
        specification.trigger_offset_to_trace_data = Limits(lower_limit=-100000, upper_limit=100000, units=Units.milli_seconds)
        specification.frequency_counter_resolution = Limits(lower_limit=0.000001, upper_limit=0.1, units=Units.MHz)
        specification.display_line_position = Limits(lower_limit=-100, upper_limit=50, units=Units.dB)
        specification.peak_detection_threshold = Limits(lower_limit=-180, upper_limit=30, units=Units.dBm)
        specification.power_attenuation_value = Limits(lower_limit=0, upper_limit=70, units=Units.dB)
        specification.output_power_offset = Limits(lower_limit=-327, upper_limit=327, units=Units.count)
        specification.register_number = Limits(lower_limit=0, upper_limit=127, units=Units.count)
        return specification

    def get_instrument_name(self):
        code = "*IDN?"
        return self.command(code, read_operation=True)

    def preset_instrument(self):
        code = "*RST;SWE:POIN 1001;:DET POS;:INIT:CONT ON;:FREQ:STAR 2.5GHZ;:FREQ:STOP 4.8GHZ;:BAND:VID 30KHZ;:DISP:MENU:STAT OFF;:CAL:AUTO OFF;"
        code += ":SENS:ROSC:SOUR:TYPE EXT;:STAT:QUES:FREQ:ENAB 2;"
        self.command(code)
        return self

    def set_center_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        code = f':SENSe:FREQuency:CENTer {frequency} MHz'
        self.command(code)
        return self

    def get_center_frequency(self):
        code = ':FREQ:CENT?'
        return self.command(code, read_operation=True)

    def set_span(self, frequency):
        self.check_limit(self.specification.span, frequency)
        code = f':SENSe:FREQuency:SPAN {frequency} MHz'
        self.command(code)
        return self

    def get_span(self):
        code = ':SENSe:FREQuency:SPAN?'
        return float(self.command(code, read_operation=True))

    def set_span_as_full_range(self):
        code = ':SENSe:FREQuency:SPAN:FULL'
        self.command(code)
        return self

    def set_normal_marker(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALC:MARK{marker_number}:MODE POS'
        self.command(code)
        return self

    def set_peak_search(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALC:MARK{marker_number}:MAX'
        self.command(code)
        return self

    def set_min_search(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALC:MARK{marker_number}:MIN'
        self.command(code)
        return self

    def set_delta_marker_on(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALC:MARK{marker_number}:MODE DELT'
        self.command(code)
        return self

    def set_delta_marker_peak(self, marker_number=1):
        code = f':CALCulate:MARKer{marker_number}:MAXimum'
        self.command(code)
        return self

    def set_delta_marker_maximum_next(self, marker_number=1):
        code = f':CALCulate:MARKer{marker_number}:MAXimum:NEXT'
        self.command(code)
        return self

    def set_delta_marker_maximum_right(self, marker_number=1):
        code = f':CALCulate:MARKer{marker_number}:MAXimum:RIGHT'
        self.command(code)
        return self

    def set_delta_marker_maximum_left(self, marker_number=1):
        code = f':CALCulate:MARKer{marker_number}:MAXimum:LEFT'
        self.command(code)
        return self

    def set_marker_to_center_frequency(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:CENTer'
        self.command(code)
        return self

    def set_marker_to_reference_level(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALC:MARK{marker_number}:RLEV'
        self.command(code)
        return self

    def set_move_delta_marker_x_position(self, x_position_offset):
        self.check_limit(self.specification.delta_marker_x_offset, x_position_offset)
        code = f':CALC:MARK1:X {x_position_offset}'
        self.command(code)
        return self

    def set_resolution_bandwidth(self, frequency):
        self.check_limit(self.specification.rbw, frequency)
        code = f':SENSe:BANDwidth:RESolution {frequency} KHz'
        self.command(code)
        return self

    def get_resolution_bandwidth(self):
        code = ':BAND:RES?'
        return self.command(code, read_operation=True)

    def set_resolution_bandwidth_auto_on_off(self, on_or_off):
        code = f':SENSe:BANDwidth:RESolution:AUTO {on_or_off}'
        self.command(code)
        return self

    def set_video_bandwidth(self, frequency):
        self.check_limit(self.specification.vbw, frequency)
        code = f':SENSe:BANDwidth:VIDeo {frequency} KHz'
        self.command(code)
        return self

    def get_video_bandwidth(self):
        code = ':BAND:VID?'
        return self.command(code, read_operation=True)

    def set_video_bandwidth_auto_on_off(self, on_or_off):
        code = f':SENSe:BANDwidth:VIDeo:AUTO {on_or_off}'
        self.command(code)
        return self

    def set_marker_continuous_peaking_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:CPEak:STATe {on_or_off}'
        self.command(code)
        return self

    def set_marker_peak_excursion_value(self, value):
        self.check_limit(self.specification.peak_excursion_value, value)
        code = f':CALCulate:MARKer:PEAK:EXCursion {value} dB'
        self.command(code)
        return self

    def set_marker_peak_search_mode_to_maximum(self):
        code = ':CALCulate:MARKer:PEAK:SEARch:MODE MAXimum'
        self.command(code)
        return self

    def set_marker_peak_search_mode_to_minimum(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:MINImum'
        self.command(code)
        return self

    def set_marker_peak_search_mode_to_parameter(self):
        code = ':CALCulate:MARKer:PEAK:SEARch:MODE PARameter'
        self.command(code)
        return self

    def set_marker_find_next_left_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:MAXimum:LEFT'
        self.command(code)
        return self

    def set_marker_find_next_right_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:MAXimum:RIGHt'
        self.command(code)
        return self

    def set_marker_find_next_peak(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:MAXimum:NEXT'
        self.command(code)
        return self

    def set_marker_assign_to_trace(self, marker_number, trace_number):
        self.check_limit(self.specification.markers, marker_number)
        self.check_limit(self.specification.trace, trace_number)
        code = f':CALCulate:MARKer{marker_number}:TRACe {trace_number}'
        self.command(code)
        return self

    def set_markers_off(self):
        code = ':CALCulate:MARKer:AOFF'
        self.command(code)
        return self

    def set_clear_status_byte_register(self):
        code = '*CLS'
        self.command(code)
        return self

    def set_reference_level(self, value):
        self.check_limit(self.specification.reference_level, value)
        code = f':DISPlay:WINDow:TRACe:Y:RLEVel {value} dBm'
        self.command(code)
        return self

    def get_reference_level(self):
        code = ':DISP:WIND:TRAC:Y:RLEV?'
        return self.command(code, read_operation=True)

    def set_frequency_counter_state_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:FCOunt:STATe {on_or_off}'
        self.command(code)
        return self

    def set_start_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        code = f':SENSe:FREQuency:STARt {frequency} MHz'
        self.command(code)
        return self

    def get_start_frequency(self):
        code = ':FREQ:STAR?'
        return float(self.command(code, read_operation=True))

    def set_stop_frequency(self, frequency):
        self.check_limit(self.specification.frequency, frequency)
        code = f':SENSe:FREQuency:STOP {frequency} MHz'
        self.command(code)
        return self

    def get_stop_frequency(self):
        code = ':FREQ:STOP?'
        return float(self.command(code, read_operation=True))

    def set_sweep_points(self, number_of_points):
        self.check_limit(self.specification.sweep_points, number_of_points)
        code = f':SENSe:SWEep:POINts {number_of_points}'
        self.command(code)
        return self

    def set_sweep_time(self, sweep_time):
        self.check_limit(self.specification.sweep_time, sweep_time)
        code = f':SENSe:SWEep:TIME {sweep_time} S'
        self.command(code)
        return self

    def get_delta_marker_delta_y_value(self, marker_number=1):
        code = f':CALC:MARK{marker_number}:Y?'
        return float(self.command(code, read_operation=True))

    def get_delta_marker_delta_x_value(self, marker_number=1):
        code = f':CALC:MARK{marker_number}:X?'
        return float(self.command(code, read_operation=True))

    def get_sweep_time(self):
        code = ':SENSe:SWEep:TIME?'
        return self.command(code, read_operation=True)

    def set_sweep_time_automatic_on_off(self, on_or_off):
        code = f':SENSe:SWEep:TIME:AUTO {on_or_off}'
        self.command(code)
        return self

    def set_trace_data_smooth(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe:MATH:SMOoth TRACE {trace_number}'
        self.command(code)
        return self

    def set_trace_data_smooth_weightage(self, averaging_window_length):
        self.check_limit(self.specification.trace_data_smooth_weightage, averaging_window_length)
        code = f':TRACe:MATH:SMOoth:POINts {averaging_window_length}'
        self.command(code)
        return self

    def set_trace_copy(self, destination_trace_number, source_trace_number):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number)
        code = f':TRACe:COPY {destination_trace_number},{source_trace_number}'
        self.command(code)
        return self

    def set_trace_exchange(self, trace_number1, trace_number2):
        self.check_limit(self.specification.trace, trace_number1)
        self.check_limit(self.specification.trace, trace_number2)
        code = f':TRACe:EXCHange {trace_number1},{trace_number2}'
        self.command(code)
        return self

    def set_trace_math_add(self, destination_trace_number, source_trace_number1, source_trace_number2):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number1)
        self.check_limit(self.specification.trace, source_trace_number2)
        code = f':TRACe:MATH:ADD {destination_trace_number},{source_trace_number1},{source_trace_number2}'
        self.command(code)
        return self

    def set_trace_math_subtract(self, destination_trace_number, source_trace_number1, source_trace_number2):
        self.check_limit(self.specification.trace, destination_trace_number)
        self.check_limit(self.specification.trace, source_trace_number1)
        self.check_limit(self.specification.trace, source_trace_number2)
        code = f':TRACe:MATH:SUBTract {destination_trace_number},{source_trace_number1},{source_trace_number2}'
        self.command(code)
        return self

    def set_trace_mode_to_blank(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe{trace_number}:MODE BLANK'
        self.command(code)
        return self

    def set_trace_mode_to_maxhold(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe{trace_number}:MODE MAXHold'
        self.command(code)
        return self

    def set_trace_mode_to_minhold(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe{trace_number}:MODE MINHold'
        self.command(code)
        return self

    def set_trace_mode_to_normal(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe{trace_number}:MODE WRITe'
        self.command(code)
        return self

    def set_trace_mode_to_view(self, trace_number):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe{trace_number}:MODE VIEW'
        self.command(code)
        return self

    def set_vbw_to_rbw_ratio(self, ratio):
        self.check_limit(self.specification.vbw_to_rbw_ratio, ratio)
        code = f':SENSe:BANDwidth:VIDeo:RATio {ratio}'
        self.command(code)
        return self

    def set_vbw_to_rbw_ratio_mode_on_off(self, on_or_off):
        code = f':SENSe:BANDwidth:VIDeo:RATio:AUTO {on_or_off}'
        self.command(code)
        return self

    def set_y_axis_scaling(self, amplitude):
        self.check_limit(self.specification.y_axis_log_scaling, amplitude)
        code = f':DISP:WIND:TRAC:Y:SPAC LOG;:DISP:WIND:TRAC:Y:PDIV {amplitude} dB'
        self.command(code)
        return self

    def get_y_axis_scaling(self):
        code = ':DISPlay:WINDow:TRACe:Y:PDIVision?'
        return self.command(code, read_operation=True)

    def set_trace_peak_sort_by_amplitude_of_trace1_data(self):
        code = ':TRACe:MATH:PEAK:SORT AMPLitude'
        self.command(code)
        return self

    def set_trace_peak_sort_by_frequency_of_trace1_data(self):
        code = ':TRACe:MATH:PEAK:SORT FREQuency'
        self.command(code)
        return self

    def set_trigger_polarity_to_negative(self):
        code = ':TRIGger:SEQuence:EXTernal:SLOPe NEGative'
        self.command(code)
        return self

    def set_trigger_polarity_to_positive(self):
        code = ':TRIGger:SEQuence:EXTernal:SLOPe POSitive'
        self.command(code)
        return self

    def set_trigger_source_as_external(self):
        code = ':TRIGger:SEQuence:SOURce EXTernal'
        self.command(code)
        return self

    def set_trigger_source_as_free_run(self):
        code = ':TRIGger:SEQuence:SOURce IMMediate'
        self.command(code)
        return self

    def set_trigger_delay(self, delay):
        self.check_limit(self.specification.trigger_delay, delay)
        code = f':TRIGger:SEQuence:DELay {delay} ms'
        self.command(code)
        return self

    def set_trigger_delay_state_on_off(self, on_or_off):
        code = f':TRIGger:DELay:STATe {on_or_off}'
        self.command(code)
        return self

    def set_trigger_offset_state_on_off(self, on_or_off):
        code = f':TRIGger:SEQuence:OFFset:STATe {on_or_off}'
        self.command(code)
        return self

    def set_trigger_offset_to_trace_data(self, offset_delay):
        self.check_limit(self.specification.trigger_offset_to_trace_data, offset_delay)
        code = f':TRIGger:SEQuence:OFFset {offset_delay} ms'
        self.command(code)
        return self

    def set_frequency_counter_resolution(self, frequency_counter_resolution, marker_number=1):
        self.check_limit(self.specification.frequency_counter_resolution, frequency_counter_resolution)
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:FCOunt:RESolution {frequency_counter_resolution} MHz'
        self.command(code)
        return self

    def set_frequency_counter_resolution_to_auto_on_off(self, on_or_off):
        code = f':CALCulate:MARKer:FCOunt:RESolution:AUTO {on_or_off}'
        self.command(code)
        return self

    def get_error_code(self):
        code = ':SYSTem:ERRor:VERBose ON; :SYSTem:ERRor?'
        return self.command(code, read_operation=True)

    def get_frequency_counter_value(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:FCOunt:X?'
        return float(self.command(code, read_operation=True))

    def get_marker_value_x_data(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:X?'
        return float(self.command(code, read_operation=True))

    def get_marker_value_y_data(self, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:Y?'
        return float(self.command(code, read_operation=True))

    def get_trace_mean(self, trace_number=1):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRACe:MATH:MEAN? TRACE{trace_number}'
        return float(self.command(code, read_operation=True))

    def get_number_of_signal_peaks_found_on_trace1(self):
        code = ':TRACe:MATH:PEAK:POINts?'
        return float(self.command(code, read_operation=True))

    def get_signal_peaks_data_of_trace1(self, sort_by_amplitudes=True):
        if not sort_by_amplitudes:
            self.set_trace_peak_sort_by_frequency_of_trace1_data()
        code_to_check_points = ':TRACe:MATH:PEAK:POIN?'
        no_of_points = int(self.command(code_to_check_points, read_operation=True))
        if no_of_points != 0:
            code = ':TRACe:MATH:PEAK:DATA?'
            return self.command(code, read_operation=True, read_buffer_length=10000)
        else:
            return []

    def get_trace_data(self, trace_number=1):
        self.check_limit(self.specification.trace, trace_number)
        code = f':TRAC? TRACE{trace_number}'
        result = self.command(code, read_operation=True, read_buffer_length=20000)
        result = result.split(',')
        trace_data = [float(i) for i in result]
        return trace_data

    def set_displayline_state_on_off(self, on_or_off):
        code = f':DISPlay:WINDow:TRACe:Y:DLINe:STATe {on_or_off}'
        self.command(code)
        return self

    def get_displayline_state(self):
        code = ':DISPlay:WINDow:TRACe:Y:DLINe:STATe?'
        return self.command(code, read_operation=True)

    def set_displayline_position(self, amplitude):
        self.check_limit(self.specification.display_line_position, amplitude)
        code = f':DISPlay:WINDow:TRACe:Y:DLINe {amplitude}'
        self.command(code)
        return self

    def get_displayline_position(self):
        code = ':DISPlay:WINDow:TRACe:Y:DLINe?'
        return self.command(code, read_operation=True)

    def set_peak_detection_threshold(self, amplitude):
        self.check_limit(self.specification.peak_detection_threshold, amplitude)
        code = f':CALCulate:MARKer:PEAK:THReshold {amplitude} dBm'
        self.command(code)
        return self

    def get_peak_detection_threshold(self):
        code = ':CALCulate:MARKer:PEAK:THReshold?'
        return self.command(code, read_operation=True)

    def set_initiate_sweep(self):
        code = ':INITiate:IMMediate'
        self.command(code)
        return self

    def set_continuous_sweep_on_off(self, on_or_off):
        code = f':INIT:CONT {on_or_off};:INIT'
        self.command(code)
        return self

    def set_power_attenuation_value(self, attenuation):
        self.check_limit(self.specification.power_attenuation_value, attenuation)
        code = f':SOURce:POWer:ATTenuation {attenuation} dB'
        self.command(code)
        return self

    def get_power_attenuation_value(self):
        code = ':SOURce:POWer:ATTenuation?'
        return self.command(code, read_operation=True)

    def set_output_power_offset(self, amplitude_offset):
        self.check_limit(self.specification.output_power_offset, amplitude_offset)
        code = f':SOURce:CORRection:OFFSet {amplitude_offset}'
        self.command(code)
        return self

    def set_free_run(self):
        code = ':TRIG:SEQ:SOUR IMM'
        self.command(code)
        return self

    def set_save_instrument_state(self, register_number=1):
        self.check_limit(self.specification.register_number, register_number)
        code = f'*SAV {register_number}'
        self.command(code)
        return self

    def get_recall_instrument_state(self, register_number=1):
        self.check_limit(self.specification.register_number, register_number)
        code = f'*RCL {register_number}'
        self.command(code)
        return self

    def set_marker_on_off(self, on_or_off, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        code = f':CALCulate:MARKer{marker_number}:STATe {on_or_off}'
        self.command(code)
        return self

    def set_move_marker_to_x_position(self, frequency, marker_number=1):
        self.check_limit(self.specification.markers, marker_number)
        self.check_limit(self.specification.frequency, frequency)
        code = f':CALC:MARK{marker_number}:X {frequency} MHz'
        self.command(code)
        return self

    def set_wait_until_to_complete_current_action(self):
        code = '*WAI'
        self.command(code)
        return self

    def get_trace_dump(self, file_name):
        file_name = file_name + '.png'
        code = ":DISP:MENU:STAT OFF; :MMEM:DEL 'C:\\TEMP\\SASCRN.PNG'; :MMEM:STOR:SCR 'C:\\TEMP\\SASCRN.PNG'"
        self.command(code)
        self.set_wait_until_to_complete_current_action()
        code = "MMEM:DATA? 'C:\\TEMP\\SASCRN.PNG'"
        file_size_digit_length = self.command(code, True, 2)
        file_size = self.command('', True, int(file_size_digit_length[1:]))
        self.command('', True, int(file_size), get_raw_data=True, sa_dump_file_name=file_name)
        return file_name

    def is_external_lo_reference_connected(self):
        code = ':STATus:QUEStionable:FREQuency:CONDition?'
        lock_status = self.command(code, read_operation=True)
        return int(lock_status) == 0

    def is_carrier_presence_at_frequency(self, frequency):
        from ..exceptions import DriverNotImplementedError
        raise DriverNotImplementedError(
            "is_carrier_presence_at_frequency is not implemented for N9030A"
        )


factory.register_component(InstrumentTypes.SpectrumAnalyzer, "N9030A", AG_N9030A)
