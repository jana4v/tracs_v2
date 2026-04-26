package wampApi

import (
	"context"
	"fmt"
	"log"

	"github.com/arangodb/go-driver"
	"github.com/arangodb/go-driver/http"
	"github.com/gammazero/nexus/v3/client"
	"github.com/gammazero/nexus/v3/wamp"
)

var dbClient driver.Client

// Initialize Database Connection
func initDB() bool {
	conn, err := http.NewConnection(http.ConnectionConfig{
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

// Ensure Database Exists
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

// Ensure Collection Exists
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
func formatResponse(data interface{}, err error) client.InvokeResult {
	response := map[string]interface{}{
		"data":  nil,
		"error": nil,
	}
	if err != nil {
		response["error"] = err.Error()
	} else {
		response["data"] = data
	}
	return client.InvokeResult{Args: wamp.List{response}}
}

// Upsert (Insert or Update) Document
func upsertDocument(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName, _ := params["db_name"].(string)
	collectionName, _ := params["collection_name"].(string)
	document, _ := params["document"].(map[string]interface{})

	if dbName == "" || collectionName == "" {
		return formatResponse(nil, fmt.Errorf("database and collection names are required"))
	}
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, err)
	}
	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		return formatResponse(nil, err)
	}

	if key, exists := document["_key"]; exists {
		if keyStr, ok := key.(string); ok {
			meta, err := col.UpdateDocument(ctx, keyStr, document)
			if err == nil {
				return formatResponse(fmt.Sprintf("Document updated (Rev: %s)", meta.Rev), nil)
			}
		}
	}

	meta, err := col.CreateDocument(ctx, document)
	if err != nil {
		return formatResponse(nil, err)
	}
	return formatResponse(fmt.Sprintf("Document inserted (Key: %s, Rev: %s)", meta.Key, meta.Rev), nil)
}

// Update Documents by Query
func updateDocumentsByQuery(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName := params["db_name"].(string)
	query := params["query"].(string)
	bindVars := params["bindvars"].(map[string]interface{})

	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("database error: %v", err))
	}

	_, err = database.Query(ctx, query, bindVars)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("update error: %v", err))
	}

	return formatResponse("Documents updated successfully", nil)
}

// Get Document by _key
func getDocument(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName, _ := params["db_name"].(string)
	collectionName, _ := params["collection_name"].(string)
	_key, _ := params["_key"].(string)

	if dbName == "" || collectionName == "" || _key == "" {
		return formatResponse(nil, fmt.Errorf("missing required parameters"))
	}
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, err)
	}
	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		return formatResponse(nil, err)
	}

	var document map[string]interface{}
	_, err = col.ReadDocument(ctx, _key, &document)
	if err != nil {
		return formatResponse(nil, err)
	}
	return formatResponse(document, nil)
}

// Get Multiple Documents
func getDocuments(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName := params["db_name"].(string)
	query := params["query"].(string)
	bindVars := params["bindvars"].(map[string]interface{})

	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("database error: %v", err))
	}

	cursor, err := database.Query(ctx, query, bindVars)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("query error: %v", err))
	}
	defer cursor.Close()

	var results []interface{}
	for {
		var doc interface{}
		_, err := cursor.ReadDocument(ctx, &doc)
		if driver.IsNoMoreDocuments(err) {
			break
		}
		results = append(results, doc)
	}

	return formatResponse(results, nil)
}

// Delete a single document by _key
func deleteDocument(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName, _ := params["db_name"].(string)
	collectionName, _ := params["collection_name"].(string)
	_key, keyExists := params["_key"].(string)

	if dbName == "" || collectionName == "" || !keyExists {
		return formatResponse(nil, fmt.Errorf("database name, collection name, and _key are required"))
	}
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, err)
	}
	col, err := ensureCollection(ctx, database, collectionName)
	if err != nil {
		return formatResponse(nil, err)
	}

	_, err = col.RemoveDocument(ctx, _key)
	if err != nil {
		return formatResponse(nil, err)
	}
	return formatResponse("Document deleted successfully", nil)
}

// Delete multiple documents based on a query
func deleteDocumentsByQuery(ctx context.Context, inv *wamp.Invocation) client.InvokeResult {
	params := inv.Arguments[0].(map[string]interface{})
	dbName, _ := params["db_name"].(string)
	query, queryExists := params["query"].(string)
	bindVars, _ := params["bindvars"].(map[string]interface{})

	if dbName == "" || !queryExists {
		return formatResponse(nil, fmt.Errorf("database name and query are required"))
	}
	database, err := ensureDatabase(ctx, dbName)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("database error: %v", err))
	}

	_, err = database.Query(ctx, query, bindVars)
	if err != nil {
		return formatResponse(nil, fmt.Errorf("delete error: %v", err))
	}

	return formatResponse("Documents deleted successfully", nil)
}

func RegisternosqlDbProcedures(wampArangoClient *client.Client) {
	var isConnected bool = initDB()
	if isConnected {

		// Register all database operations
		wampArangoClient.Register("scg.nosqlDb.create_or_update_document", upsertDocument, nil)
		wampArangoClient.Register("scg.nosqlDb.read_document", getDocument, nil)
		wampArangoClient.Register("scg.nosqlDb.read_documents", getDocuments, nil)
		wampArangoClient.Register("scg.nosqlDb.update_documents", updateDocumentsByQuery, nil)
		wampArangoClient.Register("scg.nosqlDb.delete_document", deleteDocument, nil)
		wampArangoClient.Register("scg.nosqlDb.delete_documents", deleteDocumentsByQuery, nil)
	}
}
