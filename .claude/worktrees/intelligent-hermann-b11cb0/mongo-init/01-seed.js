// ASTRA MongoDB Seed Data
// Initializes TM/TC/SCO mnemonics and sample procedures

db = db.getSiblingDB('astra');

// ============================================================
// TM Mnemonics - Telemetry parameter definitions
// ============================================================
db.tm_mnemonics.drop();
db.tm_mnemonics.insertMany([
  // TM Bank 1 - Power & ADCS
  { bank: 1, mnemonic: "xyz_sts",     full_ref: "TM1.xyz_sts",     description: "XYZ axis status flag",        data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["on", "off"],           subsystem: "ADCS" },
  { bank: 1, mnemonic: "abc",         full_ref: "TM1.abc",         description: "ABC sensor value",             data_type: "number",  unit: "",    range_min: 0,    range_max: 100,  enum_values: [],                      subsystem: "ADCS" },
  { bank: 1, mnemonic: "voltage_bus", full_ref: "TM1.voltage_bus", description: "Main bus voltage",             data_type: "number",  unit: "V",   range_min: 20,   range_max: 35,   enum_values: [],                      subsystem: "POWER" },
  { bank: 1, mnemonic: "VOLT",        full_ref: "TM1.VOLT",        description: "Bus voltage (alias)",          data_type: "number",  unit: "V",   range_min: 20,   range_max: 35,   enum_values: [],                      subsystem: "POWER" },
  { bank: 1, mnemonic: "STATUS",      full_ref: "TM1.STATUS",      description: "System status",                data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["OK", "ERROR", "INIT"], subsystem: "OBC" },
  { bank: 1, mnemonic: "RW_STATUS",   full_ref: "TM1.RW_STATUS",   description: "Reaction wheel status",        data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["READY", "RUNNING", "STOPPED", "ERROR"], subsystem: "ADCS" },
  { bank: 1, mnemonic: "RW_SPEED",    full_ref: "TM1.RW_SPEED",    description: "Reaction wheel speed",         data_type: "number",  unit: "RPM", range_min: 0,    range_max: 6000, enum_values: [],                      subsystem: "ADCS" },
  { bank: 1, mnemonic: "RW_MODE",     full_ref: "TM1.RW_MODE",     description: "Reaction wheel operating mode",data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["NOMINAL", "SAFE", "TEST"], subsystem: "ADCS" },

  // TM Bank 2 - Reaction Wheels (detailed)
  { bank: 2, mnemonic: "rw_speed",    full_ref: "TM2.rw_speed",    description: "Reaction wheel speed (bank 2)", data_type: "number",  unit: "RPM", range_min: 0,    range_max: 6000, enum_values: [],                      subsystem: "ADCS" },
  { bank: 2, mnemonic: "xyz_sts",     full_ref: "TM2.xyz_sts",     description: "XYZ status (bank 2)",           data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["on", "off"],           subsystem: "ADCS" },
  { bank: 2, mnemonic: "STATUS",      full_ref: "TM2.STATUS",      description: "Bank 2 system status",          data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["OK", "ERROR", "INIT"], subsystem: "OBC" },
  { bank: 2, mnemonic: "RW_STATUS",   full_ref: "TM2.RW_STATUS",   description: "Reaction wheel status (bank 2)",data_type: "string",  unit: "",    range_min: null, range_max: null, enum_values: ["READY", "RUNNING", "STOPPED", "ERROR"], subsystem: "ADCS" },

  // TM Bank 3 - Thermal
  { bank: 3, mnemonic: "TEMP_PANEL1", full_ref: "TM3.TEMP_PANEL1", description: "Solar panel 1 temperature",    data_type: "number",  unit: "degC", range_min: -40,  range_max: 85,   enum_values: [],                     subsystem: "THERMAL" },
  { bank: 3, mnemonic: "TEMP_PANEL2", full_ref: "TM3.TEMP_PANEL2", description: "Solar panel 2 temperature",    data_type: "number",  unit: "degC", range_min: -40,  range_max: 85,   enum_values: [],                     subsystem: "THERMAL" },
  { bank: 3, mnemonic: "TEMP_BATT",   full_ref: "TM3.TEMP_BATT",   description: "Battery temperature",          data_type: "number",  unit: "degC", range_min: -10,  range_max: 45,   enum_values: [],                     subsystem: "THERMAL" },
  { bank: 3, mnemonic: "HEATER_STS",  full_ref: "TM3.HEATER_STS",  description: "Heater status",                data_type: "string",  unit: "",     range_min: null, range_max: null, enum_values: ["ON", "OFF", "AUTO"],  subsystem: "THERMAL" },
]);

