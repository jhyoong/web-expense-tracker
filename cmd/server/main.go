// cmd/server/main.go
package main

import (
	"expense-tracker/internal/database"
	"expense-tracker/internal/handlers"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func main() {
	db, err := database.New("./expenses.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	h := handlers.New(db)
	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api").Subrouter()
	api.HandleFunc("/expenses", h.GetExpenses).Methods("GET")
	api.HandleFunc("/expenses", h.CreateExpense).Methods("POST")
	api.HandleFunc("/expenses/stats", h.GetStats).Methods("GET")
	api.HandleFunc("/import/csv", h.ImportFromCSV).Methods("POST")
	api.HandleFunc("/import/confirm", h.ConfirmImport).Methods("POST")
	
	// Category rules routes
	api.HandleFunc("/categorization-rules", h.GetCategoryRules).Methods("GET")
	api.HandleFunc("/categorization-rules", h.CreateCategoryRule).Methods("POST")
	api.HandleFunc("/categorization-rules/{id}", h.UpdateCategoryRule).Methods("PUT")
	api.HandleFunc("/categorization-rules/{id}", h.DeleteCategoryRule).Methods("DELETE")
	api.HandleFunc("/categories", h.GetCategories).Methods("GET")

	// Static files
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static/"))))
	r.HandleFunc("/", h.IndexPage).Methods("GET")

	log.Println("Server starting on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe("0.0.0.0:8080", r))
}
