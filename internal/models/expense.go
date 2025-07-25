// internal/models/expense.go
package models

import (
    "time"
)

type Expense struct {
    ID            int       `json:"id"`
    Date          time.Time `json:"date"`
    Category      string    `json:"category"`
    Description   string    `json:"description"`
    Amount        float64   `json:"amount"`
    Vendor        string    `json:"vendor"`
    PaymentMethod string    `json:"payment_method"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type ExpenseFilter struct {
    StartDate time.Time
    EndDate   time.Time
    Category  string
}