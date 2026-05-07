import time, math, numpy
import scipy.special as besel
import logging
import re, asyncio
import numpy as np
import tkinter as tk
from tkinter import ttk, messagebox, scrolledtext
import csv
from datetime import datetime
from pathlib import Path
import threading
import os
import json
import redis

logging.basicConfig(filename='test.log', level=logging.DEBUG)
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__file__).parent))

import InstrumentApi as ins
from InstrumentApi.Models import InstrumentAddress

class log_handler(logging.StreamHandler):
    def __init__(self):
        logging.StreamHandler.__init__(self)

    def emit(self, record):
        print(record)


rootlogger = logging.getLogger("")
rootlogger.setLevel(logging.DEBUG)
logger = logging.getLogger("TEST")

log = log_handler()
rootlogger.addHandler(log)


class modIndex(object):
    @classmethod
    def sideBandLevelToModIndex(self,sidebandLevelWithRespectToCarrier):
        besselChart=dict((round((20*math.log10(besel.j0(modIndex))-20*math.log10(besel.j1(modIndex))),2),
                          round(modIndex,2)) for modIndex in np.arange(0.01,2,0.01))
        difference=dict((round(abs(key+sidebandLevelWithRespectToCarrier),2),key) for key in besselChart.keys())
        return besselChart.get(difference.get(min(difference.keys())))


    def getModIndexFromTrace(self, traceData, span, tone):
        if tone == 0 or tone == None:
            return None
        frsp_bin = span / 1001.0  # MHz per bin (frequency resolution)
        window_length = int(0.010 / frsp_bin)  # 10 kHz window in bins
        half_window_length = int(window_length / 2)
        window_center = 501
        cf_peak= np.max(traceData)
        #cf_peak = np.max(traceData[window_center - half_window_length:window_center + half_window_length])
        window_center = 501 + int((tone / 1000000) / frsp_bin)  # offset in bins for tone frequency
        sb1_peak = np.max(traceData[window_center - half_window_length:window_center + half_window_length])
        if abs(abs(cf_peak)-abs(sb1_peak)) > 20:
            return None
        window_center = 501 - int((tone / 1000000) / frsp_bin)  # offset in bins for tone frequency
        sb2_peak = np.max(traceData[window_center - half_window_length:window_center + half_window_length])
        sb1_modIndex = self.sideBandLevelToModIndex(sb1_peak-cf_peak)
        sb2_modIndex = self.sideBandLevelToModIndex(sb2_peak-cf_peak)
        logger.info(f'{round(sb1_peak-cf_peak,2)}, {round(sb2_peak-cf_peak,2)}, {round(((sb1_modIndex + sb2_modIndex) / 2),2)}')
        return round(sb1_peak-cf_peak,2), round(sb2_peak-cf_peak,2), (sb1_modIndex+sb2_modIndex)/2
    
    async def PMmodIndex(self,centerFrequency=None,tone1Frequency=32000,tone2Frequency=0,
                         method='MAX_HOLD',SA:ins.SpectrumAnalyzer=None,CF_level_at_traceHold = None,noOfIterations = 5,isRanging=False):
        logger.info('Calculating span,VBW and RBW')
        # span,RBW,VBW=[max(tone1Frequency,tone2Frequency)*2.8/1e6,
        #               max(tone1Frequency,tone2Frequency)/20000,
        #               max(tone1Frequency,tone2Frequency)*2.3/1e6]
        span, RBW, VBW = [max(tone1Frequency, tone2Frequency) * 2.8 / 1e6,
                          max(tone1Frequency, tone2Frequency) / 20000,
                          1]
        if CF_level_at_traceHold is None:
            SA.set_span(span)
            SA.set_resolution_bandwidth(RBW)
            SA.set_video_bandwidth(VBW)
            SA.set_marker_on_off(0)
            SA.set_center_frequency(centerFrequency)
        await asyncio.sleep(1)
        _sa_sweep_time = float(SA.get_sweep_time() )
        await asyncio.sleep(_sa_sweep_time * 2+1)
        SA.set_peak_search()
        await asyncio.sleep(1)
        SA.set_marker_to_center_frequency()
        CF_level = SA.get_delta_marker_delta_y_value()
        sa_freq_value = float(SA.get_marker_value_x_data())
        SA.set_reference_level(CF_level + 4)
        await asyncio.sleep(0.5)

        SC1modIndex_A=0
        SC2modIndex_A=0
        TM1sideBands_A_upper = []
        TM1sideBands_A_lower = []
        TM2sideBands_A_upper = []
        TM2sideBands_A_lower = []
        iteration_details = []  # Store details for each iteration
        noOfIterations = 5
        for i in range(0,noOfIterations):
            logger.info('Measuring mod index sample {0}'.format(i))
            #if not isRanging:
            SA.set_trace_mode_to_normal(1)
            SA.set_center_frequency(centerFrequency)
            await asyncio.sleep(0.5)
            SA.set_peak_search()
            await asyncio.sleep(0.5)
            SA.set_marker_to_center_frequency()
            await asyncio.sleep(0.5)
            if method=='MAX_HOLD':
                #if not isRanging:
                SA.set_trace_mode_to_maxhold(1)
            if isRanging:
                await asyncio.sleep(_sa_sweep_time*5+1)
                await asyncio.sleep(0.5)
            else:
                await asyncio.sleep(_sa_sweep_time*5+1)
                await asyncio.sleep(1)

            #SA.setPeakSearch()
            #CF_level = SA.getMarkerValue_Y_Data()
            #await asyncio.sleep(0.1)
            tracedata = np.array(SA.get_trace_data(1))
            #data = await self.getModIndexFromDeltaMarker(SA,tone1Frequency)
            #data = self.getModIndexFromPeak(SA_obj=SA,tone_Hz=tone1Frequency,sa_freq_value=sa_freq_value,sa_carrier_level=CF_level)
            
            # Store iteration data
            iter_data = {'iteration': i}
            
            data = self.getModIndexFromTrace(tracedata, span, tone1Frequency)
            if data and data[0] != 0:
                TM1sideBands_A_upper.append(data[0])
                TM1sideBands_A_lower.append(data[1])
                SC1modIndex_A += data[2]
                iter_data['32khz_sb1'] = data[0]
                iter_data['32khz_sb2'] = data[1]
                iter_data['32khz_modindex'] = data[2]
            else:
                iter_data['32khz_sb1'] = None
                iter_data['32khz_sb2'] = None
                iter_data['32khz_modindex'] = None
                
            if i == 0 and SC1modIndex_A < 0.2:
                raise Exception('Either No {0} Hz PM Modulation detected on  DL at frequency:{1} MHz or Modulation Index is too Low'.format(tone1Frequency,centerFrequency))
            await asyncio.sleep(2)
            #data = await self.getModIndexFromDeltaMarker(SA, tone2Frequency)
            #data = self.getModIndexFromPeak(SA_obj=SA, tone_Hz=tone2Frequency, sa_freq_value=sa_freq_value,
                                           # sa_carrier_level=CF_level)
            if tone2Frequency:
                data = self.getModIndexFromTrace(tracedata, span, tone2Frequency)
                if data and data[0] != 0:
                    TM2sideBands_A_upper.append(data[0])
                    TM2sideBands_A_lower.append(data[1])
                    SC2modIndex_A += data[2]
                    iter_data['128khz_sb1'] = data[0]
                    iter_data['128khz_sb2'] = data[1]
                    iter_data['128khz_modindex'] = data[2]
                else:
                    iter_data['128khz_sb1'] = None
                    iter_data['128khz_sb2'] = None
                    iter_data['128khz_modindex'] = None
            else:
                iter_data['128khz_sb1'] = None
                iter_data['128khz_sb2'] = None
                iter_data['128khz_modindex'] = None
            
            iteration_details.append(iter_data)


        SA.set_trace_mode_to_normal(1)
        SC1modIndex = round(SC1modIndex_A/noOfIterations,2)
        TM1sideBands = [min(TM1sideBands_A_upper),min(TM1sideBands_A_lower)]
        if len(TM2sideBands_A_upper) > 1:
            SC2modIndex = round(SC2modIndex_A / noOfIterations, 2)
            TM2sideBands = [min(TM2sideBands_A_upper),min(TM2sideBands_A_lower)]
            return SC1modIndex, SC2modIndex, TM1sideBands, TM2sideBands, iteration_details
        else:
            return SC1modIndex, 0, TM1sideBands, 0, iteration_details

