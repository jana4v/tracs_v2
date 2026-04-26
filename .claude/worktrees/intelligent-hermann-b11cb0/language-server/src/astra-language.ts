// ASTRA DSL language definitions
// Mirrors ACSParser.jl DSL_KEYWORDS and statement documentation

export const DSL_KEYWORDS = [
  'TEST_NAME', 'PRE_TEST_REQ', 'SEND', 'SENDTCP',
  'WAIT', 'UNTIL', 'TIMEOUT', 'CHECK', 'WITHIN',
  'EXPECTED', 'ALERT_MSG', 'ABORT_TEST', 'CALL', 'BREAK',
  'IF', 'ELSE', 'END', 'FOR', 'IN', 'TO', 'WHILE',
  'ON_FAIL', 'ON_TIMEOUT', 'AND', 'OR', 'NOT',
] as const

export const BLOCK_OPENERS = ['IF', 'FOR', 'WHILE', 'ON_FAIL', 'ON_TIMEOUT'] as const

export const BLOCK_CLOSERS = ['END'] as const

export const STATEMENT_DOCS: Record<string, string> = {
  TEST_NAME:
    'Declares the procedure name. Must be the first non-comment line.\n\n**Syntax:** `TEST_NAME <name>`\n\n**Example:** `TEST_NAME 4rw-config1`',
  PRE_TEST_REQ:
    'Pre-test requirement. Test aborts if condition is false at start.\n\n**Syntax:** `PRE_TEST_REQ <condition>`\n\n**Example:** `PRE_TEST_REQ TM1.xyz_sts == "on" AND TM1.abc > 20`',
  SEND:
    'Send a telecommand to hardware/simulator.\n\n**Syntax:** `SEND <command> [args...]`\n\n**Example:** `SEND START_RW`',
  SENDTCP:
    'Send command via TCP socket.\n\n**Syntax:** `SENDTCP <host> <port> <data>`\n\n**Example:** `SENDTCP 192.168.1.100 5000 "CMD_START"`',
  WAIT:
    'Pause execution for specified duration, or wait for a condition.\n\n**Syntax:** `WAIT <seconds>` or `WAIT UNTIL <condition> TIMEOUT <seconds>`\n\n**Example:** `WAIT 5` or `WAIT UNTIL TM1.STATUS == "OK" TIMEOUT 30`',
  CHECK:
    'Verify a telemetry condition.\n\n**Syntax:** `CHECK <condition> [WITHIN <seconds>]`\n\n**Example:** `CHECK TM1.RW_SPEED <= 100 WITHIN 10`',
  EXPECTED:
    'Assert expected telemetry state.\n\n**Syntax:** `EXPECTED <condition>`\n\n**Example:** `EXPECTED TM1.RW_MODE == "NOMINAL"`',
  ALERT_MSG:
    'Display an alert message to the operator.\n\n**Syntax:** `ALERT_MSG "<message>"`\n\n**Example:** `ALERT_MSG "Voltage too low"`',
  ABORT_TEST:
    'Immediately stop test execution.\n\n**Syntax:** `ABORT_TEST`',
  CALL:
    'Execute another loaded procedure.\n\n**Syntax:** `CALL <procedure_name>`\n\n**Example:** `CALL 4rw-config1`',
  BREAK:
    'Exit the current FOR or WHILE loop.\n\n**Syntax:** `BREAK`',
  IF:
    'Conditional block.\n\n**Syntax:**\n```\nIF <condition>\n    ...\nELSE\n    ...\nEND\n```',
  ELSE:
    'Alternative branch in an IF block.\n\n**Syntax:** `ELSE` (must be inside an IF block)',
  END:
    'Closes an IF, FOR, WHILE, ON_FAIL, or ON_TIMEOUT block.\n\n**Syntax:** `END`',
  FOR:
    'Loop block with counter variable.\n\n**Syntax:**\n```\nFOR <var> IN <start> TO <end>\n    ...\nEND\n```\n\n**Example:** `FOR i IN 1 TO 5`',
  WHILE:
    'Conditional loop.\n\n**Syntax:**\n```\nWHILE <condition>\n    ...\nEND\n```',
  ON_FAIL:
    'Error handler block. Executes if the test encounters a failure.\n\n**Syntax:**\n```\nON_FAIL\n    ...\nEND\n```',
  ON_TIMEOUT:
    'Timeout handler block. Executes if a WAIT UNTIL times out.\n\n**Syntax:**\n```\nON_TIMEOUT\n    ...\nEND\n```',
  AND: 'Logical AND operator for combining conditions.',
  OR: 'Logical OR operator for combining conditions.',
  NOT: 'Logical NOT operator for negating a condition.',
  IN: 'Used in FOR loops: `FOR i IN 1 TO 10`',
  TO: 'Used in FOR loops: `FOR i IN 1 TO 10`',
  UNTIL: 'Used with WAIT: `WAIT UNTIL <condition> TIMEOUT <seconds>`',
  TIMEOUT: 'Used with WAIT UNTIL: specifies maximum wait time.',
  WITHIN: 'Used with CHECK: `CHECK <condition> WITHIN <seconds>`',
}

export const SNIPPET_TEMPLATES: Record<string, { label: string; insertText: string; documentation: string }> = {
  IF: {
    label: 'IF...END',
    insertText: 'IF ${1:condition}\n    ${2}\nEND',
    documentation: 'Conditional block',
  },
  IF_ELSE: {
    label: 'IF...ELSE...END',
    insertText: 'IF ${1:condition}\n    ${2}\nELSE\n    ${3}\nEND',
    documentation: 'Conditional block with else branch',
  },
  FOR: {
    label: 'FOR...END',
    insertText: 'FOR ${1:i} IN ${2:1} TO ${3:10}\n    ${4}\nEND',
    documentation: 'Loop with counter variable',
  },
  WHILE: {
    label: 'WHILE...END',
    insertText: 'WHILE ${1:condition}\n    ${2}\nEND',
    documentation: 'Conditional loop',
  },
  ON_FAIL: {
    label: 'ON_FAIL...END',
    insertText: 'ON_FAIL\n    ${1:ALERT_MSG "Error occurred"}\n    ${2:ABORT_TEST}\nEND',
    documentation: 'Error handler block',
  },
  ON_TIMEOUT: {
    label: 'ON_TIMEOUT...END',
    insertText: 'ON_TIMEOUT\n    ${1:ALERT_MSG "Timeout occurred"}\nEND',
    documentation: 'Timeout handler block',
  },
  CHECK_WITHIN: {
    label: 'CHECK...WITHIN',
    insertText: 'CHECK ${1:TM1.param} ${2:==} ${3:value} WITHIN ${4:10}',
    documentation: 'Verify telemetry condition within timeout',
  },
  WAIT_UNTIL: {
    label: 'WAIT UNTIL...TIMEOUT',
    insertText: 'WAIT UNTIL ${1:TM1.param == "value"} TIMEOUT ${2:30}',
    documentation: 'Wait for condition with timeout',
  },
}
