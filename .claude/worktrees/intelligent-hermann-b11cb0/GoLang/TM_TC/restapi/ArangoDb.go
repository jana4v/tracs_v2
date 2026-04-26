package restapi

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/arangodb/go-driver"
	ahttp "github.com/arangodb/go-driver/http"
	"github.com/gorilla/mux"
)

var dbClient driver.Client

func initDB() bool {
	conn, err := ahttp.NewConnection(ahttp.ConnectionConfig{
		Endpoints: []string{"http://localhost:8529"},
	})
	if err != nil {
		log.Fatalf("Failed to create DB connection: %v", err)
		return false
	}
	dbClient, err = driver.NewClient(driver.ClientConfig{
		Connection:     conn,
		Authentication: driver.BasicAuthentication("root", "qazwsxedc"),
	})
	if err != nil {
		log.Fatalf("Failed to create DB client: %v", err)
		return false
	}
	return true
}

func ensureDatabase(ctx context.Context, dbName string) (driver.Database, error) {
	database, err := dbClient.Database(ctx, dbName)
	if err == nil {
		return database, nil
	}
	if driver.IsNotFoundGeneral(err) {
		return dbClient.CreateDatabase(ctx, dbName, nil)
	}
	return nil, err
}

func ensureCollection(ctx context.Context, db driver.Database, collectionName string) (driver.Collection, error) {
	collection, err := db.Collection(ctx, collectionName)
	if err == nil {
		return collection, nil
	}
	if driver.IsNotFoundGeneral(err) {
		return db.CreateCollection(ctx, collectionName, nil)
	}
	return nil, err
}

func upsertDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)
	ctx := context.Background()
	dbName := req["db_name"].(string)
	collectionName := req["collection_name"].(string)
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Collection error: %v", err), http.StatusInternalServerError)
		return
	}
	document := req["document"].(map[string]interface{})

	if key, exists := document["_key"]; exists {
		if keyStr, ok := key.(string); ok {
			_, err := col.UpdateDocument(ctx, keyStr, document)
			if err == nil {
				w.Write([]byte("Document updated"))
				return
			}
		}
	}

	_, err = col.CreateDocument(ctx, document)
	if err != nil {
		http.Error(w, fmt.Sprintf("Insert error: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("Document inserted"))
}

func getDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)
	ctx := context.Background()
	dbName := req["db_name"].(string)
	collectionName := req["collection_name"].(string)
	_key := req["_key"].(string)
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Collection error: %v", err), http.StatusInternalServerError)
		return
	}

	var document map[string]interface{}
	_, err = col.ReadDocument(ctx, _key, &document)
	if err != nil {
		http.Error(w, "Document not found", http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(document)
}
func getDocumentsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		DBName   string                 `json:"db_name"`
		Query    string                 `json:"query"`
		BindVars map[string]interface{} `json:"bindvars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	database, err := ensureDatabase(ctx, params.DBName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	cursor, err := database.Query(ctx, params.Query, params.BindVars)
	if err != nil {
		http.Error(w, fmt.Sprintf("Query error: %v", err), http.StatusInternalServerError)
		return
	}
	defer cursor.Close()

	// Generic slice to store results
	var results []interface{}

	for {
		var doc interface{} // Use interface{} to handle any type
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		}
		results = append(results, doc)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func deleteDocumentHandler(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)
	ctx := context.Background()
	dbName := req["db_name"].(string)
	collectionName := req["collection_name"].(string)
	_key := req["_key"].(string)
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Collection error: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = col.RemoveDocument(ctx, _key)
	if err != nil {
		http.Error(w, "Delete error", http.StatusInternalServerError)
		return
	}

	w.Write([]byte("Document deleted successfully"))
}

// Update Documents by Query (REST API)
func updateDocumentsByQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		DBName   string                 `json:"db_name"`
		Query    string                 `json:"query"`
		BindVars map[string]interface{} `json:"bindvars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	database, err := ensureDatabase(ctx, params.DBName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = database.Query(ctx, params.Query, params.BindVars)
	if err != nil {
		http.Error(w, fmt.Sprintf("Update error: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"message": "Documents updated successfully"}`))
}

func deleteDocumentsByQueryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	var params struct {
		DBName   string                 `json:"db_name"`
		Query    string                 `json:"query"`
		BindVars map[string]interface{} `json:"bindvars"`
	}

	if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}

	if params.DBName == "" || params.Query == "" {
		http.Error(w, "Error: Database name and query are required", http.StatusBadRequest)
		return
	}

	ctx := context.Background()
	database, err := ensureDatabase(ctx, params.DBName)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = database.Query(ctx, params.Query, params.BindVars)
	if err != nil {
		http.Error(w, fmt.Sprintf("Delete error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"message": "Documents deleted successfully"})
}

func registerRoutesForArangoDb(r *mux.Router) {
	if initDB() {
		r.HandleFunc("/nosqlDb/upsert", upsertDocumentHandler).Methods("POST")
		r.HandleFunc("/nosqlDb/get", getDocumentHandler).Methods("POST")
		r.HandleFunc("/nosqlDb/get-documents", getDocumentsHandler).Methods("POST")
		r.HandleFunc("/nosqlDb/delete", deleteDocumentHandler).Methods("POST")
		r.HandleFunc("/nosqlDb/delete-query", deleteDocumentsByQueryHandler).Methods("POST")
		r.HandleFunc("/nosqlDb/update-query", updateDocumentsByQueryHandler).Methods("POST")

	} else {
		log.Fatal("Failed to initialize database connection")
	}

}
