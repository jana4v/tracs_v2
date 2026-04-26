#!/usr/bin/env julia
"""
Seed script: Insert sample test procedures into MongoDB via ASTRA API.
Run with: julia --project=backend backend/seed_procedures.jl
"""

using HTTP, JSON3

const API = "http://localhost:8080/api/v1"

function seed_procedure(test_name::String, content::String; project::String="SAT-1", created_by::String="ASTRA-Seed")
    body = JSON3.write(Dict(
        "test_name" => test_name,
        "content" => content,
        "project" => project,
        "created_by" => created_by,
    ))
    resp = HTTP.post("$API/procedures", ["Content-Type" => "application/json"], body)
    result = JSON3.read(String(resp.body))
    if get(result, :saved, false)
        println("  ✓ $test_name (v$(result.version))")
    else
        println("  - $test_name ($(get(result, :reason, "skipped")))")
    end
end

println("Seeding ASTRA test procedures...")
println("=" ^ 50)

# 1. Reaction Wheel Configuration
seed_procedure("rw-config-nominal", """
TEST_NAME rw-config-nominal
# Reaction Wheel nominal configuration test
PRE_TEST_REQ TM1.PWR_BUS == "ON" AND TM1.RW_PWR > 0

SEND RW_INIT
WAIT 3
CHECK TM1.RW_STATUS == "IDLE"

SEND RW_SET_SPEED 1000
WAIT 5
CHECK TM1.RW_SPEED >= 950 WITHIN 10

SEND RW_SET_SPEED 2000
WAIT 5
CHECK TM1.RW_SPEED >= 1950 WITHIN 10

EXPECTED TM1.RW_MODE == "NOMINAL"
""")

# 2. Solar Array Deployment
seed_procedure("sa-deploy-seq", """
TEST_NAME sa-deploy-seq
# Solar Array deployment sequence verification
PRE_TEST_REQ TM1.SA_LOCK == "ENGAGED" AND TM1.PWR_BUS == "ON"

SEND SA_UNLOCK
WAIT 2
CHECK TM1.SA_LOCK == "DISENGAGED"

SEND SA_DEPLOY_PRI
WAIT 10
CHECK TM1.SA_PRI_ANGLE >= 85 WITHIN 30

SEND SA_DEPLOY_SEC
WAIT 10
CHECK TM1.SA_SEC_ANGLE >= 85 WITHIN 30

CHECK TM1.SA_CURRENT > 0.5
EXPECTED TM1.SA_STATUS == "DEPLOYED"

ON_FAIL
    ALERT_MSG "Solar array deployment failed"
    SEND SA_SAFE_MODE
END
""")

# 3. Star Tracker Calibration
seed_procedure("str-calibration", """
TEST_NAME str-calibration
# Star Tracker calibration and alignment test
PRE_TEST_REQ TM1.STR_PWR == "ON"

SEND STR_INIT
WAIT 5
CHECK TM1.STR_STATUS == "READY"

SEND STR_START_CAL
WAIT 15
CHECK TM1.STR_CAL_STATUS == "IN_PROGRESS" WITHIN 5

WAIT 30
CHECK TM1.STR_CAL_STATUS == "COMPLETE" WITHIN 60

CHECK TM1.STR_ACCURACY <= 0.01
CHECK TM1.STR_QUATERNION_VALID == 1

EXPECTED TM1.STR_MODE == "TRACKING"
""")

# 4. Thruster Firing Test
seed_procedure("thr-fire-test", """
TEST_NAME thr-fire-test
# Thruster firing test sequence
PRE_TEST_REQ TM1.PROP_PRESS > 200 AND TM1.THR_ARMED == "YES"

SEND THR_PREHEAT
WAIT 10
CHECK TM1.THR_TEMP >= 50 WITHIN 15

FOR i IN 1 TO 4
    SEND THR_FIRE_\$(i) 100
    WAIT 2
    CHECK TM1.THR_\$(i)_STATUS == "FIRED"
END

CHECK TM1.PROP_PRESS > 180
SEND THR_SAFE

EXPECTED TM1.THR_ARMED == "NO"

ON_FAIL
    ALERT_MSG "Thruster test anomaly detected"
    SEND THR_SAFE
    ABORT_TEST
END
""")

