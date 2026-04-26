package restapi

import (
	"fmt"
	"net/http"
	"path/filepath"
	shared "scg/shared"

	"github.com/gorilla/mux"
)

var UmacsEnvVariables shared.RedisUmacsEnvData

// Middleware to enable CORS
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow all origins (you can restrict this to specific origins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass down the request to the next handler
		next.ServeHTTP(w, r)
	})
}

func RestApiServer() {

	mainDist := shared.GetTomlConfigValue("paths.main_dist")
	spasdacsDist := shared.GetTomlConfigValue("paths.spasdacs_dist")

	// Convert to absolute path
	mainDist, err := filepath.Abs(mainDist)
	if err != nil {
		fmt.Println("Error converting to absolute path:", err)
		return
	}
	spasdacsDist, err = filepath.Abs(spasdacsDist)
	if err != nil {
		fmt.Println("Error converting to absolute path:", err)
		return
	}
	rdb = shared.GetRedisConnection()
	// Note: rdb.Close() is not deferred here because background goroutines need the connection
	// to remain open for the lifetime of the application
	UmacsEnvVariables = shared.ReadUmacsEnvData(rdb)

	r := mux.NewRouter()

	// Apply CORS middleware to the router
	r.Use(corsMiddleware)

	go pollForTcFileStatuses(ctx, rdb)
	// Start the payload state updater
	StartPayloadStateUpdater() // This already starts a goroutine internally
	// Start the test procedure queue processor
	StartTestProcedureQueueProcessor()
	// Serve static files from the "spasdacsDist" directory for the /spasdacs route
	fsSpasdacs := http.FileServer(http.Dir(spasdacsDist))
	r.PathPrefix("/spasdacs/").Handler(http.StripPrefix("/spasdacs", fsSpasdacs))

	registerRoutesForTm(r)
	registerRoutesForPriorityQueue(r)
	registerRoutesForWampPublisher(r)
	registerRoutesForRedisChannelValue(r)
	registerRoutesForRedisChannelValuePoling(r)
	registerRoutesForUmacsTcInterface(r)
	//registerRoutesForFileTransferSftp(r)
	registerRoutesForLogicalExpressionEvaluater(r)
	registerRoutesForApiDocs(r)
	//registerRoutesForArangoDb(r)
	registerRoutesForMongoDb(r)
	registerRoutesForGetPayloadState(r)
	// Serve static files from the "dist" directory
	fs := http.FileServer(http.Dir(mainDist))
	r.PathPrefix("/").Handler(fs)

	fmt.Println("Server started, listening on port 11000")
	if err := http.ListenAndServe(":11000", r); err != nil {
		fmt.Printf("Failed to start server: %v\n", err)
	}
}