// Create indexes for tm_mnemonics
db.tm_mnemonics.createIndex({ bank: 1, mnemonic: 1 }, { unique: true });
db.tm_mnemonics.createIndex({ full_ref: 1 });
db.tm_mnemonics.createIndex({ subsystem: 1 });

// ============================================================
// TC Mnemonics - Telecommand definitions
// ============================================================
db.tc_mnemonics.drop();
db.tc_mnemonics.insertMany([
  { command: "START_RW",     full_ref: "TC.START_RW",     description: "Start reaction wheel motor",          parameters: [],                                                              subsystem: "ADCS",  category: "reaction-wheel" },
  { command: "STOP_RW",      full_ref: "TC.STOP_RW",      description: "Stop reaction wheel motor",           parameters: [],                                                              subsystem: "ADCS",  category: "reaction-wheel" },
  { command: "RAMP_RW",      full_ref: "TC.RAMP_RW",      description: "Ramp reaction wheel speed",           parameters: [{ name: "speed", type: "number", required: true, default: 100 }], subsystem: "ADCS",  category: "reaction-wheel" },
  { command: "CONFIGURE",    full_ref: "TC.CONFIGURE",     description: "Configure subsystem parameters",      parameters: [{ name: "target", type: "string", required: true }],             subsystem: "OBC",   category: "configuration" },
  { command: "SET_MODE",     full_ref: "TC.SET_MODE",      description: "Set operating mode",                  parameters: [{ name: "mode", type: "string", required: true }],               subsystem: "OBC",   category: "operations" },
  { command: "ENABLE_HTR",   full_ref: "TC.ENABLE_HTR",    description: "Enable heater",                       parameters: [{ name: "heater_id", type: "number", required: true }],          subsystem: "THERMAL", category: "thermal" },
  { command: "DISABLE_HTR",  full_ref: "TC.DISABLE_HTR",   description: "Disable heater",                      parameters: [{ name: "heater_id", type: "number", required: true }],          subsystem: "THERMAL", category: "thermal" },
  { command: "DEPLOY_PANEL", full_ref: "TC.DEPLOY_PANEL",  description: "Deploy solar panel",                  parameters: [{ name: "panel_id", type: "number", required: true }],           subsystem: "POWER", category: "deployment" },
  { command: "RESET_SUBSYS", full_ref: "TC.RESET_SUBSYS",  description: "Reset a subsystem",                   parameters: [{ name: "subsystem", type: "string", required: true }],          subsystem: "OBC",   category: "operations" },
]);

// Create indexes for tc_mnemonics
db.tc_mnemonics.createIndex({ command: 1 }, { unique: true });
db.tc_mnemonics.createIndex({ full_ref: 1 });
db.tc_mnemonics.createIndex({ subsystem: 1 });

// ============================================================
// SCO Commands - Spacecraft Operations
// ============================================================
db.sco_commands.drop();
db.sco_commands.insertMany([
  { command: "SAFE_MODE",      full_ref: "SCO.SAFE_MODE",      description: "Enter spacecraft safe mode",       parameters: [],                                                    subsystem: "OBC",   category: "operations" },
  { command: "REBOOT",         full_ref: "SCO.REBOOT",         description: "Reboot spacecraft on-board computer", parameters: [],                                                 subsystem: "OBC",   category: "operations" },
  { command: "DEPLOY_SOLAR",   full_ref: "SCO.DEPLOY_SOLAR",   description: "Deploy all solar panels",          parameters: [],                                                    subsystem: "POWER", category: "deployment" },
  { command: "DETUMBLE",       full_ref: "SCO.DETUMBLE",       description: "Start detumbling mode",            parameters: [],                                                    subsystem: "ADCS",  category: "stabilization" },
  { command: "POINT_NADIR",    full_ref: "SCO.POINT_NADIR",    description: "Point to nadir (Earth-facing)",    parameters: [],                                                    subsystem: "ADCS",  category: "pointing" },
  { command: "POWER_CYCLE",    full_ref: "SCO.POWER_CYCLE",    description: "Power cycle a subsystem",          parameters: [{ name: "subsystem", type: "string", required: true }], subsystem: "POWER", category: "operations" },
  { command: "COMM_ENABLE",    full_ref: "SCO.COMM_ENABLE",    description: "Enable communication subsystem",   parameters: [],                                                    subsystem: "COMM",  category: "communication" },
  { command: "COMM_DISABLE",   full_ref: "SCO.COMM_DISABLE",   description: "Disable communication subsystem",  parameters: [],                                                    subsystem: "COMM",  category: "communication" },
]);