# 5. Thermal Control Validation
seed_procedure("thermal-ctrl-val", """
TEST_NAME thermal-ctrl-val
# Thermal control system validation
PRE_TEST_REQ TM1.HTR_PWR == "ON"

SEND HTR_ENABLE_ZONE_1
WAIT 5
CHECK TM1.ZONE1_TEMP >= 15 WITHIN 30

SEND HTR_ENABLE_ZONE_2
WAIT 5
CHECK TM1.ZONE2_TEMP >= 15 WITHIN 30

IF TM1.ZONE1_TEMP > 45
    SEND HTR_DISABLE_ZONE_1
    ALERT_MSG "Zone 1 overtemp, heater disabled"
END

IF TM1.ZONE2_TEMP > 45
    SEND HTR_DISABLE_ZONE_2
    ALERT_MSG "Zone 2 overtemp, heater disabled"
END

CHECK TM1.MLI_STATUS == "NOMINAL"
EXPECTED TM1.THERMAL_MODE == "AUTO"
""")

# 6. Communication Link Test
seed_procedure("comm-link-test", """
TEST_NAME comm-link-test
# S-Band communication link establishment test
PRE_TEST_REQ TM1.COMM_PWR == "ON" AND TM1.ANT_DEPLOYED == 1

SEND COMM_SET_FREQ 2250.5
WAIT 2
CHECK TM1.COMM_FREQ == 2250.5

SEND COMM_SET_POWER 5
WAIT 2
SEND COMM_TX_ENABLE

WAIT 10
CHECK TM1.COMM_LOCK == "ACQUIRED" WITHIN 30
CHECK TM1.COMM_SNR >= 10
CHECK TM1.COMM_BER <= 0.001

SEND COMM_SEND_BEACON
WAIT 5
CHECK TM1.BEACON_ACK == 1 WITHIN 15

EXPECTED TM1.COMM_STATUS == "LINKED"
""")

# 7. AOCS Mode Transition
seed_procedure("aocs-mode-trans", """
TEST_NAME aocs-mode-trans
# Attitude and Orbit Control mode transition test
PRE_TEST_REQ TM1.AOCS_PWR == "ON" AND TM1.STR_MODE == "TRACKING"

# Safe mode to detumble
SEND AOCS_SET_MODE DETUMBLE
WAIT 5
CHECK TM1.AOCS_MODE == "DETUMBLE" WITHIN 10
CHECK TM1.BODY_RATE <= 2.0 WITHIN 60

# Detumble to sun pointing
SEND AOCS_SET_MODE SUN_POINT
WAIT 10
CHECK TM1.AOCS_MODE == "SUN_POINT" WITHIN 15
CHECK TM1.SUN_ANGLE <= 5.0 WITHIN 120

# Sun pointing to nominal
SEND AOCS_SET_MODE NOMINAL
WAIT 10
CHECK TM1.AOCS_MODE == "NOMINAL" WITHIN 30

EXPECTED TM1.POINTING_ERROR <= 0.1

ON_FAIL
    ALERT_MSG "AOCS mode transition failure"
    SEND AOCS_SET_MODE SAFE
END
""")

# 8. Power Bus Regulation
seed_procedure("pwr-bus-reg", """
TEST_NAME pwr-bus-reg
# Power bus regulation and battery charge test
PRE_TEST_REQ TM1.PWR_BUS == "ON"

CHECK TM1.BUS_VOLTAGE >= 27.5
CHECK TM1.BUS_VOLTAGE <= 32.5
CHECK TM1.BUS_CURRENT > 0

SEND PWR_ENABLE_LOAD_1
WAIT 3
CHECK TM1.BUS_VOLTAGE >= 27.0

SEND PWR_ENABLE_LOAD_2
WAIT 3
CHECK TM1.BUS_VOLTAGE >= 26.5

SEND PWR_ENABLE_LOAD_3
WAIT 3
CHECK TM1.BUS_VOLTAGE >= 26.0

CHECK TM1.BAT_SOC >= 50
CHECK TM1.BAT_TEMP >= 5
CHECK TM1.BAT_TEMP <= 40

EXPECTED TM1.PWR_STATUS == "REGULATED"
""")

