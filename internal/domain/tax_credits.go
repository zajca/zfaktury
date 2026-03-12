package domain

import "time"

// TaxSpouseCredit represents the spouse tax credit for a given year.
// At most one record per year. Credit applies if spouse income < 68000 CZK.
type TaxSpouseCredit struct {
	ID               int64
	Year             int
	SpouseName       string
	SpouseBirthNumber string
	SpouseIncome     Amount // if < 6800000 halere -> credit applies
	SpouseZTP        bool   // ZTP/P holder -> doubled credit
	MonthsClaimed    int    // 1-12
	CreditAmount     Amount // computed by service
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// TaxChildCredit represents a child tax benefit entry for a given year.
// Multiple children per year are supported.
type TaxChildCredit struct {
	ID            int64
	Year          int
	ChildName     string
	BirthNumber   string
	ChildOrder    int    // 1, 2, 3 (3 = 3rd and subsequent)
	MonthsClaimed int    // 1-12
	ZTP           bool   // ZTP/P holder -> doubled benefit
	CreditAmount  Amount // computed by service
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TaxPersonalCredits represents personal tax credits (student, disability) for a year.
// At most one record per year.
type TaxPersonalCredits struct {
	Year             int
	IsStudent        bool
	StudentMonths    int    // 1-12
	DisabilityLevel  int    // 0=none, 1=1st/2nd degree, 2=3rd degree, 3=ZTP/P holder
	CreditStudent    Amount // computed by service
	CreditDisability Amount // computed by service
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
