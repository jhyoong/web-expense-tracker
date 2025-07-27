package repository

import (
    "expense-tracker/internal/database"
    "expense-tracker/internal/models"
)

type ExpenseRepository interface {
    GetAll(filter models.ExpenseFilter, page, limit int) ([]models.Expense, *PaginationInfo, error)
    GetByID(id int) (*models.Expense, error)
    Create(expense *models.Expense) error
    Update(id int, expense *models.Expense) error
    Delete(id int) error
    GetStats(startDate, endDate string) (map[string]interface{}, error)
    GetMonthlyStats(startDate, endDate string) (map[string]interface{}, error)
    BulkInsert(expenses []models.Expense) ([]models.Expense, error)
}

type PaginationInfo struct {
    Total       int  `json:"total"`
    Page        int  `json:"page"`
    Limit       int  `json:"limit"`
    HasNext     bool `json:"has_next"`
    HasPrevious bool `json:"has_previous"`
}

type expenseRepository struct {
    db *database.DB
}

func NewExpenseRepository(db *database.DB) ExpenseRepository {
    return &expenseRepository{db: db}
}

func (r *expenseRepository) GetAll(filter models.ExpenseFilter, page, limit int) ([]models.Expense, *PaginationInfo, error) {
    // Build query with filters
    query := `
        SELECT id, date, category, description, amount, vendor, payment_method, created_at, updated_at
        FROM expenses
        WHERE 1=1
    `
    args := []interface{}{}
    
    if !filter.StartDate.IsZero() {
        query += " AND date >= ?"
        args = append(args, filter.StartDate)
    }
    if !filter.EndDate.IsZero() {
        query += " AND date <= ?"
        args = append(args, filter.EndDate)
    }
    if filter.Category != "" {
        query += " AND category = ?"
        args = append(args, filter.Category)
    }
    
    // Count total for pagination
    countQuery := "SELECT COUNT(*) FROM expenses WHERE 1=1"
    countArgs := args // Use same filters for count
    if !filter.StartDate.IsZero() {
        countQuery += " AND date >= ?"
    }
    if !filter.EndDate.IsZero() {
        countQuery += " AND date <= ?"
    }
    if filter.Category != "" {
        countQuery += " AND category = ?"
    }
    
    var total int
    err := r.db.QueryRow(countQuery, countArgs...).Scan(&total)
    if err != nil {
        return nil, nil, err
    }
    
    // Add ordering and pagination
    query += " ORDER BY date DESC"
    if limit > 0 {
        offset := (page - 1) * limit
        query += " LIMIT ? OFFSET ?"
        args = append(args, limit, offset)
    }
    
    rows, err := r.db.Query(query, args...)
    if err != nil {
        return nil, nil, err
    }
    defer rows.Close()
    
    var expenses []models.Expense
    for rows.Next() {
        var e models.Expense
        err := rows.Scan(&e.ID, &e.Date, &e.Category, &e.Description, 
                        &e.Amount, &e.Vendor, &e.PaymentMethod, &e.CreatedAt, &e.UpdatedAt)
        if err != nil {
            return nil, nil, err
        }
        expenses = append(expenses, e)
    }
    
    // Build pagination info
    pagination := &PaginationInfo{
        Total:       total,
        Page:        page,
        Limit:       limit,
        HasNext:     page*limit < total,
        HasPrevious: page > 1,
    }
    
    return expenses, pagination, nil
}

func (r *expenseRepository) GetByID(id int) (*models.Expense, error) {
    query := `
        SELECT id, date, category, description, amount, vendor, payment_method, created_at, updated_at
        FROM expenses WHERE id = ?
    `
    
    var e models.Expense
    err := r.db.QueryRow(query, id).Scan(&e.ID, &e.Date, &e.Category, &e.Description,
                                          &e.Amount, &e.Vendor, &e.PaymentMethod, 
                                          &e.CreatedAt, &e.UpdatedAt)
    if err != nil {
        return nil, err
    }
    
    return &e, nil
}

