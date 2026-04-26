package restapi

import (
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
)

// serveTemplate serves the documentation HTML templates.
func serveTemplate(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	tmpl := vars["template"]
	t, err := template.ParseFiles("templates/" + tmpl + ".html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

// registerRoutesForLogicalExpressionEvaluator registers routes for the API and documentation.
func registerRoutesForApiDocs(r *mux.Router) {
	r.HandleFunc("/docs/{template}", serveTemplate).Methods("GET")
	r.HandleFunc("/docs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "templates/index.html")
	}).Methods("GET")
}
