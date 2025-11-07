package case_omit

import "time"

// insert has an ID field with shift:"omit" tag, so it should use RETURNING id
type insert struct {
	ID          int64 `shift:"omit"` // ID is omitted, so HasID should be false
	Name        string
	DateOfBirth time.Time `shift:"dob"`
	InternalID  string    `shift:"omit"` // This field should be omitted from INSERT
}

// update has a field with shift:"omit" tag that should be excluded
type update struct {
	ID        int64
	Name      string
	InternalID string `shift:"omit"` // This field should be omitted from UPDATE
	Amount    int64
}

// complete has no omitted fields
type complete struct {
	ID int64
}

