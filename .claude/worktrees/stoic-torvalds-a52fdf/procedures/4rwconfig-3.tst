TEST_NAME 4rw-config3
# This test calls sub-procedures and demonstrates advanced features

# Call sub-procedures
CALL 4rw-config1

# Inline Julia computation
adjusted = TM1.abc + 10
println("Adjusted value: ", adjusted)

# Conditional execution based on computed value
IF adjusted > 50
    SEND START_RW
ELSE
    ALERT_MSG "Value too low"
END

# Loop to ramp up reaction wheels
FOR i IN 1 TO 3
    SEND RAMP_RW_$(i)
    WAIT 2
END

# Verify final state
CHECK TM1.RW_SPEED <= 100 WITHIN 10

# Expected telemetry state
EXPECTED TM1.RW_MODE == "NOMINAL"

# Error handler
ON_FAIL
    ALERT_MSG "RW Configuration failed"
    ABORT_TEST
END
