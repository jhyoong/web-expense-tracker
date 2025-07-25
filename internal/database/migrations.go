// internal/database/migrations.go
package database

const createTablesSQL = `
CREATE TABLE IF NOT EXISTS expenses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    date DATE NOT NULL,
    category TEXT NOT NULL,
    description TEXT,
    amount DECIMAL(10,2) NOT NULL,
    vendor TEXT,
    payment_method TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categorization_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    category TEXT NOT NULL,
    keyword TEXT NOT NULL,
    case_sensitive BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_expenses_date ON expenses(date);
CREATE INDEX IF NOT EXISTS idx_expenses_category ON expenses(category);
CREATE INDEX IF NOT EXISTS idx_categorization_rules_category ON categorization_rules(category);
CREATE INDEX IF NOT EXISTS idx_categorization_rules_keyword ON categorization_rules(keyword);
`

const seedCategoryRulesSQL = `
INSERT OR IGNORE INTO categorization_rules (category, keyword, case_sensitive) VALUES
-- Transportation
('Transportation', 'BUS', false),
('Transportation', 'MRT', false),
('Transportation', 'GRAB', false),
('Transportation', 'TAXI', false),
('Transportation', 'TRANSPORT', false),

-- Food & Dining
('Food & Dining', 'MCDONALDS', false),
('Food & Dining', 'SUBWAY', false),
('Food & Dining', 'COFFEE', false),
('Food & Dining', 'RESTAURANT', false),
('Food & Dining', 'CAFE', false),
('Food & Dining', 'KITCHEN', false),
('Food & Dining', 'SUSHI', false),
('Food & Dining', 'RAMEN', false),
('Food & Dining', 'DINING', false),
('Food & Dining', 'FOOD', false),
('Food & Dining', 'MEAL', false),

-- Shopping
('Shopping', 'SHOPPING', false),
('Shopping', 'STORE', false),
('Shopping', 'MART', false),
('Shopping', 'RETAIL', false),
('Shopping', 'PURCHASE', false),

-- Utilities
('Utilities', 'UTILITIES', false),
('Utilities', 'ELECTRIC', false),
('Utilities', 'WATER', false),
('Utilities', 'GAS', false),

-- Healthcare
('Healthcare', 'PHARMACY', false),
('Healthcare', 'CLINIC', false),
('Healthcare', 'HOSPITAL', false),
('Healthcare', 'MEDICAL', false);
`