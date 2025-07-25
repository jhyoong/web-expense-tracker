// internal/handlers/import.go
package handlers

import (
	"encoding/csv"
	"encoding/json"
	"expense-tracker/internal/models"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (h *Handler) ImportFromCSV(w http.ResponseWriter, r *http.Request) {
	log.Printf("ImportFromCSV: Received %s request", r.Method)
	
	// Only allow POST requests
	if r.Method != http.MethodPost {
		log.Printf("ImportFromCSV: Method not allowed: %s", r.Method)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse multipart form with 10 MB limit
	log.Printf("ImportFromCSV: Parsing multipart form...")
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("ImportFromCSV: Failed to parse form: %v", err)
		http.Error(w, "Failed to parse form: file too large or invalid", http.StatusBadRequest)
		return
	}
	
	// Get the uploaded file
	log.Printf("ImportFromCSV: Getting uploaded file...")
	file, header, err := r.FormFile("csv")
	if err != nil {
		log.Printf("ImportFromCSV: Failed to get file: %v", err)
		http.Error(w, "No CSV file provided", http.StatusBadRequest)
		return
	}
	defer file.Close()
	
	log.Printf("ImportFromCSV: Received file: %s (%d bytes)", header.Filename, header.Size)
	
	// Validate file type
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".csv") {
		http.Error(w, "File must be a CSV", http.StatusBadRequest)
		return
	}
	
	// Parse the CSV
	expenses, err := h.parseCSV(file)
	if err != nil {
		log.Printf("ImportFromCSV: Failed to parse CSV: %v", err)
		http.Error(w, fmt.Sprintf("Failed to parse CSV: %v", err), http.StatusBadRequest)
		return
	}
	
	// Validate that we found some expenses
	if len(expenses) == 0 {
		http.Error(w, "No valid transactions found in CSV. Please check the file format.", http.StatusBadRequest)
		return
	}
	
	// Return parsed expenses for preview
	w.Header().Set("Content-Type", "application/json")
	response := map[string]interface{}{
		"success":  true,
		"expenses": expenses,
		"count":    len(expenses),
		"filename": header.Filename,
		"message":  fmt.Sprintf("Successfully parsed %d transactions", len(expenses)),
	}
	
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

func (h *Handler) parseCSV(file io.Reader) ([]models.Expense, error) {
	reader := csv.NewReader(file)
	
	// Read header row
	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header row: %v", err)
	}
	
	// Map header positions
	headerMap := make(map[string]int)
	for i, col := range header {
		headerMap[strings.ToUpper(strings.TrimSpace(col))] = i
	}
	
	// Validate required columns
	requiredCols := []string{"TRANSACTION_DATE", "DESCRIPTION", "AMOUNT"}
	for _, col := range requiredCols {
		if _, exists := headerMap[col]; !exists {
			return nil, fmt.Errorf("missing required column: %s", col)
		}
	}
	
	var expenses []models.Expense
	var errors []string
	lineNum := 1 // Start at 1 since we already read the header
	
	// Read data rows
	for {
		lineNum++
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: failed to read record: %v", lineNum, err))
			continue
		}
		
		// Skip empty rows
		if len(record) == 0 || (len(record) == 1 && strings.TrimSpace(record[0]) == "") {
			continue
		}
		
		// Parse expense from record
		expense, err := h.parseExpenseFromRecord(record, headerMap, lineNum)
		if err != nil {
			errors = append(errors, fmt.Sprintf("line %d: %v", lineNum, err))
			continue
		}
		
		expenses = append(expenses, expense)
	}
	
	// Return error if we have too many parsing errors
	if len(errors) > 0 && len(expenses) == 0 {
		return nil, fmt.Errorf("failed to parse any valid expenses. Errors: %s", strings.Join(errors, "; "))
	}
	
	// Log warnings if we had some errors but still got some valid expenses
	if len(errors) > 0 {
		log.Printf("ImportFromCSV: Parsed %d valid expenses with %d errors: %s", len(expenses), len(errors), strings.Join(errors, "; "))
	}
	
	return expenses, nil
}

