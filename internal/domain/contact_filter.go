package domain

// ContactFilter holds filtering options for listing contacts.
type ContactFilter struct {
	Search   string `json:"search"`
	Type     string `json:"type"`     // "company" or "individual"
	Favorite *bool  `json:"favorite"` // nil = no filter
	Limit    int    `json:"limit"`
	Offset   int    `json:"offset"`
}