// Create indexes for sco_commands
db.sco_commands.createIndex({ command: 1 }, { unique: true });
db.sco_commands.createIndex({ full_ref: 1 });
db.sco_commands.createIndex({ subsystem: 1 });

// ============================================================
// Procedures - Sample test procedures
// ============================================================
db.procedures.drop();
db.procedures.insertMany([
  {
    name: "4rw-config1",
    content: 'TEST_NAME 4rw-config1\nPRE_TEST_REQ TM1.xyz_sts == "on" AND TM1.abc > 20\nSEND START_RW\nWAIT 5\nCHECK TM1.RW_STATUS == "READY"\n\nIF TM1.VOLT > 5\n    SEND START_RW\nELSE\n    ALERT_MSG "Voltage too low"\n    ABORT_TEST\nEND',
    description: "Basic reaction wheel configuration test",
    category: "reaction-wheel",
    version: 1,
    created_at: new Date(),
    updated_at: new Date()
  },
  {
    name: "4rw-config3",
    content: 'TEST_NAME 4rw-config3\n# This test calls sub-procedures and demonstrates advanced features\n\n# Call sub-procedures\nCALL 4rw-config1\n\n# Inline Julia computation\nadjusted = TM1.abc + 10\nprintln("Adjusted value: ", adjusted)\n\n# Conditional execution based on computed value\nIF adjusted > 50\n    SEND START_RW\nELSE\n    ALERT_MSG "Value too low"\nEND\n\n# Loop to ramp up reaction wheels\nFOR i IN 1 TO 3\n    SEND RAMP_RW_$(i)\n    WAIT 2\nEND\n\n# Verify final state\nCHECK TM1.RW_SPEED <= 100 WITHIN 10\n\n# Expected telemetry state\nEXPECTED TM1.RW_MODE == "NOMINAL"\n\n# Error handler\nON_FAIL\n    ALERT_MSG "RW Configuration failed"\n    ABORT_TEST\nEND',
    description: "Advanced RW config test with sub-procedure calls, loops, inline Julia, and error handling",
    category: "reaction-wheel",
    version: 1,
    created_at: new Date(),
    updated_at: new Date()
  }
]);

// Create indexes for procedures
db.procedures.createIndex({ name: 1 }, { unique: true });
db.procedures.createIndex({ category: 1 });

// ============================================================
// Test Results - Empty collection with indexes
// ============================================================
db.test_results.drop();
db.createCollection("test_results");
db.test_results.createIndex({ test_name: 1 });
db.test_results.createIndex({ started_at: -1 });
db.test_results.createIndex({ status: 1 });

// ============================================================
// TM History - Empty collection with TTL index
// ============================================================
db.tm_history.drop();
db.createCollection("tm_history");
db.tm_history.createIndex({ timestamp: -1 });
db.tm_history.createIndex({ timestamp: 1 }, { expireAfterSeconds: 604800 }); // 7-day TTL