func (h *Handler) parseExpenseFromRecord(record []string, headerMap map[string]int, lineNum int) (models.Expense, error) {
	var expense models.Expense
	
	// Parse required fields
	dateStr := strings.TrimSpace(h.getFieldValue(record, headerMap, "TRANSACTION_DATE"))
	if dateStr == "" {
		return expense, fmt.Errorf("empty transaction date")
	}
	
	// Try multiple date formats
	date, err := h.parseDate(dateStr)
	if err != nil {
		return expense, fmt.Errorf("invalid date format '%s': %v", dateStr, err)
	}
	expense.Date = date
	
	// Description (required)
	description := strings.TrimSpace(h.getFieldValue(record, headerMap, "DESCRIPTION"))
	if description == "" {
		return expense, fmt.Errorf("empty description")
	}
	expense.Description = description
	
	// Amount (required)
	amountStr := strings.TrimSpace(h.getFieldValue(record, headerMap, "AMOUNT"))
	if amountStr == "" {
		return expense, fmt.Errorf("empty amount")
	}
	
	amount, err := h.parseAmount(amountStr)
	if err != nil {
		return expense, fmt.Errorf("invalid amount '%s': %v", amountStr, err)
	}
	expense.Amount = amount
	
	// Optional fields
	location := strings.TrimSpace(h.getFieldValue(record, headerMap, "LOCATION"))
	if location != "" {
		expense.Vendor = location
	}
	
	creditCard := strings.TrimSpace(h.getFieldValue(record, headerMap, "CREDIT_CARD"))
	if creditCard != "" {
		expense.PaymentMethod = creditCard
	} else {
		expense.PaymentMethod = "CSV Import"
	}
	
	// Set defaults
	expense.Category = h.categorizeExpense(description)
	
	return expense, nil
}

func (h *Handler) getFieldValue(record []string, headerMap map[string]int, fieldName string) string {
	if pos, exists := headerMap[fieldName]; exists && pos < len(record) {
		return record[pos]
	}
	return ""
}

func (h *Handler) parseDate(dateStr string) (time.Time, error) {
	// Try common date formats
	formats := []string{
		"2006-01-02",           // YYYY-MM-DD
		"01/02/2006",           // MM/DD/YYYY
		"02/01/2006",           // DD/MM/YYYY
		"2006/01/02",           // YYYY/MM/DD
		"01-02-2006",           // MM-DD-YYYY
		"02-01-2006",           // DD-MM-YYYY
		"2006-01-02 15:04:05",  // YYYY-MM-DD HH:MM:SS
		"01/02/2006 15:04:05",  // MM/DD/YYYY HH:MM:SS
	}
	
	for _, format := range formats {
		if date, err := time.Parse(format, dateStr); err == nil {
			return date, nil
		}
	}
	
	return time.Time{}, fmt.Errorf("unsupported date format")
}

func (h *Handler) parseAmount(amountStr string) (float64, error) {
	// Remove common currency symbols and whitespace
	amountStr = strings.TrimSpace(amountStr)
	amountStr = strings.ReplaceAll(amountStr, "$", "")
	amountStr = strings.ReplaceAll(amountStr, ",", "")
	amountStr = strings.TrimSpace(amountStr)
	
	if amountStr == "" {
		return 0, fmt.Errorf("empty amount after cleaning")
	}
	
	// Parse the amount
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number format: %v", err)
	}
	
	// Validate reasonable range (allow negative amounts for refunds)
	if amount < -999999.99 {
		return 0, fmt.Errorf("amount too small (below -999999.99)")
	}
	
	if amount > 999999.99 {
		return 0, fmt.Errorf("amount too large")
	}
	
	return amount, nil
}

func (h *Handler) categorizeExpense(description string) string {
	// Query categorization rules from database
	query := `
		SELECT category, keyword, case_sensitive
		FROM categorization_rules
		ORDER BY category, keyword
	`
	
	rows, err := h.db.Query(query)
	if err != nil {
		// Fallback to "Other" if database query fails
		return "Other"
	}
	defer rows.Close()
	
	// Check each rule against the description
	for rows.Next() {
		var category, keyword string
		var caseSensitive bool
		
		if err := rows.Scan(&category, &keyword, &caseSensitive); err != nil {
			continue
		}
		
		// Prepare strings for comparison based on case sensitivity
		searchIn := description
		searchFor := keyword
		
		if !caseSensitive {
			searchIn = strings.ToUpper(description)
			searchFor = strings.ToUpper(keyword)
		}
		
		// Check if keyword matches
		if strings.Contains(searchIn, searchFor) {
			return category
		}
	}
	
	return "Other"
}