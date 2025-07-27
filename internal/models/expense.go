// internal/models/expense.go
package models

import (
    "time"
    "encoding/json"
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


// UnmarshalJSON handles custom date format from frontend
func (e *Expense) UnmarshalJSON(data []byte) error {
    type Alias Expense
    aux := &struct {
        Date string `json:"date"`
        *Alias
    }{
        Alias: (*Alias)(e),
    }
    
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }
    
    // Try parsing the date in the format sent from frontend (YYYY-MM-DD)
    if aux.Date != "" {
        parsedDate, err := time.Parse("2006-01-02", aux.Date)
        if err != nil {
            // Fallback to RFC3339 format in case it's already in that format
            parsedDate, err = time.Parse(time.RFC3339, aux.Date)
            if err != nil {
                return err
            }
        }
        e.Date = parsedDate
    }
    
    return nil
}