package domain

// ContactFilter holds filtering options for listing contacts.
type ContactFilter struct {
	Search   string
	Type     string // "company" or "individual"
	Favorite *bool  // nil = no filter
	Limit    int
	Offset   int
}
