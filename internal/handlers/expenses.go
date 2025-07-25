// internal/handlers/expenses.go
package handlers

import (
    "encoding/json"
    "expense-tracker/internal/database"
    "expense-tracker/internal/models"
    "net/http"
)

type Handler struct {
    db *database.DB
}

func New(db *database.DB) *Handler {
    return &Handler{db: db}
}

func (h *Handler) GetExpenses(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    category := r.URL.Query().Get("category")
    
    query := `
        SELECT id, date, category, description, amount, vendor, payment_method, created_at, updated_at
        FROM expenses
        WHERE 1=1
    `
    args := []interface{}{}
    
    if startDate != "" {
        query += " AND date >= ?"
        args = append(args, startDate)
    }
    if endDate != "" {
        query += " AND date <= ?"
        args = append(args, endDate)
    }
    if category != "" {
        query += " AND category = ?"
        args = append(args, category)
    }
    
    query += " ORDER BY date DESC"
    
    rows, err := h.db.Query(query, args...)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var expenses []models.Expense
    for rows.Next() {
        var e models.Expense
        err := rows.Scan(&e.ID, &e.Date, &e.Category, &e.Description, 
                        &e.Amount, &e.Vendor, &e.PaymentMethod, &e.CreatedAt, &e.UpdatedAt)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        expenses = append(expenses, e)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(expenses)
}

func (h *Handler) GetStats(w http.ResponseWriter, r *http.Request) {
    startDate := r.URL.Query().Get("start_date")
    endDate := r.URL.Query().Get("end_date")
    
    // Category breakdown
    query := `
        SELECT category, SUM(amount) as total
        FROM expenses
        WHERE date BETWEEN ? AND ?
        GROUP BY category
        ORDER BY total DESC
    `
    
    rows, err := h.db.Query(query, startDate, endDate)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    stats := make(map[string]interface{})
    categories := make(map[string]float64)
    
    var totalAmount float64
    for rows.Next() {
        var category string
        var amount float64
        if err := rows.Scan(&category, &amount); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        categories[category] = amount
        totalAmount += amount
    }
    
    stats["categories"] = categories
    stats["total"] = totalAmount
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(stats)
}

func (h *Handler) CreateExpense(w http.ResponseWriter, r *http.Request) {
    var expense models.Expense
    if err := json.NewDecoder(r.Body).Decode(&expense); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    query := `
        INSERT INTO expenses (date, category, description, amount, vendor, payment_method, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    result, err := h.db.Exec(query, expense.Date, expense.Category, expense.Description, 
                           expense.Amount, expense.Vendor, expense.PaymentMethod)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    expense.ID = int(id)
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
    
    // Begin transaction for bulk insert
    tx, err := h.db.Begin()
    if err != nil {
        http.Error(w, "Failed to begin transaction: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer tx.Rollback()
    
    query := `
        INSERT INTO expenses (date, category, description, amount, vendor, payment_method, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    stmt, err := tx.Prepare(query)
    if err != nil {
        http.Error(w, "Failed to prepare statement: "+err.Error(), http.StatusInternalServerError)
        return
    }
    defer stmt.Close()
    
    var savedExpenses []models.Expense
    var totalAmount float64
    
    for _, expense := range expenses {
        result, err := stmt.Exec(expense.Date, expense.Category, expense.Description, 
                               expense.Amount, expense.Vendor, expense.PaymentMethod)
        if err != nil {
            http.Error(w, "Failed to insert expense: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        id, err := result.LastInsertId()
        if err != nil {
            http.Error(w, "Failed to get inserted ID: "+err.Error(), http.StatusInternalServerError)
            return
        }
        
        expense.ID = int(id)
        savedExpenses = append(savedExpenses, expense)
        totalAmount += expense.Amount
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        http.Error(w, "Failed to commit transaction: "+err.Error(), http.StatusInternalServerError)
        return
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

func (h *Handler) IndexPage(w http.ResponseWriter, r *http.Request) {
    // TODO: Serve the index.html template
    http.ServeFile(w, r, "./web/templates/index.html")
}