# 9. Payload Data Handling
seed_procedure("pdh-data-acq", """
TEST_NAME pdh-data-acq
# Payload data handling and storage test
PRE_TEST_REQ TM1.PDH_PWR == "ON" AND TM1.MEM_AVAIL > 1024

SEND PDH_INIT
WAIT 3
CHECK TM1.PDH_STATUS == "READY"

SEND PDH_START_ACQ
WAIT 5
CHECK TM1.PDH_MODE == "ACQUIRING" WITHIN 10

WAIT 20
SEND PDH_STOP_ACQ

CHECK TM1.PDH_FRAMES_STORED > 0
CHECK TM1.MEM_USED > 0
CHECK TM1.PDH_CRC_ERRORS == 0

SEND PDH_PLAYBACK
WAIT 10
CHECK TM1.PDH_PLAYBACK_STATUS == "COMPLETE" WITHIN 30

EXPECTED TM1.PDH_STATUS == "IDLE"
""")

# 10. GPS Receiver Test
seed_procedure("gps-receiver-test", """
TEST_NAME gps-receiver-test
# GPS receiver acquisition and position fix test
PRE_TEST_REQ TM1.GPS_PWR == "ON"

SEND GPS_COLD_START
WAIT 5
CHECK TM1.GPS_STATUS == "SEARCHING" WITHIN 10

WAIT 60
CHECK TM1.GPS_SATS_VISIBLE >= 4 WITHIN 120
CHECK TM1.GPS_FIX == "3D" WITHIN 180

CHECK TM1.GPS_PDOP <= 6.0
CHECK TM1.GPS_POS_ERR <= 25.0

EXPECTED TM1.GPS_STATUS == "TRACKING"

ON_FAIL
    ALERT_MSG "GPS acquisition failed"
END
""")

# 11. Magnetometer Calibration
seed_procedure("mag-cal-sequence", """
TEST_NAME mag-cal-sequence
# Magnetometer calibration with rotation sequence
PRE_TEST_REQ TM1.MAG_PWR == "ON" AND TM1.AOCS_MODE == "NOMINAL"

SEND MAG_START_CAL
WAIT 3
CHECK TM1.MAG_CAL_STATUS == "IN_PROGRESS"

# X-axis rotation
SEND AOCS_ROTATE X 360
WAIT 30
CHECK TM1.MAG_X_CAL == "DONE" WITHIN 45

# Y-axis rotation
SEND AOCS_ROTATE Y 360
WAIT 30
CHECK TM1.MAG_Y_CAL == "DONE" WITHIN 45

# Z-axis rotation
SEND AOCS_ROTATE Z 360
WAIT 30
CHECK TM1.MAG_Z_CAL == "DONE" WITHIN 45

CHECK TM1.MAG_RESIDUAL <= 0.05
EXPECTED TM1.MAG_CAL_STATUS == "COMPLETE"
""")

# 12. Full System Health Check
seed_procedure("sys-health-check", """
TEST_NAME sys-health-check
# Comprehensive system health check
PRE_TEST_REQ TM1.PWR_BUS == "ON"

# Power subsystem
CHECK TM1.BUS_VOLTAGE >= 27.0
CHECK TM1.BAT_SOC >= 30

# Thermal
CHECK TM1.ZONE1_TEMP >= -10
CHECK TM1.ZONE1_TEMP <= 50
CHECK TM1.ZONE2_TEMP >= -10
CHECK TM1.ZONE2_TEMP <= 50

# AOCS
CHECK TM1.AOCS_PWR == "ON"
CHECK TM1.BODY_RATE <= 5.0

# Communications
CHECK TM1.COMM_PWR == "ON"

# Payload
CHECK TM1.MEM_AVAIL > 512

# Propulsion
CHECK TM1.PROP_PRESS > 100

EXPECTED TM1.SYS_STATUS == "HEALTHY"
""")

println()
println("=" ^ 50)
println("Seeding complete!")
