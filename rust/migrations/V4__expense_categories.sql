CREATE TABLE expense_categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    key TEXT NOT NULL UNIQUE,
    label_cs TEXT NOT NULL,
    label_en TEXT NOT NULL,
    color TEXT NOT NULL DEFAULT '#6B7280',
    sort_order INTEGER NOT NULL DEFAULT 0,
    is_default INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    deleted_at TEXT
);
CREATE INDEX idx_expense_categories_key ON expense_categories(key);
CREATE INDEX idx_expense_categories_deleted_at ON expense_categories(deleted_at);

-- Insert 16 default categories for Czech OSVC
INSERT INTO expense_categories (key, label_cs, label_en, sort_order, is_default) VALUES
('office_supplies', 'Kancelarske potreby', 'Office supplies', 1, 1),
('software', 'Software a licence', 'Software & licenses', 2, 1),
('hardware', 'Hardware a technika', 'Hardware & equipment', 3, 1),
('telecom', 'Telefon a internet', 'Telecom & internet', 4, 1),
('travel', 'Cestovni naklady', 'Travel expenses', 5, 1),
('fuel', 'Pohonne hmoty', 'Fuel', 6, 1),
('vehicle', 'Provoz vozidla', 'Vehicle operation', 7, 1),
('rent', 'Najem', 'Rent', 8, 1),
('utilities', 'Energie', 'Utilities', 9, 1),
('education', 'Vzdelavani', 'Education & training', 10, 1),
('marketing', 'Marketing a reklama', 'Marketing & advertising', 11, 1),
('insurance', 'Pojisteni', 'Insurance', 12, 1),
('accounting', 'Uctovnictvi a dane', 'Accounting & taxes', 13, 1),
('postage', 'Postovne', 'Postage & shipping', 14, 1),
('services', 'Sluzby', 'Services', 15, 1),
('other', 'Ostatni', 'Other', 99, 1);