# Add the backend/src directory to Python path



def get_instrument_address(address: str) -> InstrumentAddress:
    if re.search("[gG][pP][iI][bB]", address):
        _address = int(address.split("::")[1])
        board_index = int(address.split("::")[0][-1])
        return InstrumentAddress(ip_or_gpib_address=_address, port_or_gpib_bus=board_index)
    else:
        ip_port = address.split(":")
        if len(ip_port) > 1:
            ip_address = ip_port[0]
            tcp_port_number = int(ip_port[1])
        else:
            ip_address = address
            tcp_port_number = 0
        return InstrumentAddress(ip_or_gpib_address=ip_address, port_or_gpib_bus=tcp_port_number)


class ModIndexGUI:
    def __init__(self, root):
        self.root = root
        self.root.title("Modulation Index Continuous Measurement")
        self.root.geometry("700x750")
        
        self.sa = None
        self.is_measuring = False
        self.measurement_thread = None
        self.data_folder = Path(os.path.expandvars("%PUBLIC%")) / "mode index data"
        self.data_folder.mkdir(parents=True, exist_ok=True)
        self.settings_file = self.data_folder / "settings.json"
        
        self.create_widgets()
        self.load_settings()
        
        # Save settings on window close
        self.root.protocol("WM_DELETE_WINDOW", self.on_closing)
        
    def create_widgets(self):
        # Connection Frame
        conn_frame = ttk.LabelFrame(self.root, text="Spectrum Analyzer Connection", padding=10)
        conn_frame.pack(fill="x", padx=10, pady=5)
        
        ttk.Label(conn_frame, text="Address (GPIB/IP):").grid(row=0, column=0, sticky="w", pady=5)
        self.address_entry = ttk.Entry(conn_frame, width=30)
        self.address_entry.grid(row=0, column=1, padx=5, pady=5)
        self.address_entry.insert(0, "GPIB0::18")
        
        ttk.Label(conn_frame, text="SA Model:").grid(row=1, column=0, sticky="w", pady=5)
        self.model_var = tk.StringVar(value="N9030A")
        model_combo = ttk.Combobox(conn_frame, textvariable=self.model_var, width=28, state="readonly")
        model_combo['values'] = ("N9030A", "E4440A", "E4448A", "N9020A")
        model_combo.grid(row=1, column=1, padx=5, pady=5)
        
        self.connect_btn = ttk.Button(conn_frame, text="Connect", command=self.connect_sa)
        self.connect_btn.grid(row=0, column=2, rowspan=2, padx=5, pady=5)
        
        # Configuration Frame
        config_frame = ttk.LabelFrame(self.root, text="Measurement Configuration", padding=10)
        config_frame.pack(fill="x", padx=10, pady=5)
        
        ttk.Label(config_frame, text="Center Frequency (MHz):").grid(row=0, column=0, sticky="w", pady=5)
        self.center_freq_entry = ttk.Entry(config_frame, width=20)
        self.center_freq_entry.grid(row=0, column=1, padx=5, pady=5)
        self.center_freq_entry.insert(0, "6000.0")
        
        # Telemetry Mnemonics
        ttk.Label(config_frame, text="Telemetry Mnemonics:").grid(row=1, column=0, sticky="w", pady=5)
        self.tm_mnemonics_entry = ttk.Entry(config_frame, width=40)
        self.tm_mnemonics_entry.grid(row=1, column=1, padx=5, pady=5, sticky="ew")
        self.tm_mnemonics_entry.insert(0, "")
        ttk.Label(config_frame, text="(comma-separated)", font=("Arial", 8), foreground="gray").grid(row=1, column=2, sticky="w", pady=5)
        
        ttk.Label(config_frame, text="Measurement Interval (sec):").grid(row=2, column=0, sticky="w", pady=5)
        self.interval_entry = ttk.Entry(config_frame, width=20)
        self.interval_entry.grid(row=2, column=1, padx=5, pady=5)
        self.interval_entry.insert(0, "10")
        
        # Output filename
        ttk.Label(config_frame, text="Output Filename:").grid(row=3, column=0, sticky="w", pady=5)
        self.filename_entry = ttk.Entry(config_frame, width=40)
        self.filename_entry.grid(row=3, column=1, padx=5, pady=5, sticky="ew")
        self.filename_entry.insert(0, "")
        ttk.Label(config_frame, text="(appended to timestamp)", font=("Arial", 8), foreground="gray").grid(row=3, column=2, sticky="w", pady=5)
        
        # Control Frame
        control_frame = ttk.Frame(self.root, padding=10)
        control_frame.pack(fill="x", padx=10, pady=5)
        
        self.start_btn = ttk.Button(control_frame, text="Start Continuous Measurement", 
                                     command=self.start_measurement, state="disabled")
        self.start_btn.pack(side="left", padx=5)
        
        self.stop_btn = ttk.Button(control_frame, text="Stop Measurement", 
                                    command=self.stop_measurement, state="disabled")
        self.stop_btn.pack(side="left", padx=5)
        
        # Status Frame
        status_frame = ttk.LabelFrame(self.root, text="Status", padding=10)
        status_frame.pack(fill="both", expand=True, padx=10, pady=5)
        
        self.status_text = scrolledtext.ScrolledText(status_frame, height=15, width=80)
        self.status_text.pack(fill="both", expand=True)
        
        # Results Frame
        results_frame = ttk.LabelFrame(self.root, text="Current Results", padding=10)
        results_frame.pack(fill="x", padx=10, pady=5)
        
        self.results_label = ttk.Label(results_frame, text="No measurements yet", font=("Arial", 10))
        self.results_label.pack()
        
        # Data folder info
        info_label = ttk.Label(self.root, 
                               text=f"Data saved to: {self.data_folder}", 
                               font=("Arial", 8), foreground="blue")
        info_label.pack(pady=5)
    
    def load_settings(self):
        """Load user settings from JSON file"""
        try:
            if self.settings_file.exists():
                with open(self.settings_file, 'r') as f:
                    settings = json.load(f)
                
                # Restore connection settings
                if 'address' in settings:
                    self.address_entry.delete(0, tk.END)
                    self.address_entry.insert(0, settings['address'])
                
                if 'model' in settings:
                    self.model_var.set(settings['model'])
                
                # Restore measurement settings
                if 'center_frequency' in settings:
                    self.center_freq_entry.delete(0, tk.END)
                    self.center_freq_entry.insert(0, settings['center_frequency'])
                
                if 'interval' in settings:
                    self.interval_entry.delete(0, tk.END)
                    self.interval_entry.insert(0, settings['interval'])
                
                # Restore telemetry mnemonics
                if 'tm_mnemonics' in settings:
                    self.tm_mnemonics_entry.delete(0, tk.END)
                    self.tm_mnemonics_entry.insert(0, settings['tm_mnemonics'])
                
                # Restore filename
                if 'filename' in settings:
                    self.filename_entry.delete(0, tk.END)
                    self.filename_entry.insert(0, settings['filename'])
                
                self.log("Settings loaded from previous session")
        except Exception as e:
            self.log(f"Could not load settings: {str(e)}")
    
    def save_settings(self):
        """Save user settings to JSON file"""
        try:
            settings = {
                'address': self.address_entry.get(),
                'model': self.model_var.get(),
                'center_frequency': self.center_freq_entry.get(),
                'interval': self.interval_entry.get(),
                'tm_mnemonics': self.tm_mnemonics_entry.get(),
                'filename': self.filename_entry.get()
            }
            
            with open(self.settings_file, 'w') as f:
                json.dump(settings, f, indent=4)
            
            self.log("Settings saved")
        except Exception as e:
            self.log(f"Could not save settings: {str(e)}")
    
    def on_closing(self):
        """Handle window closing event"""
        if self.is_measuring:
            if messagebox.askokcancel("Quit", "Measurement is running. Stop and quit?"):
                self.is_measuring = False
                time.sleep(0.5)  # Give thread time to stop
                self.save_settings()
                self.root.destroy()
        else:
            self.save_settings()
            self.root.destroy()
        
    def log(self, message):
        timestamp = datetime.now().strftime("%Y-%m-%d %H:%M:%S")
        self.status_text.insert(tk.END, f"[{timestamp}] {message}\n")
        self.status_text.see(tk.END)
        self.root.update_idletasks()
        
    def connect_sa(self):
        try:
            self.log("Connecting to Spectrum Analyzer...")
            address = self.address_entry.get()
            model = self.model_var.get()
            
            sa_address = get_instrument_address(address)
            self.sa = ins.get_instrument(ins.InstrumentTypes.SpectrumAnalyzer, model, sa_address)
            
            self.log(f"Successfully connected to {model} at {address}")
            self.connect_btn.config(state="disabled")
            self.start_btn.config(state="normal")
            self.save_settings()  # Save connection settings
            messagebox.showinfo("Success", "Connected to Spectrum Analyzer")
        except Exception as e:
            self.log(f"Error connecting: {str(e)}")
            messagebox.showerror("Connection Error", str(e))
            
    def start_measurement(self):
        # Validate inputs
        try:
            center_freq = float(self.center_freq_entry.get())
            interval = int(self.interval_entry.get())
            
            # Get telemetry mnemonics
            tm_mnemonics_str = self.tm_mnemonics_entry.get().strip()
            tm_mnemonics = [m.strip() for m in tm_mnemonics_str.split(',') if m.strip()] if tm_mnemonics_str else []
            
            # Get user filename
            user_filename = self.filename_entry.get().strip()
                
            self.is_measuring = True
            self.start_btn.config(state="disabled")
            self.stop_btn.config(state="normal")
            self.connect_btn.config(state="disabled")
            
            self.log("Starting continuous measurement...")
            self.log(f"Center Frequency: {center_freq} MHz")
            self.log(f"Measuring: 32 kHz (always) and 128 kHz (auto-detected from trace)")
            if tm_mnemonics:
                self.log(f"Telemetry Mnemonics: {', '.join(tm_mnemonics)}")
            if user_filename:
                self.log(f"Output filename suffix: {user_filename}")
            
            self.save_settings()  # Save measurement configuration
            
            # Start measurement in separate thread
            self.measurement_thread = threading.Thread(
                target=self.measurement_loop,
                args=(center_freq, tm_mnemonics, user_filename, interval),
                daemon=True
            )
            self.measurement_thread.start()
            
        except ValueError as e:
            messagebox.showerror("Input Error", "Please enter valid numbers")
            
    def stop_measurement(self):
        self.is_measuring = False
        self.log("Stopping measurement...")
        self.stop_btn.config(state="disabled")
        self.start_btn.config(state="normal")
        
    def measurement_loop(self, center_freq, tm_mnemonics, user_filename, interval):
        mi_calculator = modIndex()
        measurement_count = 0
        
        # Initialize Redis connection
        redis_client = None
        try:
            redis_client = redis.from_url("redis://localhost:6379?decode_responses=True")
            redis_client.ping()
            self.log("Connected to Redis for telemetry data")
        except Exception as e:
            self.log(f"Warning: Could not connect to Redis: {str(e)}")
            self.log("Continuing without telemetry data...")
        
        # Create CSV file for this session
        timestamp = datetime.now().strftime("%Y%m%d_%H%M%S")
        # Append user filename if provided
        if user_filename:
            csv_filename = self.data_folder / f"modindex_{timestamp}_{user_filename}.csv"
            csv_detailed_filename = self.data_folder / f"modindex_{timestamp}_{user_filename}_detailed.csv"
        else:
            csv_filename = self.data_folder / f"modindex_{timestamp}.csv"
            csv_detailed_filename = self.data_folder / f"modindex_{timestamp}_detailed.csv"
        
        # Write CSV header - always include both tones, 128kHz will be N/A if not detected
        header = ["Timestamp", "Measurement#", "Center_Freq_MHz", 
                  "32kHz_ModIndex", "32kHz_SB1", "32kHz_SB2",
                  "128kHz_ModIndex", "128kHz_SB1", "128kHz_SB2"]
        # Add telemetry mnemonic columns
        for mnemonic in tm_mnemonics:
            header.append(f"TM_{mnemonic}")
        
        with open(csv_filename, 'w', newline='') as csvfile:
            writer = csv.writer(csvfile)
            writer.writerow(header)
        
        # Create detailed CSV header
        detailed_header = ["Timestamp", "Measurement#", "Iteration#", "Center_Freq_MHz",
                          "32kHz_ModIndex", "32kHz_SB1", "32kHz_SB2",
                          "128kHz_ModIndex", "128kHz_SB1", "128kHz_SB2"]
        # Add telemetry mnemonic columns
        for mnemonic in tm_mnemonics:
            detailed_header.append(f"TM_{mnemonic}")
        
        with open(csv_detailed_filename, 'w', newline='') as csvfile:
            writer = csv.writer(csvfile)
            writer.writerow(detailed_header)
            
        self.log(f"Created data file: {csv_filename.name}")
        self.log(f"Created detailed data file: {csv_detailed_filename.name}")
        
        while self.is_measuring:
            try:
                measurement_count += 1
                self.log(f"\n=== Measurement #{measurement_count} ===")
                
                results = {"timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
                          "measurement": measurement_count,
                          "center_freq": center_freq}
                
                # Perform live SA measurement
                try:
                    if self.sa:
                        # Always try to measure both tones, detection happens from trace data
                        loop = asyncio.new_event_loop()
                        asyncio.set_event_loop(loop)
                        result = loop.run_until_complete(
                            mi_calculator.PMmodIndex(
                                centerFrequency=center_freq,
                                tone1Frequency=32000,
                                tone2Frequency=128000,  # Always try to measure 128kHz
                                SA=self.sa,
                                noOfIterations=3
                            )
                        )
                        loop.close()
                        
                        if result:
                            SC1modIndex, SC2modIndex, TM1sideBands, TM2sideBands, iteration_details = result
                            
                            # Store 32 kHz results
                            results["32khz_modindex"] = SC1modIndex
                            results["32khz_sb1"] = TM1sideBands[0] if TM1sideBands else "N/A"
                            results["32khz_sb2"] = TM1sideBands[1] if TM1sideBands else "N/A"
                            
                            log_msg = f"  32 kHz Mod Index: {SC1modIndex:.3f}"
                            
                            # Store 128 kHz results - automatically detected from trace
                            if SC2modIndex > 0:
                                results["128khz_modindex"] = SC2modIndex
                                results["128khz_sb1"] = TM2sideBands[0] if TM2sideBands else "N/A"
                                results["128khz_sb2"] = TM2sideBands[1] if TM2sideBands else "N/A"
                                log_msg += f", 128 kHz Mod Index: {SC2modIndex:.3f}"
                            else:
                                # 128 kHz not detected in trace
                                results["128khz_modindex"] = "N/A"
                                results["128khz_sb1"] = "N/A"
                                results["128khz_sb2"] = "N/A"
                                log_msg += ", 128 kHz: Not detected in trace"
                            
                            self.log(log_msg)
                    else:
                        self.log("  No SA connected")
                        results["32khz_modindex"] = "N/A"
                        results["32khz_sb1"] = "N/A"
                        results["32khz_sb2"] = "N/A"
                        results["128khz_modindex"] = "N/A"
                        results["128khz_sb1"] = "N/A"
                        results["128khz_sb2"] = "N/A"
                
                except Exception as e:
                    self.log(f"  Error in measurement: {str(e)}")
                    results["32khz_modindex"] = "Error"
                    results["32khz_sb1"] = "Error"
                    results["32khz_sb2"] = "Error"
                    results["128khz_modindex"] = "Error"
                    results["128khz_sb1"] = "Error"
                    results["128khz_sb2"] = "Error"
                
                # Read telemetry data
                if tm_mnemonics and redis_client:
                    try:
                        for mnemonic in tm_mnemonics:
                            try:
                                tm_value = redis_client.hget("TM_MAP", mnemonic.strip().lower())
                                results[f"tm_{mnemonic}"] = tm_value if tm_value is not None else "None"
                                self.log(f"  TM {mnemonic}: {tm_value}")
                            except Exception as e:
                                results[f"tm_{mnemonic}"] = "Error"
                                self.log(f"  Error reading TM {mnemonic}: {str(e)}")
                    except Exception as e:
                        self.log(f"  Error reading telemetry: {str(e)}")
                        for mnemonic in tm_mnemonics:
                            results[f"tm_{mnemonic}"] = "Error"
                elif tm_mnemonics:
                    # Redis not connected, store N/A for all telemetry
                    for mnemonic in tm_mnemonics:
                        results[f"tm_{mnemonic}"] = "N/A"
                
                # Save to CSV
                row = [results["timestamp"], results["measurement"], results["center_freq"],
                       results.get("32khz_modindex", "N/A"),
                       results.get("32khz_sb1", "N/A"),
                       results.get("32khz_sb2", "N/A"),
                       results.get("128khz_modindex", "N/A"),
                       results.get("128khz_sb1", "N/A"),
                       results.get("128khz_sb2", "N/A")]
                
                # Add telemetry data to row
                for mnemonic in tm_mnemonics:
                    row.append(results.get(f"tm_{mnemonic}", "N/A"))
                
                with open(csv_filename, 'a', newline='') as csvfile:
                    writer = csv.writer(csvfile)
                    writer.writerow(row)
                
                # Save detailed iteration data to detailed CSV
                if 'iteration_details' in locals() and iteration_details:
                    with open(csv_detailed_filename, 'a', newline='') as csvfile:
                        writer = csv.writer(csvfile)
                        for iter_data in iteration_details:
                            detailed_row = [
                                results["timestamp"],
                                results["measurement"],
                                iter_data.get('iteration', 'N/A'),
                                results["center_freq"],
                                iter_data.get('32khz_modindex', 'N/A') if iter_data.get('32khz_modindex') is not None else 'N/A',
                                iter_data.get('32khz_sb1', 'N/A') if iter_data.get('32khz_sb1') is not None else 'N/A',
                                iter_data.get('32khz_sb2', 'N/A') if iter_data.get('32khz_sb2') is not None else 'N/A',
                                iter_data.get('128khz_modindex', 'N/A') if iter_data.get('128khz_modindex') is not None else 'N/A',
                                iter_data.get('128khz_sb1', 'N/A') if iter_data.get('128khz_sb1') is not None else 'N/A',
                                iter_data.get('128khz_sb2', 'N/A') if iter_data.get('128khz_sb2') is not None else 'N/A'
                            ]
                            # Add telemetry data to detailed row
                            for mnemonic in tm_mnemonics:
                                detailed_row.append(results.get(f"tm_{mnemonic}", "N/A"))
                            writer.writerow(detailed_row)
                
                # Update results display
                result_text = f"Measurement #{measurement_count}: 32kHz={results.get('32khz_modindex', 'N/A')}  128kHz={results.get('128khz_modindex', 'N/A')}"
                
                self.root.after(0, lambda txt=result_text: self.results_label.config(text=txt))
                
                # Wait for next measurement
                if self.is_measuring:
                    self.log(f"Waiting {interval} seconds until next measurement...")
                    time.sleep(interval)
                    
            except Exception as e:
                self.log(f"Error in measurement loop: {str(e)}")
                import traceback
                self.log(traceback.format_exc())
                time.sleep(interval)
        
        self.log("Measurement stopped")


def run_gui():
    root = tk.Tk()
    app = ModIndexGUI(root)
    root.mainloop()


if __name__ == "__main__":
    run_gui()

