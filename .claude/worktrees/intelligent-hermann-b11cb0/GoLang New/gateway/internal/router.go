package gateway

import (
	"log/slog"

	"github.com/go-chi/chi/v5"
	"github.com/mainframe/tm-system/internal/clients"
	"github.com/mainframe/tm-system/internal/repository/sqlite"
	"github.com/redis/go-redis/v9"
)

// RegisterRoutes mounts all gateway API handlers on the given chi.Router.
// ingestSimURL is the base URL of the ingest service sim API (e.g. "http://localhost:8082").
func RegisterRoutes(r chi.Router, rdb *redis.Client, sdb *clients.SQLiteDB, ingestSimURL string, logger *slog.Logger) {
	// Create repository implementations from the shared SQLiteDB.
	tmRepo := sqlite.NewTMMnemonicRepo(sdb, logger)
	tcRepo := sqlite.NewTCMnemonicRepo(sdb, logger)
	scoRepo := sqlite.NewSCOCommandRepo(sdb, logger)
	dtmRepo := sqlite.NewDTMRepo(sdb, logger)
	udtmRepo := sqlite.NewUDTMRepo(sdb, logger)
	spasdacsRepo := sqlite.NewSpasdacsRepo(sdb, logger)

	// Instantiate handlers using store interfaces.
	telemetry := &TelemetryHandler{ingestSimURL: ingestSimURL, logger: logger}
	chains := &ChainsHandler{rdb: rdb, logger: logger}
	limits := &LimitsHandler{rdb: rdb, logger: logger}
	simulator := &SimulatorHandler{rdb: rdb, logger: logger}
	udtm := &UDTMHandler{rdb: rdb, logger: logger}
	dtm := &DTMHandler{rdb: rdb, logger: logger}
	mnemonics := &MnemonicsHandler{tm: tmRepo, tc: tcRepo, sco: scoRepo, rdb: rdb, logger: logger}
	udtmCrud := &UDTMCrudHandler{udtm: udtmRepo, tm: tmRepo, rdb: rdb, logger: logger}
	dtmCrud := &DTMCrudHandler{dtm: dtmRepo, tm: tmRepo, rdb: rdb, logger: logger}
	maps := &MapsHandler{rdb: rdb, logger: logger}
	redisHash := &RedisHashHandler{rdb: rdb, logger: logger}
	spasdacs := &SpasdacsHandler{store: spasdacsRepo, logger: logger}
	tmUpload := &TelemetryUploadHandler{tm: tmRepo, rdb: rdb, logger: logger}
	tcUpload := &TelecommandUploadHandler{tc: tcRepo, rdb: rdb, logger: logger}
	simProxy := &SimProxyHandler{ingestSimURL: ingestSimURL, logger: logger}
	simStart := &SimStartHandler{tm: tmRepo, dtm: dtmRepo, udtm: udtmRepo, ingestSimURL: ingestSimURL, logger: logger}

	r.Route("/api/go/v1", func(r chi.Router) {
		// Existing Redis-backed endpoints
		r.Post("/get-telemetry", telemetry.GetTelemetry)
		r.Get("/chain-status", chains.GetChainStatus)
		r.Get("/chain-mismatches", chains.GetChainMismatches)
		r.Get("/limit-failures", limits.GetLimitFailures)
		r.Get("/simulator-status", simulator.GetSimulatorStatus)
		r.Post("/simulator/stop", simulator.StopLegacySimulator)
		r.Put("/udtm/values", udtm.PutValues)
		r.Put("/dtm/values", dtm.PutValues)

		// Monaco autocomplete — SQLite catalog reads
		r.Get("/mnemonics/tm", mnemonics.GetTMMnemonics)
		r.Get("/mnemonics/tm/id_to_mnemonic_mapping", mnemonics.GetTMParamIDMnemonicMapping)
		r.Get("/mnemonics/tm/{subsystem}", mnemonics.GetTMMnemonicsBySubsystem)
		r.Get("/get/mnemonics/tm", mnemonics.GetAllTMMnemonics)
		r.Get("/get/mnemonics/tm/{subsystem}", mnemonics.GetTMMnemonicsBySubsystemGET)
		r.Get("/get/mnemonics/tm/{subsystem}/{mnemonic}/range", mnemonics.GetMnemonicRange)
		r.Get("/mnemonics/tc", mnemonics.GetTCMnemonics)
		r.Get("/mnemonics/tc/{subsystem}", mnemonics.GetTCMnemonicsBySubsystem)
		r.Get("/telecommand/subsystems", mnemonics.GetTCSubsystems)
		r.Get("/telecommand/record", mnemonics.GetTCRecord)
		r.Get("/mnemonics/sco", mnemonics.GetSCOCommands)
		r.Get("/mnemonics/all", mnemonics.GetAllMnemonics)
		r.Get("/telemetry/subsystems", mnemonics.GetSubsystems)
		r.Get("/tm/mnemonics", mnemonics.GetLiveTMMnemonics)

		// TM API endpoints (Go v1)
		r.Post("/telemetry/upload", tmUpload.UploadTelemetry)
		r.Get("/telemetry/limits/{subsystem}", mnemonics.GetTMLimitsBySubsystem)
		r.Put("/telemetry/limits", mnemonics.UpdateTMLimits)
		r.Put("/telemetry/limits/bulk", mnemonics.UpdateTMLimitsBulk)
		r.Put("/telemetry/tolerance", mnemonics.UpdateTMTolerance)
		r.Put("/telemetry/expected-value", mnemonics.UpdateTMExpectedValue)
		r.Put("/telemetry/ignore-limit-check", mnemonics.UpdateTMIgnoreLimitCheck)
		r.Put("/telemetry/ignore-change-detection", mnemonics.UpdateTMIgnoreChangeDetection)
		r.Put("/telemetry/ignore-chain-comparision", mnemonics.UpdateTMIgnoreChainComparision)
		r.Put("/telemetry/available-chains", mnemonics.UpdateTMAvailableChains)

		// TC API endpoints (Go v1)
		r.Post("/telecommand/upload", tcUpload.UploadTelecommand)

		// UD_TM CRUD — SQLite-backed, versioned
		r.Get("/ud-tm", udtmCrud.GetUDTM)
		r.Post("/ud-tm", udtmCrud.SaveUDTM)
		r.Get("/ud-tm/versions", udtmCrud.GetUDTMVersions)
		r.Get("/ud-tm/versions/{version}", udtmCrud.GetUDTMVersion)

		// DTM procedure CRUD — SQLite-backed, publishes DTM_PROCEDURES_UPDATED
		r.Get("/dtm/procedures", dtmCrud.GetDTMProcedures)
		r.Post("/dtm/procedures", dtmCrud.SaveDTMProcedures)

		// Full Redis map reads — returns array of {param, value} dicts
		r.Get("/maps/{name}", maps.GetMap)

		// Generic Redis hash state storage API (GUI state persistence)
		r.Post("/redis/hash/write", redisHash.WriteHash)
		r.Post("/redis/hash/read", redisHash.ReadHash)

		// Spasdacs diagrams stored in SQLite "spasdacs" table
		r.Get("/diagrams", spasdacs.GetDiagrams)
		r.Get("/diagrams/{id}", spasdacs.GetDiagram)
		r.Post("/diagrams", spasdacs.PostDiagram)
		r.Patch("/diagrams/{id}", spasdacs.PatchDiagram)
		r.Delete("/diagrams/{id}", spasdacs.DeleteDiagram)

		// Sim API — value injection (FIXED mode / DTM / UDTM)
		r.Put("/sim/streams/{streamID}/values", simProxy.PutStreamValues)
		r.Get("/sim/streams", simProxy.GetStreams)
		r.Get("/sim/streams/{streamID}", simProxy.GetStream)

		// Simulation control — random simulation driven by range data from SQLite
		r.Post("/sim/streams/{streamID}/start", simStart.StartSim)
		r.Post("/sim/streams/{streamID}/stop", simStart.StopSim)
		r.Get("/sim/streams/{streamID}/status", simStart.GetSimStatus)
		r.Get("/sim/streams/status", simStart.GetAllSimStatus)
	})
}
