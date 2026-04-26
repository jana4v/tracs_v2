# ASTRA Genie.jl Development Configuration

using Genie

# Server configuration
Genie.config.run_as_server = true
Genie.config.server_port = 8080

# CORS configuration
Genie.config.cors_headers["Access-Control-Allow-Origin"] = "*"
Genie.config.cors_headers["Access-Control-Allow-Methods"] = "GET, POST, DELETE, OPTIONS"
Genie.config.cors_headers["Access-Control-Allow-Headers"] = "Content-Type"

# WebSocket configuration
Genie.config.websockets_server = true

# Logging
Genie.config.log_level = :info
Genie.config.log_to_file = false