// ============================================================
// Versioned Procedures - Procedures with version history
// ============================================================
db.versioned_procedures.drop();
db.versioned_procedures.insertMany([
  {
    test_name: "4rw-config1",
    versions: [
      {
        version: 1,
        content: 'TEST_NAME 4rw-config1\nPRE_TEST_REQ TM1.xyz_sts == "on" AND TM1.abc > 20\nSEND START_RW\nWAIT 5\nCHECK TM1.RW_STATUS == "READY"\n\nIF TM1.VOLT > 5\n    SEND START_RW\nELSE\n    ALERT_MSG "Voltage too low"\n    ABORT_TEST\nEND',
        project: "gsat7r",
        created_by: "user1",
        created_at: new Date("2026-02-10T10:00:00Z")
      },
      {
        version: 2,
        content: 'TEST_NAME 4rw-config1\nPRE_TEST_REQ TM1.xyz_sts == "on" AND TM1.abc > 20 AND TM1.voltage_bus > 24\nSEND START_RW\nWAIT 5\nCHECK TM1.RW_STATUS == "READY" WITHIN 10\n\nIF TM1.VOLT > 5\n    SEND START_RW\n    WAIT 2\n    CHECK TM1.RW_SPEED > 0 WITHIN 5\nELSE\n    ALERT_MSG "Voltage too low"\n    ABORT_TEST\nEND\n\nEXPECTED TM1.RW_MODE == "NOMINAL"',
        project: "gsat7r",
        created_by: "user2",
        created_at: new Date("2026-02-11T12:00:00Z")
      }
    ]
  },
  {
    test_name: "4rw-config3",
    versions: [
      {
        version: 1,
        content: 'TEST_NAME 4rw-config3\n# Advanced RW config test\n\nCALL 4rw-config1\n\nadjusted = TM1.abc + 10\nprintln("Adjusted value: ", adjusted)\n\nIF adjusted > 50\n    SEND START_RW\nELSE\n    ALERT_MSG "Value too low"\nEND\n\nFOR i IN 1 TO 3\n    SEND RAMP_RW\n    WAIT 2\nEND\n\nCHECK TM1.RW_SPEED <= 100 WITHIN 10\nEXPECTED TM1.RW_MODE == "NOMINAL"\n\nON_FAIL\n    ALERT_MSG "RW Configuration failed"\n    ABORT_TEST\nEND',
        project: "gsat7r",
        created_by: "user1",
        created_at: new Date("2026-02-10T14:00:00Z")
      }
    ]
  },
  {
    test_name: "thermal-check-1",
    versions: [
      {
        version: 1,
        content: 'TEST_NAME thermal-check-1\nPRE_TEST_REQ TM3.HEATER_STS == "ON"\n\nCHECK TM3.TEMP_PANEL1 > -20 WITHIN 30\nCHECK TM3.TEMP_PANEL2 > -20 WITHIN 30\nCHECK TM3.TEMP_BATT > 0 WITHIN 30\n\nIF TM3.TEMP_BATT < 5\n    SEND ENABLE_HTR\n    WAIT 10\n    CHECK TM3.TEMP_BATT > 5 WITHIN 60\nEND\n\nEXPECTED TM3.HEATER_STS == "AUTO"',
        project: "gsat7r",
        created_by: "user1",
        created_at: new Date("2026-02-09T08:00:00Z")
      },
      {
        version: 2,
        content: 'TEST_NAME thermal-check-1\nPRE_TEST_REQ TM3.HEATER_STS == "ON" AND TM1.STATUS == "OK"\n\n# Check all panel temperatures\nCHECK TM3.TEMP_PANEL1 > -20 WITHIN 30\nCHECK TM3.TEMP_PANEL2 > -20 WITHIN 30\nCHECK TM3.TEMP_BATT > 0 WITHIN 30\n\n# Battery thermal protection\nIF TM3.TEMP_BATT < 5\n    ALERT_MSG "Battery temp low, enabling heater"\n    SEND ENABLE_HTR\n    WAIT 10\n    CHECK TM3.TEMP_BATT > 5 WITHIN 60\nEND\n\nEXPECTED TM3.HEATER_STS == "AUTO"\n\nON_FAIL\n    ALERT_MSG "Thermal check failed"\nEND',
        project: "gsat7r",
        created_by: "user2",
        created_at: new Date("2026-02-10T16:00:00Z")
      }
    ]
  },
  {
    test_name: "comm-link-test",
    versions: [
      {
        version: 1,
        content: 'TEST_NAME comm-link-test\nPRE_TEST_REQ TM1.STATUS == "OK"\n\nSEND COMM_ENABLE\nWAIT 5\n\nSENDTCP 192.168.1.100 5000 "PING"\nWAIT 2\n\nCHECK TM1.STATUS == "OK" WITHIN 10\nALERT_MSG "Communication link established"',
        project: "sat-x1",
        created_by: "user3",
        created_at: new Date("2026-02-11T09:00:00Z")
      }
    ]
  }
]);

// Create indexes for versioned_procedures
db.versioned_procedures.createIndex({ test_name: 1 }, { unique: true });
db.versioned_procedures.createIndex({ "versions.project": 1 });
db.versioned_procedures.createIndex({ "versions.created_by": 1 });

print("ASTRA seed data initialized successfully.");
print("Collections: tm_mnemonics, tc_mnemonics, sco_commands, procedures, test_results, tm_history, versioned_procedures");
