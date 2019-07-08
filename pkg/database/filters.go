package database

// Filters represent the conditions given to a 'SELECT' sql request to filter
// the results of the request.
type Filters struct {
	// Limit fixes the maximum number of records selected by the request
	Limit int
	// Offset fixes the starting point from which candidate records will be selected
	Offset int
	// Order specifies the name of the column(s) used for ordering the answers,
	// followed by the direction (asc or desc).
	Order string
	// Conditions specifies all the conditions used for filtering the answers in
	// SQL format. Parameters can be given to the request using the '?' character.
	Conditions string
	// Args contains all the parameters declared in the 'Conditions' string. When
	// replacing all the '?' with these parameters, it should produce a valid
	// SQL 'WHERE' clause.
	Args []interface{}
}