func (r *expenseRepository) Create(expense *models.Expense) error {
    query := `
        INSERT INTO expenses (date, category, description, amount, vendor, payment_method, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    result, err := r.db.Exec(query, expense.Date, expense.Category, expense.Description, 
                           expense.Amount, expense.Vendor, expense.PaymentMethod)
    if err != nil {
        return err
    }
    
    id, err := result.LastInsertId()
    if err != nil {
        return err
    }
    
    expense.ID = int(id)
    return nil
}

func (r *expenseRepository) Update(id int, expense *models.Expense) error {
    query := `
        UPDATE expenses 
        SET date = ?, category = ?, description = ?, amount = ?, vendor = ?, payment_method = ?, updated_at = CURRENT_TIMESTAMP
        WHERE id = ?
    `
    
    result, err := r.db.Exec(query, expense.Date, expense.Category, expense.Description, 
                           expense.Amount, expense.Vendor, expense.PaymentMethod, id)
    if err != nil {
        return err
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    
    if rowsAffected == 0 {
        return ErrExpenseNotFound
    }
    
    expense.ID = id
    return nil
}

func (r *expenseRepository) Delete(id int) error {
    result, err := r.db.Exec("DELETE FROM expenses WHERE id = ?", id)
    if err != nil {
        return err
    }
    
    rowsAffected, err := result.RowsAffected()
    if err != nil {
        return err
    }
    
    if rowsAffected == 0 {
        return ErrExpenseNotFound
    }
    
    return nil
}

func (r *expenseRepository) GetStats(startDate, endDate string) (map[string]interface{}, error) {
    query := `
        SELECT category, SUM(amount) as total
        FROM expenses
        WHERE date BETWEEN ? AND ?
        GROUP BY category
        ORDER BY total DESC
    `
    
    rows, err := r.db.Query(query, startDate, endDate)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    stats := make(map[string]interface{})
    categories := make(map[string]float64)
    
    var totalAmount float64
    for rows.Next() {
        var category string
        var amount float64
        if err := rows.Scan(&category, &amount); err != nil {
            return nil, err
        }
        categories[category] = amount
        totalAmount += amount
    }
    
    stats["categories"] = categories
    stats["total"] = totalAmount
    
    return stats, nil
}

func (r *expenseRepository) GetMonthlyStats(startDate, endDate string) (map[string]interface{}, error) {
    query := `
        SELECT strftime('%Y-%m', date) as month, category, SUM(amount) as total
        FROM expenses
        WHERE date BETWEEN ? AND ?
        GROUP BY strftime('%Y-%m', date), category
        ORDER BY month ASC, category ASC
    `
    
    rows, err := r.db.Query(query, startDate, endDate)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    
    stats := make(map[string]interface{})
    monthlyData := make(map[string]map[string]float64)
    allCategories := make(map[string]bool)
    
    var totalAmount float64
    for rows.Next() {
        var month, category string
        var amount float64
        if err := rows.Scan(&month, &category, &amount); err != nil {
            return nil, err
        }
        
        if monthlyData[month] == nil {
            monthlyData[month] = make(map[string]float64)
        }
        monthlyData[month][category] = amount
        allCategories[category] = true
        totalAmount += amount
    }
    
    // Convert to array format for frontend
    monthlyArray := make([]map[string]interface{}, 0)
    for month, categories := range monthlyData {
        monthData := map[string]interface{}{
            "month":      month,
            "categories": categories,
        }
        
        // Calculate monthly total
        var monthTotal float64
        for _, amount := range categories {
            monthTotal += amount
        }
        monthData["total"] = monthTotal
        
        monthlyArray = append(monthlyArray, monthData)
    }
    
    // Sort by month using a more robust approach
    for i := 0; i < len(monthlyArray)-1; i++ {
        for j := i + 1; j < len(monthlyArray); j++ {
            month1, ok1 := monthlyArray[i]["month"].(string)
            month2, ok2 := monthlyArray[j]["month"].(string)
            if ok1 && ok2 && month1 > month2 {
                monthlyArray[i], monthlyArray[j] = monthlyArray[j], monthlyArray[i]
            }
        }
    }
    
    // Get list of all categories for consistent ordering
    categoryList := make([]string, 0, len(allCategories))
    for category := range allCategories {
        categoryList = append(categoryList, category)
    }
    
    stats["monthly"] = monthlyArray
    stats["categories"] = categoryList
    stats["total"] = totalAmount
    
    return stats, nil
}

func (r *expenseRepository) BulkInsert(expenses []models.Expense) ([]models.Expense, error) {
    tx, err := r.db.Begin()
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    query := `
        INSERT INTO expenses (date, category, description, amount, vendor, payment_method, created_at, updated_at)
        VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
    `
    
    stmt, err := tx.Prepare(query)
    if err != nil {
        return nil, err
    }
    defer stmt.Close()
    
    var savedExpenses []models.Expense
    
    for _, expense := range expenses {
        result, err := stmt.Exec(expense.Date, expense.Category, expense.Description, 
                               expense.Amount, expense.Vendor, expense.PaymentMethod)
        if err != nil {
            return nil, err
        }
        
        id, err := result.LastInsertId()
        if err != nil {
            return nil, err
        }
        
        expense.ID = int(id)
        savedExpenses = append(savedExpenses, expense)
    }
    
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    return savedExpenses, nil
}