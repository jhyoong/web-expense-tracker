// internal/models/category_rule.go
package models

import (
    "time"
)

type CategoryRule struct {
    ID            int       `json:"id"`
    Category      string    `json:"category"`
    Keyword       string    `json:"keyword"`
    CaseSensitive bool      `json:"case_sensitive"`
    CreatedAt     time.Time `json:"created_at"`
    UpdatedAt     time.Time `json:"updated_at"`
}

type CategoryRuleFilter struct {
    Category string
}