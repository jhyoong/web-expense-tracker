// internal/handlers/category_rules.go
package handlers

import (
    "encoding/json"
    "expense-tracker/internal/models"
    "net/http"
    "strconv"
    "strings"
    
    "github.com/gorilla/mux"
)

func (h *Handler) GetCategoryRules(w http.ResponseWriter, r *http.Request) {
    category := r.URL.Query().Get("category")
    
    query := `
        SELECT id, category, keyword, case_sensitive, created_at, updated_at
        FROM categorization_rules
        WHERE 1=1
    `
    args := []interface{}{}
    
    if category != "" {
        query += " AND category = ?"
        args = append(args, category)
    }
    
    query += " ORDER BY category, keyword"
    
    rows, err := h.db.Query(query, args...)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var rules []models.CategoryRule
    for rows.Next() {
        var r models.CategoryRule
        err := rows.Scan(&r.ID, &r.Category, &r.Keyword, &r.CaseSensitive, &r.CreatedAt, &r.UpdatedAt)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        rules = append(rules, r)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rules)
}

func (h *Handler) CreateCategoryRule(w http.ResponseWriter, r *http.Request) {
    var rule models.CategoryRule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Validate required fields
    if strings.TrimSpace(rule.Category) == "" {
        http.Error(w, "Category is required", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(rule.Keyword) == "" {
        http.Error(w, "Keyword is required", http.StatusBadRequest)
        return
    }
    
    query := `
        INSERT INTO categorization_rules (category, keyword, case_sensitive, created_at, updated_at)
        VALUES (?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    result, err := h.db.Exec(query, rule.Category, rule.Keyword, rule.CaseSensitive)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    rule.ID = int(id)
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusCreated)
    json.NewEncoder(w).Encode(rule)
}

func (h *Handler) UpdateCategoryRule(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "Invalid rule ID", http.StatusBadRequest)
        return
    }
    
    var rule models.CategoryRule
    if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
        http.Error(w, err.Error(), http.StatusBadRequest)
        return
    }
    
    // Validate required fields
    if strings.TrimSpace(rule.Category) == "" {
        http.Error(w, "Category is required", http.StatusBadRequest)
        return
    }
    if strings.TrimSpace(rule.Keyword) == "" {
        http.Error(w, "Keyword is required", http.StatusBadRequest)
        return
    }
    
    query := `
        UPDATE categorization_rules 
        SET category = ?, keyword = ?, case_sensitive = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    
    result, err := h.db.Exec(query, rule.Category, rule.Keyword, rule.CaseSensitive, id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    if rowsAffected == 0 {
        http.Error(w, "Rule not found", http.StatusNotFound)
        return
    }
    
    rule.ID = id
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(rule)
}

func (h *Handler) DeleteCategoryRule(w http.ResponseWriter, r *http.Request) {
    vars := mux.Vars(r)
    id, err := strconv.Atoi(vars["id"])
    if err != nil {
        http.Error(w, "Invalid rule ID", http.StatusBadRequest)
        return
    }
    
    query := "DELETE FROM categorization_rules WHERE id = ?"
    
    result, err := h.db.Exec(query, id)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    
    if rowsAffected == 0 {
        http.Error(w, "Rule not found", http.StatusNotFound)
        return
    }
    
    w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) GetCategories(w http.ResponseWriter, r *http.Request) {
    query := `
        SELECT DISTINCT category
        FROM categorization_rules
        ORDER BY category
    `
    
    rows, err := h.db.Query(query)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    defer rows.Close()
    
    var categories []string
    for rows.Next() {
        var category string
        if err := rows.Scan(&category); err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }
        categories = append(categories, category)
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(categories)
}