package restapi

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var m_client *mongo.Client

func getCol(vars map[string]string) *mongo.Collection {
	db := vars["db"]
	col := vars["col"]
	return m_client.Database(db).Collection(col)
}

// --- FIND ---
func findHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Filter     bson.M `json:"filter"`
		Sort       bson.M `json:"sort"`
		Limit      int64  `json:"limit"`
		Skip       int64  `json:"skip"`
		Projection bson.M `json:"projection"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts := options.Find().
		SetSort(req.Sort).
		SetLimit(req.Limit).
		SetSkip(req.Skip).
		SetProjection(req.Projection)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	cur, err := getCol(vars).Find(ctx, req.Filter, opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer cur.Close(ctx)
	var results []bson.M
	if err = cur.All(ctx, &results); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// --- INSERT ---
func insertHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var docs []bson.M
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&docs)
	if err != nil {
		// Try single document
		var singleDoc bson.M
		decoder = json.NewDecoder(r.Body) // reset decoder (rewind not possible, so in real API you'd buffer first)
		if err2 := decoder.Decode(&singleDoc); err2 != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		docs = append(docs, singleDoc)
	}

	// Convert []bson.M to []interface{}
	var docsAny []interface{}
	for _, doc := range docs {
		docsAny = append(docsAny, doc)
	}
	res, err := getCol(vars).InsertMany(context.Background(), docsAny)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res.InsertedIDs)
}

// --- UPDATE ---
func updateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Filter bson.M `json:"filter"`
		Update bson.M `json:"update"`
		Many   bool   `json:"many"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var res *mongo.UpdateResult
	var err error
	if req.Many {
		res, err = getCol(vars).UpdateMany(context.Background(), req.Filter, req.Update)
	} else {
		res, err = getCol(vars).UpdateOne(context.Background(), req.Filter, req.Update)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"matched":  res.MatchedCount,
		"modified": res.ModifiedCount,
	})
}

// --- DELETE ---
func deleteHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	var req struct {
		Filter bson.M `json:"filter"`
		Many   bool   `json:"many"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var res *mongo.DeleteResult
	var err error
	if req.Many {
		res, err = getCol(vars).DeleteMany(context.Background(), req.Filter)
	} else {
		res, err = getCol(vars).DeleteOne(context.Background(), req.Filter)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": res.DeletedCount,
	})
}

// --- CREATE COLLECTION ---
func createCollectionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	db := vars["db"]
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		http.Error(w, "Missing collection name", http.StatusBadRequest)
		return
	}
	err := m_client.Database(db).CreateCollection(context.Background(), req.Name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "created"})
}

// --- DROP COLLECTION ---
func dropCollectionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	db := vars["db"]
	col := vars["col"]
	err := m_client.Database(db).Collection(col).Drop(context.Background())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "dropped"})
}

func initMongoDb() bool {
	var err error
	m_client, err = mongo.Connect(context.Background(), options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		log.Println("MongoDB connection error:", err)
		return false
	}
	return true
}

func registerRoutesForMongoDb(r *mux.Router) {
	if initMongoDb() {
		r.HandleFunc("/{db}/{col}/find", findHandler).Methods("POST")
		r.HandleFunc("/{db}/{col}/insert", insertHandler).Methods("POST")
		r.HandleFunc("/{db}/{col}/update", updateHandler).Methods("POST")
		r.HandleFunc("/{db}/{col}/delete", deleteHandler).Methods("POST")
		r.HandleFunc("/{db}/create", createCollectionHandler).Methods("POST")
		r.HandleFunc("/{db}/drop/{col}", dropCollectionHandler).Methods("POST")
	} else {
		log.Fatal("Failed to initialize MongoDB connection")
	}
}
