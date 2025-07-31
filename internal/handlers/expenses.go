// internal/handlers/expenses.go
package handlers

import (
    "encoding/json"
    "expense-tracker/internal/database"
    "expense-tracker/internal/models"
    "expense-tracker/internal/repository"
    "net/http"
    "strconv"
    "time"
    
    "github.com/gorilla/mux"
)

type Handler struct {
    db         *database.DB
    expenseRepo repository.ExpenseRepository
}

func New(db *database.DB) *Handler {
    return &Handler{
        db:         db,
        expenseRepo: repository.NewExpenseRepository(db),
    }
}

func (h *Handler) GetExpenses(w http.ResponseWriter, r *http.Request) {
    // Parse filters
    var filter models.ExpenseFilter
    
    if startDateStr := r.URL.Query().Get("start_date"); startDateStr != "" {
        if parsedDate, err := time.Parse("2006-01-02", startDateStr); err == nil {
            filter.StartDate = parsedDate
        }
    }
    
    if endDateStr := r.URL.Query().Get("end_date"); endDateStr != "" {
        if parsedDate, err := time.Parse("2006-01-02", endDateStr); err == nil {
            filter.EndDate = parsedDate
        }
    }
    
    filter.Category = r.URL.Query().Get("category")
    
    // Parse pagination
    page := 1
    if pageStr := r.URL.Query().Get("page"); pageStr != "" {
        if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
            page = p
        }
    }
    
    limit := 20 // Default page size
    if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
        if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
            limit = l
        }
    }
    
    expenses, pagination, err := h.expenseRepo.GetAll(filter, page, limit)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    response := map[string]interface{}{
        "expenses":   expenses,
        "pagination": pagination,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    category := r.URL.Query().Get("category")
    
    stats, err := h.expenseRepo.GetStats(startDate, endDate, category)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (h *Handler) GetMonthlyStats(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    category := r.URL.Query().Get("category")
    
    stats, err := h.expenseRepo.GetMonthlyStats(startDate, endDate, category)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
    var expense models.Expense
    if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    if err := h.expenseRepo.Create(&expense); err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(expense)
}

func (h *Handler) ConfirmImport(w http.ResponseWriter, r *http.Request) {
    if r.Method != http.MethodPost {
        http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
        return
    }
    
    var expenses []models.Expense
    if err := json.NewDecoder(r.Body).Decode(&expenses); err != nil {
        http.Error(w, "Invalid JSON data: "+err.Error(), http.StatusBadRequest)
        return
    }
    
    if len(expenses) == 0 {
        http.Error(w, "No expenses to import", http.StatusBadRequest)
        return
    }
    
    savedExpenses, err := h.expenseRepo.BulkInsert(expenses)
    if err != nil {
        http.Error(w, "Failed to import expenses: "+err.Error(), http.StatusInternalServerError)
        return
    }
    
    var totalAmount float64
    for _, expense := range savedExpenses {
        totalAmount += expense.Amount
    }
    
    response := map[string]interface{}{
        "success": true,
        "message": "Successfully imported expenses to database",
        "count":   len(savedExpenses),
        "total":   totalAmount,
        "expenses": savedExpenses,
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(response)
}

func (h *Handler) UpdateExpense(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    idStr := vars["id"]
    
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid expense ID", http.StatusBadRequest)
        return
    }
    
    var expense models.Expense
    if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    if err := h.expenseRepo.Update(id, &expense); err != nil {
        if err == repository.ErrExpenseNotFound {
            http.Error(w, "Expense not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    // Return updated expense
    updatedExpense, err := h.expenseRepo.GetByID(id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(updatedExpense)
}

func (h *Handler) DeleteExpense(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    idStr := vars["id"]
    
    id, err := strconv.Atoi(idStr)
    if err != nil {
        http.Error(w, "Invalid expense ID", http.StatusBadRequest)
        return
    }
    
    if err := h.expenseRepo.Delete(id); err != nil {
        if err == repository.ErrExpenseNotFound {
            http.Error(w, "Expense not found", http.StatusNotFound)
            return
        }
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) IndexPage(w http.ResponseWriter, r *http.Request) {
    // TODO: Serve the index.html template
    http.ServeFile(w, r, "./web/templates/index.html")
}